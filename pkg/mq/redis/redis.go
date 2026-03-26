package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type Logger interface {
	Infow(msg string, keysAndValues ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
	Debugw(msg string, keysAndValues ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
}

type NopLogger struct{}

func (NopLogger) Infow(msg string, keysAndValues ...interface{})  {}
func (NopLogger) Errorw(msg string, keysAndValues ...interface{}) {}
func (NopLogger) Debugw(msg string, keysAndValues ...interface{}) {}
func (NopLogger) Warnw(msg string, keysAndValues ...interface{})  {}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	Stream   string
}

type Producer interface {
	Publish(ctx context.Context, topic string, msg *Message) error
	PublishAsync(ctx context.Context, topic string, msg *Message) error
	Close() error
}

type Consumer interface {
	Subscribe(ctx context.Context, topic string, group string, handler func(msg *Message) error) error
	SubscribeAsync(ctx context.Context, topic string, group string, handler func(msg *Message) error) error
	Unsubscribe(ctx context.Context, topic string) error
	Close() error
}

type Adapter interface {
	NewProducer() (Producer, error)
	NewConsumer() (Consumer, error)
	Name() string
}

type Message struct {
	ID        string
	Topic     string
	Key       string
	Body      []byte
	Headers   map[string]string
	Timestamp time.Time
}

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

func (a *RedisAdapter) NewProducer() (Producer, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     a.config.Addr,
		Password: a.config.Password,
		DB:       a.config.DB,
	})
	a.client = client
	return &RedisProducer{client: client, logger: a.logger}, nil
}

func (a *RedisAdapter) NewConsumer() (Consumer, error) {
	if a.client == nil {
		client := redis.NewClient(&redis.Options{
			Addr:     a.config.Addr,
			Password: a.config.Password,
			DB:       a.config.DB,
		})
		a.client = client
	}
	return &RedisConsumer{client: a.client, config: a.config, logger: a.logger}, nil
}

var _ Adapter = (*RedisAdapter)(nil)

type RedisProducer struct {
	client *redis.Client
	logger Logger
}

func (p *RedisProducer) Publish(ctx context.Context, topic string, msg *Message) error {
	_, err := p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: topic,
		Values: map[string]interface{}{
			"id":      msg.ID,
			"key":     msg.Key,
			"body":    string(msg.Body),
			"headers": fmt.Sprintf("%v", msg.Headers),
		},
	}).Result()
	if err != nil {
		p.logger.Errorw("redis publish error", "error", err)
		return err
	}
	p.logger.Infow("redis publish success", "topic", topic, "key", msg.Key)
	return nil
}

func (p *RedisProducer) PublishAsync(ctx context.Context, topic string, msg *Message) error {
	return p.Publish(ctx, topic, msg)
}

func (p *RedisProducer) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

type RedisConsumer struct {
	client *redis.Client
	config RedisConfig
	logger Logger
}

func (c *RedisConsumer) Subscribe(ctx context.Context, topic string, group string, handler func(msg *Message) error) error {
	streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: fmt.Sprintf("consumer-%d", time.Now().Unix()),
		Streams:  []string{topic, ">"},
		Count:    10,
		Block:    time.Second * 5,
	}).Result()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		c.logger.Errorw("redis subscribe error", "error", err)
		return err
	}
	for _, stream := range streams {
		for _, msg := range stream.Messages {
			m := &Message{
				ID:    msg.ID,
				Topic: topic,
				Key:   msg.Values["key"].(string),
				Body:  []byte(msg.Values["body"].(string)),
			}
			if err := handler(m); err != nil {
				c.logger.Errorw("handler error", "error", err)
			} else {
				c.client.XAck(ctx, topic, group, msg.ID)
			}
		}
	}
	return nil
}

func (c *RedisConsumer) SubscribeAsync(ctx context.Context, topic string, group string, handler func(msg *Message) error) error {
	return c.Subscribe(ctx, topic, group, handler)
}

func (c *RedisConsumer) Unsubscribe(ctx context.Context, topic string) error {
	return nil
}

func (c *RedisConsumer) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
