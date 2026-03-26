package mq

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	ErrNotImplemented = errors.New("operation not implemented")
	ErrInvalidConfig  = errors.New("invalid config")
	ErrConnectionLost = errors.New("connection lost")
)

type Message struct {
	ID        string
	Topic     string
	Key       string
	Body      []byte
	Headers   map[string]string
	Timestamp time.Time
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

type MQType string

const (
	MQTypeRedis    MQType = "redis"
	MQTypeNATS     MQType = "nats"
	MQTypeRocketMQ MQType = "rocketmq"
)

type Config struct {
	Type     MQType         `json:"type"`
	Redis    RedisConfig    `json:"redis"`
	NATS     NATSConfig     `json:"nats"`
	RocketMQ RocketMQConfig `json:"rocketmq"`
}

type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	Stream   string `json:"stream"`
}

type NATSConfig struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type RocketMQConfig struct {
	NameServer string `json:"nameServer"`
	Group      string `json:"group"`
	AccessKey  string `json:"accessKey"`
	SecretKey  string `json:"secretKey"`
}

func NewMessage(topic, key string, body []byte) *Message {
	return &Message{
		ID:        GenerateID(),
		Topic:     topic,
		Key:       key,
		Body:      body,
		Headers:   make(map[string]string),
		Timestamp: time.Now(),
	}
}

func (m *Message) WithHeader(key, value string) *Message {
	m.Headers[key] = value
	return m
}

func GenerateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return time.Now().Format("20060102150405") + "-" + hex.EncodeToString(b)
}

type MQFactory struct {
	mu       sync.RWMutex
	adapters map[MQType]Adapter
}

func NewMQFactory() *MQFactory {
	return &MQFactory{
		adapters: make(map[MQType]Adapter),
	}
}

func (f *MQFactory) Register(mqType MQType, adapter Adapter) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.adapters[mqType] = adapter
}

func (f *MQFactory) GetAdapter(mqType MQType) (Adapter, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	adapter, ok := f.adapters[mqType]
	if !ok {
		return nil, fmt.Errorf("mq adapter not registered: %s", mqType)
	}
	return adapter, nil
}

func (f *MQFactory) ListAdapters() []MQType {
	f.mu.RLock()
	defer f.mu.RUnlock()
	types := make([]MQType, 0, len(f.adapters))
	for t := range f.adapters {
		types = append(types, t)
	}
	return types
}

func CreateMQAdapter(cfg Config) (Adapter, error) {
	switch cfg.Type {
	case MQTypeRedis:
		return NewRedisAdapter(cfg.Redis), nil
	case MQTypeNATS:
		return NewNATSAdapter(cfg.NATS), nil
	case MQTypeRocketMQ:
		return NewRocketMQAdapter(cfg.RocketMQ), nil
	default:
		return nil, fmt.Errorf("unsupported MQ type: %s", cfg.Type)
	}
}

type NATSAdapter struct {
	config NATSConfig
}

func NewNATSAdapter(cfg NATSConfig) *NATSAdapter {
	return &NATSAdapter{config: cfg}
}

func (a *NATSAdapter) Name() string {
	return "nats"
}

func (a *NATSAdapter) NewProducer() (Producer, error) {
	return nil, ErrNotImplemented
}

func (a *NATSAdapter) NewConsumer() (Consumer, error) {
	return nil, ErrNotImplemented
}

type RocketMQAdapter struct {
	config RocketMQConfig
}

func NewRocketMQAdapter(cfg RocketMQConfig) *RocketMQAdapter {
	return &RocketMQAdapter{config: cfg}
}

func (a *RocketMQAdapter) Name() string {
	return "rocketmq"
}

func (a *RocketMQAdapter) NewProducer() (Producer, error) {
	return nil, ErrNotImplemented
}

func (a *RocketMQAdapter) NewConsumer() (Consumer, error) {
	return nil, ErrNotImplemented
}

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

func NewRedisAdapter(cfg RedisConfig) Adapter {
	return NewRedisMQAdapter(cfg)
}

// RedisMQAdapter implements Adapter interface for Redis
type RedisMQAdapter struct {
	config RedisConfig
	client *redis.Client
	logger Logger
}

// NewRedisMQAdapter creates a new Redis MQ adapter
func NewRedisMQAdapter(cfg RedisConfig) *RedisMQAdapter {
	return &RedisMQAdapter{
		config: cfg,
		logger: NopLogger{},
	}
}

func (a *RedisMQAdapter) Name() string {
	return "redis"
}

func (a *RedisMQAdapter) SetLogger(logger Logger) {
	a.logger = logger
}

func (a *RedisMQAdapter) getClient() *redis.Client {
	if a.client == nil {
		a.client = redis.NewClient(&redis.Options{
			Addr:     a.config.Addr,
			Password: a.config.Password,
			DB:       a.config.DB,
		})
	}
	return a.client
}

func (a *RedisMQAdapter) NewProducer() (Producer, error) {
	client := a.getClient()
	return &RedisProducer{
		client: client,
		logger: a.logger,
	}, nil
}

func (a *RedisMQAdapter) NewConsumer() (Consumer, error) {
	client := a.getClient()
	return &RedisConsumer{
		client: client,
		config: a.config,
		logger: a.logger,
	}, nil
}

// RedisProducer implements Producer interface for Redis
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
	// For Redis, we publish synchronously but it can be made async with goroutines
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

// RedisConsumer implements Consumer interface for Redis
type RedisConsumer struct {
	client    *redis.Client
	config    RedisConfig
	logger    Logger
	handlers  map[string]func(msg *Message) error
	stopCh    chan struct{}
	isRunning bool
}

func (c *RedisConsumer) Subscribe(ctx context.Context, topic string, group string, handler func(msg *Message) error) error {
	// Create consumer group if not exists
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
						// Acknowledge message
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
	close(c.stopCh)
	c.isRunning = false
	return nil
}

func (c *RedisConsumer) Close() error {
	if c.isRunning {
		close(c.stopCh)
	}
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

var DefaultMQFactory *MQFactory

func init() {
	DefaultMQFactory = NewMQFactory()
	DefaultMQFactory.Register(MQTypeRedis, NewRedisAdapter(RedisConfig{}))
}

type MQClient struct {
	adapter Adapter
	prod    Producer
	cons    Consumer
}

func NewMQClient(adapter Adapter) (*MQClient, error) {
	prod, err := adapter.NewProducer()
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	cons, err := adapter.NewConsumer()
	if err != nil {
		prod.Close()
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	return &MQClient{
		adapter: adapter,
		prod:    prod,
		cons:    cons,
	}, nil
}

func (c *MQClient) Publish(ctx context.Context, topic string, msg *Message) error {
	return c.prod.Publish(ctx, topic, msg)
}

func (c *MQClient) Subscribe(ctx context.Context, topic string, group string, handler func(msg *Message) error) error {
	return c.cons.Subscribe(ctx, topic, group, handler)
}

func (c *MQClient) Close() error {
	if c.prod != nil {
		c.prod.Close()
	}
	if c.cons != nil {
		c.cons.Close()
	}
	return nil
}

func (c *MQClient) Name() string {
	return c.adapter.Name()
}
