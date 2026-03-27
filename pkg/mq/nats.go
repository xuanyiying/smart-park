package mq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type NATSAdapter struct {
	config NATSConfig
	conn   *nats.Conn
	logger Logger
	mu     sync.RWMutex
}

func NewNATSAdapter(cfg NATSConfig) *NATSAdapter {
	return &NATSAdapter{
		config: cfg,
		logger: NopLogger{},
	}
}

func (a *NATSAdapter) Name() string {
	return "nats"
}

func (a *NATSAdapter) SetLogger(logger Logger) {
	a.logger = logger
}

func (a *NATSAdapter) getConn() (*nats.Conn, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.conn != nil && a.conn.IsConnected() {
		return a.conn, nil
	}

	opts := []nats.Option{
		nats.Timeout(10 * time.Second),
		nats.ReconnectWait(5 * time.Second),
		nats.MaxReconnects(10),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				a.logger.Errorw("nats disconnected", "error", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			a.logger.Infow("nats reconnected", "url", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			a.logger.Infow("nats connection closed")
		}),
	}

	if a.config.Username != "" && a.config.Password != "" {
		opts = append(opts, nats.UserInfo(a.config.Username, a.config.Password))
	}

	conn, err := nats.Connect(a.config.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	a.conn = conn
	a.logger.Infow("nats connected", "url", a.config.URL)
	return conn, nil
}

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

func (a *NATSAdapter) NewConsumer() (Consumer, error) {
	conn, err := a.getConn()
	if err != nil {
		return nil, err
	}
	return &NATSConsumer{
		conn:   conn,
		logger: a.logger,
		subs:   make(map[string]*nats.Subscription),
		jsSubs: make(map[string]*nats.Subscription),
		stopCh: make(chan struct{}),
	}, nil
}

func (a *NATSAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.conn != nil {
		a.conn.Close()
		a.conn = nil
	}
	return nil
}

type NATSProducer struct {
	conn   *nats.Conn
	logger Logger
}

func (p *NATSProducer) Publish(ctx context.Context, topic string, msg *Message) error {
	natsMsg := &nats.Msg{
		Subject: topic,
		Data:    msg.Body,
		Header:  make(nats.Header),
	}

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

	p.logger.Infow("message published", "topic", topic, "key", msg.Key, "id", msg.ID)
	return nil
}

func (p *NATSProducer) PublishAsync(ctx context.Context, topic string, msg *Message) error {
	go func() {
		if err := p.Publish(ctx, topic, msg); err != nil {
			p.logger.Errorw("async publish failed", "error", err, "topic", topic)
		}
	}()
	return nil
}

func (p *NATSProducer) Close() error {
	return nil
}

type NATSConsumer struct {
	conn      *nats.Conn
	logger    Logger
	subs      map[string]*nats.Subscription
	jsSubs    map[string]*nats.Subscription
	stopCh    chan struct{}
	mu        sync.RWMutex
	isRunning bool
}

func (c *NATSConsumer) Subscribe(ctx context.Context, topic string, group string, handler func(msg *Message) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	js, err := c.conn.JetStream()
	if err != nil {
		c.logger.Errorw("failed to get JetStream context", "error", err)
		return fmt.Errorf("failed to get JetStream context: %w", err)
	}

	streamName := fmt.Sprintf("%s-stream", topic)
	_, err = js.StreamInfo(streamName)
	if err != nil {
		_, err = js.AddStream(&nats.StreamConfig{
			Name:      streamName,
			Subjects:  []string{topic},
			Replicas:  1,
			Retention: nats.LimitsPolicy,
		})
		if err != nil {
			c.logger.Errorw("failed to create stream", "error", err, "stream", streamName)
			return fmt.Errorf("failed to create stream: %w", err)
		}
	}

	durableName := fmt.Sprintf("%s-%s", group, topic)
	sub, err := js.PullSubscribe(topic, durableName,
		nats.Durable(durableName),
		nats.ManualAck(),
		nats.DeliverAll(),
	)
	if err != nil {
		c.logger.Errorw("failed to subscribe", "error", err, "topic", topic, "group", group)
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	c.jsSubs[topic] = sub
	c.isRunning = true

	go c.consumeMessages(ctx, topic, sub, handler)

	c.logger.Infow("subscribed to topic", "topic", topic, "group", group)
	return nil
}

func (c *NATSConsumer) consumeMessages(ctx context.Context, topic string, sub *nats.Subscription, handler func(msg *Message) error) {
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
					c.logger.Errorw("failed to fetch messages", "error", err, "topic", topic)
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

	if m.ID == "" {
		m.ID = GenerateID()
	}

	return m
}

func (c *NATSConsumer) SubscribeAsync(ctx context.Context, topic string, group string, handler func(msg *Message) error) error {
	return c.Subscribe(ctx, topic, group, handler)
}

func (c *NATSConsumer) Unsubscribe(ctx context.Context, topic string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if sub, ok := c.jsSubs[topic]; ok {
		if err := sub.Unsubscribe(); err != nil {
			c.logger.Errorw("failed to unsubscribe from jetstream", "error", err, "topic", topic)
			return err
		}
		delete(c.jsSubs, topic)
	}

	if sub, ok := c.subs[topic]; ok {
		if err := sub.Unsubscribe(); err != nil {
			c.logger.Errorw("failed to unsubscribe", "error", err, "topic", topic)
			return err
		}
		delete(c.subs, topic)
	}

	c.logger.Infow("unsubscribed from topic", "topic", topic)
	return nil
}

func (c *NATSConsumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isRunning && c.stopCh != nil {
		close(c.stopCh)
		c.isRunning = false
	}

	for topic, sub := range c.jsSubs {
		if err := sub.Unsubscribe(); err != nil {
			c.logger.Errorw("failed to unsubscribe from jetstream", "error", err, "topic", topic)
		}
	}

	for topic, sub := range c.subs {
		if err := sub.Unsubscribe(); err != nil {
			c.logger.Errorw("failed to unsubscribe", "error", err, "topic", topic)
		}
	}

	c.logger.Infow("nats consumer closed")
	return nil
}
