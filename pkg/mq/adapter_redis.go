package mq

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisAdapter struct {
	config RedisConfig
	client *redis.Client
	logger Logger
}

func NewRedisAdapter(cfg RedisConfig) *RedisAdapter {
	return &RedisAdapter{
		config: cfg,
		logger: NopLogger{},
	}
}

func (a *RedisAdapter) Name() string {
	return "redis"
}

func (a *RedisAdapter) SetLogger(logger Logger) {
	a.logger = logger
}

func (a *RedisAdapter) getClient() *redis.Client {
	if a.client == nil {
		a.client = redis.NewClient(&redis.Options{
			Addr:     a.config.Addr,
			Password: a.config.Password,
			DB:       a.config.DB,
		})
	}
	return a.client
}

func (a *RedisAdapter) NewProducer() (Producer, error) {
	client := a.getClient()
	return &RedisProducer{
		client: client,
		logger: a.logger,
	}, nil
}

func (a *RedisAdapter) NewConsumer() (Consumer, error) {
	client := a.getClient()
	return &RedisConsumer{
		client: client,
		config: a.config,
		logger: a.logger,
	}, nil
}

type RedisProducer struct {
	client *redis.Client
	logger Logger
}

func (p *RedisProducer) Publish(ctx context.Context, topic string, msg *Message) error {
	_, err := p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: topic,
		Values: map[string]interface{}{
			"id":        msg.ID,
			"key":       msg.Key,
			"body":      string(msg.Body),
			"headers":   msg.Headers,
			"timestamp": msg.Timestamp.Unix(),
		},
	}).Result()
	if err != nil {
		p.logger.Errorw("failed to publish message", "error", err, "topic", topic)
		return err
	}
	p.logger.Infow("message published", "topic", topic, "key", msg.Key)
	return nil
}

func (p *RedisProducer) PublishAsync(ctx context.Context, topic string, msg *Message) error {
	go func() {
		if err := p.Publish(ctx, topic, msg); err != nil {
			p.logger.Errorw("async publish failed", "error", err, "topic", topic)
		}
	}()
	return nil
}

func (p *RedisProducer) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

type RedisConsumer struct {
	client    *redis.Client
	config    RedisConfig
	logger    Logger
	stopCh    chan struct{}
	isRunning bool
}

func (c *RedisConsumer) Subscribe(ctx context.Context, topic string, group string, handler func(msg *Message) error) error {
	_, err := c.client.XGroupCreateMkStream(ctx, topic, group, "$").Result()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		c.logger.Errorw("failed to create consumer group", "error", err, "topic", topic, "group", group)
		return err
	}

	consumer := fmt.Sprintf("consumer-%d", time.Now().UnixNano())

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopCh:
			return nil
		default:
			streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    group,
				Consumer: consumer,
				Streams:  []string{topic, ">"},
				Count:    10,
				Block:    5 * time.Second,
			}).Result()

			if err != nil {
				if err == redis.Nil {
					continue
				}
				c.logger.Errorw("failed to read from stream", "error", err, "topic", topic)
				continue
			}

			for _, stream := range streams {
				for _, msg := range stream.Messages {
					m := c.parseMessage(&msg, topic)
					if err := handler(m); err != nil {
						c.logger.Errorw("message handler failed", "error", err, "msgID", msg.ID)
					} else {
						c.client.XAck(ctx, topic, group, msg.ID)
					}
				}
			}
		}
	}
}

func (c *RedisConsumer) parseMessage(msg *redis.XMessage, topic string) *Message {
	m := &Message{
		ID:      msg.ID,
		Topic:   topic,
		Headers: make(map[string]string),
	}

	if key, ok := msg.Values["key"].(string); ok {
		m.Key = key
	}
	if body, ok := msg.Values["body"].(string); ok {
		m.Body = []byte(body)
	}
	if ts, ok := msg.Values["timestamp"].(int64); ok {
		m.Timestamp = time.Unix(ts, 0)
	}

	return m
}

func (c *RedisConsumer) SubscribeAsync(ctx context.Context, topic string, group string, handler func(msg *Message) error) error {
	go func() {
		if err := c.Subscribe(ctx, topic, group, handler); err != nil {
			c.logger.Errorw("async subscription failed", "error", err, "topic", topic)
		}
	}()
	return nil
}

func (c *RedisConsumer) Unsubscribe(ctx context.Context, topic string) error {
	if c.stopCh != nil {
		close(c.stopCh)
	}
	c.isRunning = false
	return nil
}

func (c *RedisConsumer) Close() error {
	if c.isRunning && c.stopCh != nil {
		close(c.stopCh)
	}
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
