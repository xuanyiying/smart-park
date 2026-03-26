package biz

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
)

type EtcdDiscovery struct {
	client     *EtcdClient
	instances  map[string][]*ServiceInstance
	mu         sync.RWMutex
	logger     *log.Helper
	watchChans map[string]chan []*ServiceInstance
}

type EtcdClient struct {
	endpoints []string
	username  string
	password  string
	timeout   time.Duration
}

type EtcdConfig struct {
	Endpoints []string
	Username  string
	Password  string
	Timeout   time.Duration
}

func NewEtcdDiscovery(cfg *EtcdConfig, logger log.Logger) (*EtcdDiscovery, error) {
	client := &EtcdClient{
		endpoints: cfg.Endpoints,
		username:  cfg.Username,
		password:  cfg.Password,
		timeout:   cfg.Timeout,
	}

	if client.timeout == 0 {
		client.timeout = 5 * time.Second
	}

	return &EtcdDiscovery{
		client:     client,
		instances:  make(map[string][]*ServiceInstance),
		logger:     log.NewHelper(logger),
		watchChans: make(map[string]chan []*ServiceInstance),
	}, nil
}

func (d *EtcdDiscovery) Discover(ctx context.Context, serviceName string) ([]*ServiceInstance, error) {
	key := fmt.Sprintf("/services/%s", serviceName)

	resp, err := d.client.Get(ctx, key)
	if err != nil {
		d.logger.WithContext(ctx).Warnf("failed to discover service %s: %v", serviceName, err)
		return nil, err
	}

	var instances []*ServiceInstance
	for _, kv := range resp.Kvs {
		var inst ServiceInstance
		if err := kv.Unmarshal(&inst); err != nil {
			continue
		}
		instances = append(instances, &inst)
	}

	d.mu.Lock()
	d.instances[serviceName] = instances
	d.mu.Unlock()

	return instances, nil
}

func (d *EtcdDiscovery) Watch(ctx context.Context, serviceName string) (<-chan []*ServiceInstance, error) {
	ch := make(chan []*ServiceInstance, 10)

	go func() {
		defer close(ch)

		key := fmt.Sprintf("/services/%s", serviceName)

		go func() {
			instances, _ := d.Discover(ctx, serviceName)
			if instances != nil {
				ch <- instances
			}
		}()

		watchCh := d.client.Watch(ctx, key)

		for {
			select {
			case <-ctx.Done():
				return
			case resp, ok := <-watchCh:
				if !ok {
					return
				}
				var instances []*ServiceInstance
				for _, ev := range resp.Events {
					if ev.Type == EventTypePut {
						var inst ServiceInstance
						if err := ev.Kv.Unmarshal(&inst); err != nil {
							continue
						}
						instances = append(instances, &inst)
					}
				}
				if instances != nil {
					ch <- instances
				}
			}
		}
	}()

	return ch, nil
}

func (d *EtcdDiscovery) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, ch := range d.watchChans {
		close(ch)
	}
	d.watchChans = nil

	return nil
}

type EtcdResponse struct {
	Kvs    []*KeyValue
	Events []*EtcdEvent
}

type KeyValue struct {
	Key         []byte
	Value       []byte
	ModRevision int64
}

type EtcdEvent struct {
	Type EventType
	Kv   *KeyValue
}

type EventType int

const (
	EventTypePut    EventType = 0
	EventTypeDelete EventType = 1
)

func (kv *KeyValue) Unmarshal(v interface{}) error {
	return nil
}

func (d *EtcdClient) Get(ctx context.Context, key string) (*EtcdResponse, error) {
	return &EtcdResponse{}, nil
}

func (d *EtcdClient) Watch(ctx context.Context, key string) <-chan *EtcdResponse {
	ch := make(chan *EtcdResponse, 10)
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			default:
				time.Sleep(5 * time.Second)
			}
		}
	}()
	return ch
}

type EtcdRegistry struct {
	discovery *EtcdDiscovery
}

func NewEtcdRegistry(discovery *EtcdDiscovery) *EtcdRegistry {
	return &EtcdRegistry{discovery: discovery}
}

func (r *EtcdRegistry) GetService(ctx context.Context, name string) ([]*registry.ServiceInstance, error) {
	instances, err := r.discovery.Discover(ctx, name)
	if err != nil {
		return nil, err
	}

	var result []*registry.ServiceInstance
	for _, inst := range instances {
		result = append(result, &registry.ServiceInstance{
			ID:       inst.ID,
			Name:     inst.Name,
			Version:  "",
			Metadata: inst.Metadata,
			Endpoints: []string{
				fmt.Sprintf("tcp://%s:%d", inst.Address, inst.Port),
			},
		})
	}

	return result, nil
}

func (r *EtcdRegistry) Watch(ctx context.Context, name string) (<-chan []*registry.ServiceInstance, error) {
	ch := make(chan []*registry.ServiceInstance, 10)

	go func() {
		defer close(ch)

		watchCh, err := r.discovery.Watch(ctx, name)
		if err != nil {
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case instances, ok := <-watchCh:
				if !ok {
					return
				}
				var result []*registry.ServiceInstance
				for _, inst := range instances {
					result = append(result, &registry.ServiceInstance{
						ID:       inst.ID,
						Name:     inst.Name,
						Version:  "",
						Metadata: inst.Metadata,
						Endpoints: []string{
							fmt.Sprintf("tcp://%s:%d", inst.Address, inst.Port),
						},
					})
				}
				ch <- result
			}
		}
	}()

	return ch, nil
}
