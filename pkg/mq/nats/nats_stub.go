//go:build !nats
// +build !nats

package nats

import (
	"context"
	"errors"
	"time"
)

var ErrNotImplemented = errors.New("NATS support not enabled. Build with -tags nats to enable")

// Logger interface for logging
type Logger interface {
	Infow(msg string, keysAndValues ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
	Debugw(msg string, keysAndValues ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
}

// NopLogger is a no-op logger
type NopLogger struct{}

func (NopLogger) Infow(msg string, keysAndValues ...interface{})  {}
func (NopLogger) Errorw(msg string, keysAndValues ...interface{}) {}
func (NopLogger) Debugw(msg string, keysAndValues ...interface{}) {}
func (NopLogger) Warnw(msg string, keysAndValues ...interface{})  {}

// Config holds NATS configuration
type Config struct {
	URL      string
	Username string
	Password string
}

// Message represents a message
type Message struct {
	ID        string
	Topic     string
	Key       string
	Body      []byte
	Headers   map[string]string
	Timestamp time.Time
}

// Producer interface
type Producer interface {
	Publish(ctx context.Context, topic string, msg *Message) error
	PublishAsync(ctx context.Context, topic string, msg *Message) error
	Close() error
}

// Consumer interface
type Consumer interface {
	Subscribe(ctx context.Context, topic string, group string, handler func(msg *Message) error) error
	SubscribeAsync(ctx context.Context, topic string, group string, handler func(msg *Message) error) error
	Unsubscribe(ctx context.Context, topic string) error
	Close() error
}

// Adapter interface
type Adapter interface {
	NewProducer() (Producer, error)
	NewConsumer() (Consumer, error)
	Name() string
}

// NATSAdapter implements Adapter for NATS
type NATSAdapter struct {
	config Config
	logger Logger
}

// NewNATSAdapter creates a new NATS adapter
func NewNATSAdapter(cfg Config) *NATSAdapter {
	return &NATSAdapter{
		config: cfg,
		logger: NopLogger{},
	}
}

// SetLogger sets the logger
func (a *NATSAdapter) SetLogger(logger Logger) {
	a.logger = logger
}

// Name returns adapter name
func (a *NATSAdapter) Name() string {
	return "nats"
}

// NewProducer creates a new producer
func (a *NATSAdapter) NewProducer() (Producer, error) {
	return nil, ErrNotImplemented
}

// NewConsumer creates a new consumer
func (a *NATSAdapter) NewConsumer() (Consumer, error) {
	return nil, ErrNotImplemented
}
