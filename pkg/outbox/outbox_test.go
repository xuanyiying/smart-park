package outbox

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// storedMsg is a message plus its current status in the fake store.
type storedMsg struct {
	Message
	status string
}

// fakeStore records state transitions so the dispatcher's decision logic can be
// asserted without a database.
type fakeStore struct {
	mu         sync.Mutex
	messages   map[uuid.UUID]*storedMsg
	dispatched []uuid.UUID
	failed     []uuid.UUID
	dead       []uuid.UUID
	claimErr   error
}

func newFakeStore(msgs ...Message) *fakeStore {
	fs := &fakeStore{messages: make(map[uuid.UUID]*storedMsg)}
	for i := range msgs {
		m := msgs[i]
		m.Attempts = 1 // as if claimed once
		fs.messages[m.ID] = &storedMsg{Message: m, status: StatusPending}
	}
	return fs
}

func (s *fakeStore) Enqueue(ctx context.Context, tx Execer, msgs ...Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range msgs {
		m := msgs[i]
		m.Attempts = 0
		s.messages[m.ID] = &storedMsg{Message: m, status: StatusPending}
	}
	return nil
}

func (s *fakeStore) Claim(ctx context.Context, limit int) ([]*Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.claimErr != nil {
		return nil, s.claimErr
	}
	var out []*Message
	for _, sm := range s.messages {
		if sm.status == StatusPending && len(out) < limit {
			sm.Attempts++
			out = append(out, &sm.Message)
		}
	}
	return out, nil
}

func (s *fakeStore) MarkDispatched(ctx context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[id].status = StatusDispatched
	s.dispatched = append(s.dispatched, id)
	return nil
}

func (s *fakeStore) MarkFailed(ctx context.Context, id uuid.UUID, lastErr error, retryAt time.Time, dead bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if dead {
		s.messages[id].status = StatusDead
		s.dead = append(s.dead, id)
		return nil
	}
	s.failed = append(s.failed, id)
	return nil
}

func (s *fakeStore) status(id uuid.UUID) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.messages[id].status
}

func TestNewMessageDefaults(t *testing.T) {
	msg := NewMessage("order", "o-123", "order.settled", []byte(`{"amount":1000}`), nil)
	if err := msg.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if msg.ID == uuid.Nil {
		t.Fatal("expected a fresh id")
	}
	if msg.AvailableAt.IsZero() {
		t.Fatal("expected immediate availability")
	}
}

func TestMessageValidate(t *testing.T) {
	if err := (&Message{}).Validate(); err == nil {
		t.Fatal("expected type-less message to be rejected")
	}
	if err := (&Message{Type: "x"}).Validate(); err == nil {
		t.Fatal("expected payload-less message to be rejected")
	}
	if err := (&Message{Type: "x", Payload: []byte("{}")}).Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
}

func TestDispatcherMarksDispatched(t *testing.T) {
	msg := NewMessage("order", "o-1", "order.settled", []byte(`{}`), nil)
	store := newFakeStore(msg)

	var delivered []*Message
	handler := func(ctx context.Context, m *Message) error {
		delivered = append(delivered, m)
		return nil
	}

	NewDispatcher(store, handler, DispatcherConfig{PollInterval: time.Millisecond}, nil).
		sweep(context.Background())

	if len(delivered) != 1 || delivered[0].ID != msg.ID {
		t.Fatalf("handler received %d messages, want the claimed one", len(delivered))
	}
	if store.status(msg.ID) != StatusDispatched {
		t.Fatalf("status = %s, want dispatched", store.status(msg.ID))
	}
}

func TestDispatcherRetriesThenSucceeds(t *testing.T) {
	msg := NewMessage("order", "o-1", "order.settled", []byte(`{}`), nil)
	store := newFakeStore(msg)

	attempts := 0
	handler := func(ctx context.Context, m *Message) error {
		attempts++
		if attempts < 3 {
			return errors.New("downstream unavailable")
		}
		return nil
	}

	d := NewDispatcher(store, handler, DispatcherConfig{PollInterval: time.Millisecond}, nil)
	for i := 0; i < 3; i++ {
		d.sweep(context.Background())
	}

	if attempts != 3 {
		t.Fatalf("handler attempts = %d, want 3", attempts)
	}
	if store.status(msg.ID) != StatusDispatched {
		t.Fatalf("status = %s, want dispatched after the third attempt", store.status(msg.ID))
	}
	if len(store.failed) != 2 {
		t.Fatalf("expected two failed attempts recorded, got %d", len(store.failed))
	}
}

func TestDispatcherDeadLettersAfterMaxAttempts(t *testing.T) {
	msg := NewMessage("order", "o-1", "order.settled", []byte(`{}`), nil)
	store := newFakeStore(msg)

	handler := func(ctx context.Context, m *Message) error {
		return errors.New("permanent failure")
	}

	d := NewDispatcher(store, handler, DispatcherConfig{
		PollInterval:   time.Millisecond,
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
	}, nil)
	for i := 0; i < 5; i++ {
		d.sweep(context.Background())
	}

	if store.status(msg.ID) != StatusDead {
		t.Fatalf("status = %s, want dead", store.status(msg.ID))
	}
	if len(store.dispatched) != 0 {
		t.Fatal("a permanently failing message must never be marked dispatched")
	}
}

func TestDispatcherSurvivesClaimErrors(t *testing.T) {
	store := newFakeStore()
	store.claimErr = errors.New("db down")

	handlerCalled := false
	handler := func(ctx context.Context, m *Message) error {
		handlerCalled = true
		return nil
	}

	var claimFailures int
	d := NewDispatcher(store, handler, DispatcherConfig{PollInterval: time.Millisecond}, func(m *Message, err error) {
		if m == nil {
			claimFailures++
		}
	})

	// A failing store must not panic the loop; the next sweep retries.
	d.sweep(context.Background())
	store.claimErr = nil

	msg := NewMessage("order", "o-2", "order.settled", []byte(`{}`), nil)
	_ = store.Enqueue(context.Background(), nil, msg)
	d.sweep(context.Background())

	if claimFailures != 1 {
		t.Fatalf("expected one claim failure notification, got %d", claimFailures)
	}
	if !handlerCalled {
		t.Fatal("expected the handler to run after the store recovered")
	}
}

func TestRetryDelayIsExponentialAndCapped(t *testing.T) {
	cfg := DispatcherConfig{InitialBackoff: 100 * time.Millisecond, MaxBackoff: time.Second}

	if got := retryDelay(cfg, 1); got != 100*time.Millisecond {
		t.Fatalf("first retry delay = %v, want 100ms", got)
	}
	if got := retryDelay(cfg, 2); got != 200*time.Millisecond {
		t.Fatalf("second retry delay = %v, want 200ms", got)
	}
	if got := retryDelay(cfg, 3); got != 400*time.Millisecond {
		t.Fatalf("third retry delay = %v, want 400ms", got)
	}
	if got := retryDelay(cfg, 10); got != time.Second {
		t.Fatalf("capped retry delay = %v, want 1s", got)
	}
}
