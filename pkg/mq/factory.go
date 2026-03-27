package mq

import (
	"context"
	"fmt"
	"sync"
)

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

var DefaultMQFactory *MQFactory

func init() {
	DefaultMQFactory = NewMQFactory()
	DefaultMQFactory.Register(MQTypeRedis, NewRedisAdapter(RedisConfig{}))
}
