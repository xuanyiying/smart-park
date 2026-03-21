package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// ServiceRegistry 服务注册中心
type ServiceRegistry struct {
	client *clientv3.Client
	log    *log.Helper
}

// NewServiceRegistry 创建服务注册中心
func NewServiceRegistry(etcdEndpoints []string, logger log.Logger) (*ServiceRegistry, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   etcdEndpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &ServiceRegistry{
		client: client,
		log:    log.NewHelper(logger),
	}, nil
}

// GetRegistry 获取 Kratos registry
func (r *ServiceRegistry) GetRegistry() registry.Registry {
	return registry.NewEtcd(r.client)
}

// Close 关闭连接
func (r *ServiceRegistry) Close() error {
	return r.client.Close()
}

// ServiceHealthCheck 服务健康检查
func (r *ServiceRegistry) ServiceHealthCheck(ctx context.Context, serviceName string) (bool, error) {
	// TODO: 实现健康检查逻辑
	return true, nil
}
