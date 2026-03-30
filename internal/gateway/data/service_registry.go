package data

import (
	"context"
	"time"

	etcdreg "github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/log"
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
func (r *ServiceRegistry) GetRegistry() *etcdreg.Registry {
	reg := etcdreg.New(r.client)
	return reg
}

// Close 关闭连接
func (r *ServiceRegistry) Close() error {
	return r.client.Close()
}

// ServiceHealthCheck 服务健康检查
func (r *ServiceRegistry) ServiceHealthCheck(ctx context.Context, serviceName string) (bool, error) {
	// 通过验证 ETCD 中是否至少有一个对应服务的实例活跃，来判断健康状态
	prefix := "/microservices/" + serviceName
	resp, err := r.client.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithLimit(1))
	if err != nil {
		r.log.Errorf("health check error for %s: %v", serviceName, err)
		return false, err
	}
	return len(resp.Kvs) > 0, nil
}
