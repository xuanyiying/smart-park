package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PostgresStore is the production Store backed by a PostgreSQL table.
//
// The table is created on demand by EnsureSchema so a service can adopt the
// outbox without an extra migration tool step; the statement is idempotent.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore wraps an open handle.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// EnsureSchema creates the outbox table and its indexes when missing.
//
// The claim path filters on (status, available_at) and orders by created_at, so
// that composite index keeps sweeps cheap even when the table grows large with
// dispatched history.
func (s *PostgresStore) EnsureSchema(ctx context.Context) error {
	const schema = `
CREATE TABLE IF NOT EXISTS outbox_messages (
    id             UUID PRIMARY KEY,
    aggregate_type TEXT NOT NULL,
    aggregate_id   TEXT NOT NULL,
    type           TEXT NOT NULL,
    payload        BYTEA NOT NULL,
    headers        JSONB NOT NULL DEFAULT '{}'::jsonb,
    status         TEXT NOT NULL DEFAULT 'pending',
    attempts       INT  NOT NULL DEFAULT 0,
    last_error     TEXT NOT NULL DEFAULT '',
    available_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    dispatched_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_outbox_claim
    ON outbox_messages (status, available_at, created_at);
CREATE INDEX IF NOT EXISTS idx_outbox_aggregate
    ON outbox_messages (aggregate_type, aggregate_id);`
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("outbox: ensure schema: %w", err)
	}
	return nil
}

// Enqueue appends messages, optionally inside the caller's transaction.
func (s *PostgresStore) Enqueue(ctx context.Context, tx Execer, msgs ...Message) error {
	if len(msgs) == 0 {
		return nil
	}

	execer := Execer(s.db)
	if tx != nil {
		execer = tx
	}

	for i := range msgs {
		msg := &msgs[i]
		if err := msg.Validate(); err != nil {
			return err
		}
		if msg.ID == uuid.Nil {
			msg.ID = uuid.New()
		}

		headers, err := json.Marshal(msg.Headers)
		if err != nil {
			return fmt.Errorf("outbox: encode headers: %w", err)
		}
		if string(headers) == "null" {
			headers = []byte("{}")
		}

		availableAt := msg.AvailableAt
		if availableAt.IsZero() {
			availableAt = time.Now()
		}

		_, err = execer.ExecContext(ctx, `
INSERT INTO outbox_messages
    (id, aggregate_type, aggregate_id, type, payload, headers, status, available_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			msg.ID, msg.AggregateType, msg.AggregateID, msg.Type, msg.Payload, headers, StatusPending, availableAt, time.Now())
		if err != nil {
			return fmt.Errorf("outbox: enqueue message %s: %w", msg.Type, err)
		}
	}
	return nil
}

// Claim moves due pending messages to in-flight and returns them.
//
// FOR UPDATE SKIP LOCKED is what makes concurrent dispatchers safe: each worker
// locks the rows it claims, other workers skip them, and no message is handed
// to two dispatchers at once. An in-flight row becomes re-claimable after a
// generous stale window, because a dispatcher that crashed mid-delivery never
// gets to mark its batch — without that recovery path those rows would be
// stuck in flight forever.
func (s *PostgresStore) Claim(ctx context.Context, limit int) ([]*Message, error) {
	const query = `
UPDATE outbox_messages SET status = 'in_flight', attempts = attempts + 1
WHERE id IN (
    SELECT id FROM outbox_messages
    WHERE (status = 'pending' AND available_at <= now())
       OR (status = 'in_flight' AND available_at <= now() - interval '10 minutes')
    ORDER BY created_at
    FOR UPDATE SKIP LOCKED
    LIMIT $1
)
RETURNING id, aggregate_type, aggregate_id, type, payload, headers, attempts, last_error, available_at, created_at`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("outbox: claim: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		var headers []byte
		if err := rows.Scan(
			&msg.ID, &msg.AggregateType, &msg.AggregateID, &msg.Type, &msg.Payload,
			&headers, &msg.Attempts, &msg.LastError, &msg.AvailableAt, &msg.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("outbox: scan claimed message: %w", err)
		}
		if err := json.Unmarshal(headers, &msg.Headers); err != nil {
			return nil, fmt.Errorf("outbox: decode headers: %w", err)
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("outbox: iterate claimed messages: %w", err)
	}
	return messages, nil
}

// MarkDispatched records a successful delivery.
func (s *PostgresStore) MarkDispatched(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE outbox_messages
SET status = $2, dispatched_at = now(), last_error = ''
WHERE id = $1`, id, StatusDispatched)
	if err != nil {
		return fmt.Errorf("outbox: mark dispatched %s: %w", id, err)
	}
	return nil
}

// MarkFailed either schedules a retry (pending + future available_at) or moves
// the message to the dead-letter state.
func (s *PostgresStore) MarkFailed(ctx context.Context, id uuid.UUID, lastErr error, retryAt time.Time, dead bool) error {
	status := StatusPending
	if dead {
		status = StatusDead
	}

	errText := ""
	if lastErr != nil {
		errText = lastErr.Error()
	}
	if len(errText) > 2000 {
		errText = errText[:2000]
	}

	var err error
	if dead {
		_, err = s.db.ExecContext(ctx, `
UPDATE outbox_messages SET status = $2, last_error = $3 WHERE id = $1`, id, status, errText)
	} else {
		_, err = s.db.ExecContext(ctx, `
UPDATE outbox_messages SET status = $2, last_error = $3, available_at = $4 WHERE id = $1`,
			id, status, errText, retryAt)
	}
	if err != nil {
		return fmt.Errorf("outbox: mark failed %s: %w", id, err)
	}
	return nil
}

// CountDead returns the number of dead-lettered messages, for health endpoints.
func (s *PostgresStore) CountDead(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT count(*) FROM outbox_messages WHERE status = $1`, StatusDead).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("outbox: count dead: %w", err)
	}
	return count, nil
}
