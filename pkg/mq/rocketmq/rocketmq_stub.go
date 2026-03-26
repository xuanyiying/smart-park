//go:build !rocketmq
// +build !rocketmq

package rocketmq

import (
	"context"
	"errors"
	"time"
)

var ErrNotImplemented = errors.New("RocketMQ support not enabled. Build with -tags rocketmq to enable")

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

// Config holds RocketMQ configuration
type Config struct {
	NameServer string
	Group      string
	AccessKey  string
	SecretKey  string
	Namespace  string
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

// RocketMQAdapter implements Adapter for RocketMQ
type RocketMQAdapter struct {
	config Config
	logger Logger
}

// NewRocketMQAdapter creates a new RocketMQ adapter
func NewRocketMQAdapter(cfg Config) *RocketMQAdapter {
	return &RocketMQAdapter{
		config: cfg,
		logger: NopLogger{},
	}
}

// SetLogger sets the logger
func (a *RocketMQAdapter) SetLogger(logger Logger) {
	a.logger = logger
}

// Name returns adapter name
func (a *RocketMQAdapter) Name() string {
	return "rocketmq"
}

// NewProducer creates a new producer
func (a *RocketMQAdapter) NewProducer() (Producer, error) {
	return nil, ErrNotImplemented
}

// NewConsumer creates a new consumer
func (a *RocketMQAdapter) NewConsumer() (Consumer, error) {
	return nil, ErrNotImplemented
}
