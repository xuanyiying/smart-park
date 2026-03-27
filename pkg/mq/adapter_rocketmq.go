package mq

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

type RocketMQAdapter struct {
	config RocketMQConfig
	logger Logger
	mu     sync.RWMutex
}

func NewRocketMQAdapter(cfg RocketMQConfig) *RocketMQAdapter {
	return &RocketMQAdapter{
		config: cfg,
		logger: NopLogger{},
	}
}

func (a *RocketMQAdapter) Name() string {
	return "rocketmq"
}

func (a *RocketMQAdapter) SetLogger(logger Logger) {
	a.logger = logger
}

func (a *RocketMQAdapter) getCredentials() primitive.Credentials {
	if a.config.AccessKey != "" && a.config.SecretKey != "" {
		return primitive.Credentials{
			AccessKey: a.config.AccessKey,
			SecretKey: a.config.SecretKey,
		}
	}
	return primitive.Credentials{}
}

func (a *RocketMQAdapter) NewProducer() (Producer, error) {
	opts := []producer.Option{
		producer.WithNameServer([]string{a.config.NameServer}),
		producer.WithGroupName(a.config.Group + "-producer"),
		producer.WithRetry(3),
		producer.WithSendMsgTimeout(10 * time.Second),
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
		a.logger.Errorw("failed to create producer", "error", err)
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	if err := p.Start(); err != nil {
		a.logger.Errorw("failed to start producer", "error", err)
		return nil, fmt.Errorf("failed to start producer: %w", err)
	}

	a.logger.Infow("rocketmq producer started", "nameServer", a.config.NameServer, "group", a.config.Group)
	return &RocketMQProducer{
		producer: p,
		logger:   a.logger,
	}, nil
}

func (a *RocketMQAdapter) NewConsumer() (Consumer, error) {
	opts := []consumer.Option{
		consumer.WithNameServer([]string{a.config.NameServer}),
		consumer.WithGroupName(a.config.Group),
		consumer.WithConsumerModel(consumer.Clustering),
		consumer.WithConsumeFromWhere(consumer.ConsumeFromLastOffset),
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
		a.logger.Errorw("failed to create consumer", "error", err)
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	a.logger.Infow("rocketmq consumer created", "nameServer", a.config.NameServer, "group", a.config.Group)
	return &RocketMQConsumer{
		consumer: c,
		logger:   a.logger,
		handlers: make(map[string]func(msg *Message) error),
		stopCh:   make(chan struct{}),
	}, nil
}

type RocketMQProducer struct {
	producer rocketmq.Producer
	logger   Logger
}

func (p *RocketMQProducer) Publish(ctx context.Context, topic string, msg *Message) error {
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

	result, err := p.producer.SendSync(ctx, rmqMsg)
	if err != nil {
		p.logger.Errorw("failed to publish message", "error", err, "topic", topic)
		return err
	}

	if result.Status != primitive.SendOK {
		p.logger.Errorw("message send failed", "status", result.Status, "topic", topic)
		return fmt.Errorf("send failed with status: %v", result.Status)
	}

	p.logger.Infow("message published", "topic", topic, "key", msg.Key, "msgID", result.MsgID)
	return nil
}

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

func (p *RocketMQProducer) Close() error {
	if p.producer != nil {
		return p.producer.Shutdown()
	}
	return nil
}

type RocketMQConsumer struct {
	consumer  rocketmq.PushConsumer
	logger    Logger
	handlers  map[string]func(msg *Message) error
	stopCh    chan struct{}
	mu        sync.RWMutex
	isRunning bool
}

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
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	c.handlers[topic] = handler

	if !c.isRunning {
		if err := c.consumer.Start(); err != nil {
			c.logger.Errorw("failed to start consumer", "error", err)
			return fmt.Errorf("failed to start consumer: %w", err)
		}
		c.isRunning = true
	}

	c.logger.Infow("subscribed to topic", "topic", topic, "group", group)
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

	if keys := msg.GetProperty("KEYS"); keys != "" {
		m.Key = keys
	}

	if id := msg.GetProperty("id"); id != "" {
		m.ID = id
	}

	if ts := msg.GetProperty("timestamp"); ts != "" {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			m.Timestamp = t
		}
	}

	for k, v := range msg.GetProperties() {
		if k != "KEYS" && k != "id" && k != "timestamp" {
			m.Headers[k] = v
		}
	}

	if m.ID == "" {
		m.ID = GenerateID()
	}

	return m
}

func (c *RocketMQConsumer) SubscribeAsync(ctx context.Context, topic string, group string, handler func(msg *Message) error) error {
	return c.Subscribe(ctx, topic, group, handler)
}

func (c *RocketMQConsumer) Unsubscribe(ctx context.Context, topic string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.consumer.Unsubscribe(topic); err != nil {
		c.logger.Errorw("failed to unsubscribe", "error", err, "topic", topic)
		return err
	}

	delete(c.handlers, topic)
	c.logger.Infow("unsubscribed from topic", "topic", topic)
	return nil
}

func (c *RocketMQConsumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isRunning && c.stopCh != nil {
		close(c.stopCh)
		c.isRunning = false
	}

	if c.consumer != nil {
		if err := c.consumer.Shutdown(); err != nil {
			c.logger.Errorw("failed to shutdown consumer", "error", err)
			return err
		}
	}

	c.logger.Infow("rocketmq consumer closed")
	return nil
}
