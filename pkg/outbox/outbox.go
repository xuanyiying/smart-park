// Package outbox implements the transactional outbox pattern.
//
// A service that must update its own database and notify other services cannot
// do both atomically with an HTTP or gRPC call: a crash between the commit and
// the call loses the event forever, and a call before the commit announces an
// event that never happened. The outbox removes that gap: the event is written
// to a local table inside the same transaction as the business change, and a
// background dispatcher later delivers it with retries. Delivery ends up
// at-least-once, so handlers must be idempotent — which the notification and
// gate-opening consumers already are by design.
//
// This is the answer to audit item H7. A distributed-transaction coordinator
// (Seata) was evaluated and rejected: the billing→payment→gate chain is already
// eventually consistent (conditional-update state machine, sweep reconciliation,
// daily settlement checks), the Go SDK is still incubating, its AT data-source
// proxy does not support PostgreSQL, and a TC server would add an operational
// dependency without fixing a failure mode the sweeps do not already cover.
// The outbox covers the remaining gap — an atomic hand-off between a local
// transaction and an outbound side effect.
//
// The package is storage-agnostic at its core (see Store) and ships a
// PostgreSQL implementation that claims messages with FOR UPDATE SKIP LOCKED so
// several replicas can dispatch concurrently without double-claiming.
package outbox

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Message statuses.
const (
	StatusPending    = "pending"
	StatusInFlight   = "in_flight"
	StatusDispatched = "dispatched"
	StatusDead       = "dead"
)

// Message is one unit of work produced by a business transaction.
type Message struct {
	ID uuid.UUID
	// AggregateType and AggregateID identify what produced the event, so a
	// handler (and a human debugging a dead message) can trace it back: for
	// example "order" and the order id behind an "order.settled" event.
	AggregateType string
	AggregateID   string
	Type          string
	// Payload carries the event body, typically JSON. The outbox never inspects
	// it; the producer and consumer agree on the schema.
	Payload []byte
	// Headers carry routing metadata (content type, tenant id, trace id).
	Headers map[string]string
	// AvailableAt delays first delivery, e.g. to give a downstream service time
	// to observe the committed row.
	AvailableAt time.Time
	CreatedAt   time.Time
	Attempts    int
	LastError   string
}

// Validate reports whether the message carries the minimum required fields.
func (m *Message) Validate() error {
	switch {
	case m == nil:
		return errors.New("outbox: message is nil")
	case m.Type == "":
		return errors.New("outbox: message type is required")
	case len(m.Payload) == 0:
		return errors.New("outbox: message payload is required")
	default:
		return nil
	}
}

// Execer is the database capability the outbox needs to append a message.
//
// *sql.DB and *sql.Tx satisfy it directly, and an Ent transaction does too
// (dialect.Tx exposes the same methods), which is what lets a producer append
// the event inside the very transaction that commits the business change:
//
//	database.WithTx(ctx, func(ctx context.Context) error {
//	    if err := saveOrder(ctx, order); err != nil { return err }
//	    return outbox.Enqueue(ctx, txFromCtx(ctx), msg)
//	})
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Enqueue appends messages through the given handle. When tx is nil the store
// uses its own connection pool; when it is set the rows commit or roll back
// with the caller's transaction — that pairing is the whole point of the
// pattern.
func Enqueue(ctx context.Context, store Store, tx Execer, msgs ...Message) error {
	return store.Enqueue(ctx, tx, msgs...)
}

// Store persists and claims outbox messages.
//
// The PostgreSQL implementation is the production one; tests substitute fakes.
type Store interface {
	// Enqueue appends messages. When tx is nil the store uses its own handle;
	// when it is set the rows commit or roll back with the caller's transaction.
	Enqueue(ctx context.Context, tx Execer, msgs ...Message) error
	// Claim atomically moves up to limit due pending messages to in-flight and
	// returns them. Concurrent dispatchers never receive the same message.
	Claim(ctx context.Context, limit int) ([]*Message, error)
	// MarkDispatched records a successful delivery.
	MarkDispatched(ctx context.Context, id uuid.UUID) error
	// MarkFailed records a failed delivery. dead selects the dead-letter state
	// instead of scheduling another retry.
	MarkFailed(ctx context.Context, id uuid.UUID, lastErr error, retryAt time.Time, dead bool) error
}

// NewMessage builds a message with sane defaults (fresh id, immediate
// availability).
func NewMessage(aggregateType, aggregateID, eventType string, payload []byte, headers map[string]string) Message {
	now := time.Now()
	return Message{
		ID:            uuid.New(),
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		Type:          eventType,
		Payload:       payload,
		Headers:       headers,
		AvailableAt:   now,
		CreatedAt:     now,
	}
}

// Handler delivers one claimed message. Returning nil marks the message
// dispatched; returning an error schedules a retry.
type Handler func(ctx context.Context, msg *Message) error

// DispatcherConfig tunes the polling loop.
type DispatcherConfig struct {
	// PollInterval waits between sweeps when the previous sweep found nothing.
	PollInterval time.Duration
	// BatchSize caps how many messages one sweep claims.
	BatchSize int
	// MaxAttempts caps retries; after it a message lands in the dead-letter
	// state for operator review instead of retrying forever.
	MaxAttempts int
	// InitialBackoff and MaxBackoff bound the exponential retry delay
	// (backoff = InitialBackoff << (attempts-1), capped at MaxBackoff).
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

func (c *DispatcherConfig) applyDefaults() {
	if c.PollInterval <= 0 {
		c.PollInterval = time.Second
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 32
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 10
	}
	if c.InitialBackoff <= 0 {
		c.InitialBackoff = 500 * time.Millisecond
	}
	if c.MaxBackoff <= 0 {
		c.MaxBackoff = time.Minute
	}
}

// retryDelay computes the next delivery delay after a failure. attempts is the
// number of attempts already made for the message.
func retryDelay(cfg DispatcherConfig, attempts int) time.Duration {
	shift := attempts - 1
	if shift < 0 {
		shift = 0
	}
	backoff := cfg.InitialBackoff << uint(shift)
	if backoff > cfg.MaxBackoff || backoff <= 0 {
		return cfg.MaxBackoff
	}
	return backoff
}

// Dispatcher polls the store and hands claimed messages to the handler.
type Dispatcher struct {
	store     Store
	handler   Handler
	cfg       DispatcherConfig
	onFailure func(msg *Message, err error) // optional hook for metrics/tests
}

// NewDispatcher builds a dispatcher. A nil onFailure hook is fine.
func NewDispatcher(store Store, handler Handler, cfg DispatcherConfig, onFailure func(*Message, error)) *Dispatcher {
	cfg.applyDefaults()
	return &Dispatcher{
		store:     store,
		handler:   handler,
		cfg:       cfg,
		onFailure: onFailure,
	}
}

// Run sweeps until ctx is cancelled. It is intended to run in its own goroutine
// next to the service's other background loops:
//
//	go dispatcher.Run(ctx)
func (d *Dispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(d.cfg.PollInterval)
	defer ticker.Stop()

	for {
		d.sweep(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// sweep claims and delivers one batch. Errors inside a sweep are surfaced
// through the failure hook and never stop the loop: a broken handler must not
// take down the process that also serves live traffic.
func (d *Dispatcher) sweep(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	messages, err := d.store.Claim(ctx, d.cfg.BatchSize)
	if err != nil {
		if d.onFailure != nil {
			d.onFailure(nil, fmt.Errorf("outbox: claim: %w", err))
		}
		return
	}

	for _, msg := range messages {
		d.deliver(ctx, msg)
	}
}

// deliver runs the handler for one message and records the outcome.
func (d *Dispatcher) deliver(ctx context.Context, msg *Message) {
	err := d.handler(ctx, msg)
	if err == nil {
		if markErr := d.store.MarkDispatched(ctx, msg.ID); markErr != nil && d.onFailure != nil {
			d.onFailure(msg, fmt.Errorf("outbox: mark dispatched: %w", markErr))
		}
		return
	}

	// Exponential backoff until the message exhausts its budget, then dead-letter.
	dead := msg.Attempts >= d.cfg.MaxAttempts
	retryAt := time.Now().Add(retryDelay(d.cfg, msg.Attempts))
	if dead {
		retryAt = time.Time{}
	}
	if markErr := d.store.MarkFailed(ctx, msg.ID, err, retryAt, dead); markErr != nil && d.onFailure != nil {
		d.onFailure(msg, fmt.Errorf("outbox: mark failed: %w", markErr))
	}
	if d.onFailure != nil {
		d.onFailure(msg, err)
	}
}
