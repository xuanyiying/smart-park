//go:build rocketmq
// +build rocketmq

package rocketmq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
)

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

// getCredentials returns credentials if configured
func (a *RocketMQAdapter) getCredentials() primitive.Credentials {
	if a.config.AccessKey != "" && a.config.SecretKey != "" {
		return primitive.Credentials{
			AccessKey: a.config.AccessKey,
			SecretKey: a.config.SecretKey,
		}
	}
	return primitive.Credentials{}
}

// NewProducer creates a new producer
func (a *RocketMQAdapter) NewProducer() (Producer, error) {
	opts := []producer.Option{
		producer.WithNameServer([]string{a.config.NameServer}),
		producer.WithGroupName(a.config.Group + "-producer"),
	}

	creds := a.getCredentials()
	if creds.AccessKey != "" {
		opts = append(opts, producer.WithCredentials(creds))
	}

	if a.config.Namespace != "" {
		opts = append(opts, producer.WithNamespace(a.config.Namespace))
	}

	p, err := rocketmq.NewProducer(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	if err := p.Start(); err != nil {
		return nil, fmt.Errorf("failed to start producer: %w", err)
	}

	return &RocketMQProducer{
		producer: p,
		logger:   a.logger,
	}, nil
}

// NewConsumer creates a new consumer
func (a *RocketMQAdapter) NewConsumer() (Consumer, error) {
	opts := []consumer.Option{
		consumer.WithNameServer([]string{a.config.NameServer}),
		consumer.WithGroupName(a.config.Group),
		consumer.WithConsumerModel(consumer.Clustering),
	}

	creds := a.getCredentials()
	if creds.AccessKey != "" {
		opts = append(opts, consumer.WithCredentials(creds))
	}

	if a.config.Namespace != "" {
		opts = append(opts, consumer.WithNamespace(a.config.Namespace))
	}

	c, err := rocketmq.NewPushConsumer(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	return &RocketMQConsumer{
		consumer:  c,
		logger:    a.logger,
		handlers:  make(map[string]func(msg *Message) error),
		stopCh:    make(chan struct{}),
	}, nil
}

// RocketMQProducer implements Producer for RocketMQ
type RocketMQProducer struct {
	producer rocketmq.Producer
	logger   Logger
}

// Publish publishes a message
func (p *RocketMQProducer) Publish(ctx context.Context, topic string, msg *Message) error {
	rmqMsg := &primitive.Message{
		Topic: topic,
		Body:  msg.Body,
	}

	// Set key if provided
	if msg.Key != "" {
		rmqMsg.WithKeys([]string{msg.Key})
	}

	// Add properties
	if msg.ID != "" {
		rmqMsg.WithProperty("id", msg.ID)
	}
	rmqMsg.WithProperty("timestamp", msg.Timestamp.Format(time.RFC3339))
	for k, v := range msg.Headers {
		rmqMsg.WithProperty(k, v)
	}

	result, err := p.producer.SendSync(ctx, rmqMsg)
	if err != nil {
		p.logger.Errorw("failed to publish message", "error", err, "topic", topic)
		return err
	}

	if result.Status != primitive.SendOK {
		p.logger.Errorw("message send failed", "status", result.Status, "topic", topic)
		return fmt.Errorf("send failed with status: %s", result.Status)
	}

	p.logger.Infow("message published", "topic", topic, "key", msg.Key, "msgID", result.MsgID)
	return nil
}

// PublishAsync publishes a message asynchronously
func (p *RocketMQProducer) PublishAsync(ctx context.Context, topic string, msg *Message) error {
	rmqMsg := &primitive.Message{
		Topic: topic,
		Body:  msg.Body,
	}

	if msg.Key != "" {
		rmqMsg.WithKeys([]string{msg.Key})
	}

	if msg.ID != "" {
		rmqMsg.WithProperty("id", msg.ID)
	}
	rmqMsg.WithProperty("timestamp", msg.Timestamp.Format(time.RFC3339))
	for k, v := range msg.Headers {
		rmqMsg.WithProperty(k, v)
	}

	p.producer.SendAsync(ctx, func(ctx context.Context, result *primitive.SendResult, err error) {
		if err != nil {
			p.logger.Errorw("async publish failed", "error", err, "topic", topic)
		} else if result.Status != primitive.SendOK {
			p.logger.Errorw("async send failed", "status", result.Status, "topic", topic)
		} else {
			p.logger.Infow("async message published", "topic", topic, "key", msg.Key, "msgID", result.MsgID)
		}
	}, rmqMsg)

	return nil
}

// Close closes the producer
func (p *RocketMQProducer) Close() error {
	return p.producer.Shutdown()
}

// RocketMQConsumer implements Consumer for RocketMQ
type RocketMQConsumer struct {
	consumer  rocketmq.PushConsumer
	logger    Logger
	handlers  map[string]func(msg *Message) error
	stopCh    chan struct{}
	mu        sync.RWMutex
	isRunning bool
}

// Subscribe subscribes to a topic
func (c *RocketMQConsumer) Subscribe(ctx context.Context, topic string, group string, handler func(msg *Message) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	selector := consumer.MessageSelector{
		Type:       consumer.TAG,
		Expression: "*",
	}

	err := c.consumer.Subscribe(topic, selector, func(ctx context.Context,
		msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		for _, msg := range msgs {
			m := c.parseMessage(msg, topic)
			if err := handler(m); err != nil {
				c.logger.Errorw("message handler failed", "error", err, "msgID", m.ID)
				return consumer.ConsumeRetryLater, nil
			}
		}
		return consumer.ConsumeSuccess, nil
	})

	if err != nil {
		c.logger.Errorw("failed to subscribe", "error", err, "topic", topic)
		return err
	}

	c.handlers[topic] = handler

	if !c.isRunning {
		if err := c.consumer.Start(); err != nil {
			return fmt.Errorf("failed to start consumer: %w", err)
		}
		c.isRunning = true
	}

	return nil
}

func (c *RocketMQConsumer) parseMessage(msg *primitive.MessageExt, topic string) *Message {
	m := &Message{
		ID:        msg.MsgId,
		Topic:     topic,
		Body:      msg.Body,
		Headers:   make(map[string]string),
		Timestamp: time.Unix(msg.BornTimestamp/1000, 0),
	}

	// Get key from properties
	if keys, ok := msg.GetProperty("KEYS"); ok {
		m.Key = keys
	}

	// Get custom ID if set
	if id, ok := msg.GetProperty("id"); ok {
		m.ID = id
	}

	// Parse timestamp
	if ts, ok := msg.GetProperty("timestamp"); ok {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			m.Timestamp = t
		}
	}

	// Copy all properties as headers
	for k, v := range msg.GetProperties() {
		if k != "KEYS" && k != "id" && k != "timestamp" {
			m.Headers[k] = v
		}
	}

	return m
}

// SubscribeAsync subscribes to a topic asynchronously
func (c *RocketMQConsumer) SubscribeAsync(ctx context.Context, topic string, group string, handler func(msg *Message) error) error {
	return c.Subscribe(ctx, topic, group, handler)
}

// Unsubscribe unsubscribes from a topic
func (c *RocketMQConsumer) Unsubscribe(ctx context.Context, topic string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.handlers, topic)
	return nil
}

// Close closes the consumer
func (c *RocketMQConsumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isRunning {
		close(c.stopCh)
		c.isRunning = false
	}

	return c.consumer.Shutdown()
}
