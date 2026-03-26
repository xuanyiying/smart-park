//go:build nats
// +build nats

package nats

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
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
	conn   *nats.Conn
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

func (a *NATSAdapter) getConn() (*nats.Conn, error) {
	if a.conn == nil || !a.conn.IsConnected() {
		opts := []nats.Option{
			nats.Timeout(10 * time.Second),
			nats.ReconnectWait(5 * time.Second),
			nats.MaxReconnects(10),
		}

		if a.config.Username != "" && a.config.Password != "" {
			opts = append(opts, nats.UserInfo(a.config.Username, a.config.Password))
		}

		conn, err := nats.Connect(a.config.URL, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to NATS: %w", err)
		}
		a.conn = conn
	}
	return a.conn, nil
}

// NewProducer creates a new producer
func (a *NATSAdapter) NewProducer() (Producer, error) {
	conn, err := a.getConn()
	if err != nil {
		return nil, err
	}
	return &NATSProducer{
		conn:   conn,
		logger: a.logger,
	}, nil
}

// NewConsumer creates a new consumer
func (a *NATSAdapter) NewConsumer() (Consumer, error) {
	conn, err := a.getConn()
	if err != nil {
		return nil, err
	}
	return &NATSConsumer{
		conn:      conn,
		logger:    a.logger,
		subjects:  make(map[string]*nats.Subscription),
		jsSubs:    make(map[string]*nats.Subscription),
		stopCh:    make(chan struct{}),
	}, nil
}

// NATSProducer implements Producer for NATS
type NATSProducer struct {
	conn   *nats.Conn
	logger Logger
}

// Publish publishes a message
func (p *NATSProducer) Publish(ctx context.Context, topic string, msg *Message) error {
	natsMsg := &nats.Msg{
		Subject: topic,
		Data:    msg.Body,
		Header:  make(nats.Header),
	}

	// Add headers
	natsMsg.Header.Set("id", msg.ID)
	natsMsg.Header.Set("key", msg.Key)
	natsMsg.Header.Set("timestamp", msg.Timestamp.Format(time.RFC3339))
	for k, v := range msg.Headers {
		natsMsg.Header.Set(k, v)
	}

	if err := p.conn.PublishMsg(natsMsg); err != nil {
		p.logger.Errorw("failed to publish message", "error", err, "topic", topic)
		return err
	}

	p.logger.Infow("message published", "topic", topic, "key", msg.Key)
	return nil
}

// PublishAsync publishes a message asynchronously
func (p *NATSProducer) PublishAsync(ctx context.Context, topic string, msg *Message) error {
	go func() {
		if err := p.Publish(ctx, topic, msg); err != nil {
			p.logger.Errorw("async publish failed", "error", err, "topic", topic)
		}
	}()
	return nil
}

// Close closes the producer
func (p *NATSProducer) Close() error {
	return nil
}

// NATSConsumer implements Consumer for NATS
type NATSConsumer struct {
	conn      *nats.Conn
	logger    Logger
	subjects  map[string]*nats.Subscription
	jsSubs    map[string]*nats.Subscription
	stopCh    chan struct{}
	mu        sync.RWMutex
	isRunning bool
}

// Subscribe subscribes to a topic
func (c *NATSConsumer) Subscribe(ctx context.Context, topic string, group string, handler func(msg *Message) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Use JetStream for durable subscriptions
	js, err := c.conn.JetStream()
	if err != nil {
		c.logger.Errorw("failed to get JetStream context", "error", err)
		return err
	}

	// Create consumer
	sub, err := js.PullSubscribe(topic, group,
		nats.Durable(group),
		nats.DeliverAll(),
	)
	if err != nil {
		c.logger.Errorw("failed to subscribe", "error", err, "topic", topic, "group", group)
		return err
	}

	c.jsSubs[topic] = sub
	c.isRunning = true

	// Start consuming messages
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-c.stopCh:
				return
			default:
				msgs, err := sub.Fetch(10, nats.MaxWait(5*time.Second))
				if err != nil {
					if err != nats.ErrTimeout {
						c.logger.Errorw("failed to fetch messages", "error", err)
					}
					continue
				}

				for _, msg := range msgs {
					m := c.parseMessage(msg, topic)
					if err := handler(m); err != nil {
						c.logger.Errorw("message handler failed", "error", err, "msgID", m.ID)
						msg.Nak()
					} else {
						msg.Ack()
					}
				}
			}
		}
	}()

	return nil
}

func (c *NATSConsumer) parseMessage(msg *nats.Msg, topic string) *Message {
	m := &Message{
		Topic:   topic,
		Body:    msg.Data,
		Headers: make(map[string]string),
	}

	if msg.Header != nil {
		m.ID = msg.Header.Get("id")
		m.Key = msg.Header.Get("key")
		if ts := msg.Header.Get("timestamp"); ts != "" {
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				m.Timestamp = t
			}
		}
		for k, v := range msg.Header {
			if len(v) > 0 {
				m.Headers[k] = v[0]
			}
		}
	}

	return m
}

// SubscribeAsync subscribes to a topic asynchronously
func (c *NATSConsumer) SubscribeAsync(ctx context.Context, topic string, group string, handler func(msg *Message) error) error {
	return c.Subscribe(ctx, topic, group, handler)
}

// Unsubscribe unsubscribes from a topic
func (c *NATSConsumer) Unsubscribe(ctx context.Context, topic string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if sub, ok := c.jsSubs[topic]; ok {
		if err := sub.Unsubscribe(); err != nil {
			return err
		}
		delete(c.jsSubs, topic)
	}

	if sub, ok := c.subjects[topic]; ok {
		if err := sub.Unsubscribe(); err != nil {
			return err
		}
		delete(c.subjects, topic)
	}

	return nil
}

// Close closes the consumer
func (c *NATSConsumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isRunning {
		close(c.stopCh)
		c.isRunning = false
	}

	for _, sub := range c.jsSubs {
		sub.Unsubscribe()
	}
	for _, sub := range c.subjects {
		sub.Unsubscribe()
	}

	return nil
}
