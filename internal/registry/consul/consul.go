package consul

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/heytom-labs/heytom-gateway/internal/registry"
)

// Config Consul配置
type Config struct {
	Address            string        // Consul地址
	Scheme             string        // http或https
	Token              string        // ACL Token
	Datacenter         string        // 数据中心
	WaitTime           time.Duration // 长轮询等待时间
	HealthCheckTimeout time.Duration // 健康检查超时时间
	HealthCheckTTL     time.Duration // 健康检查TTL
}

// Registry Consul注册中心实现
type Registry struct {
	client *api.Client
	config *Config
}

// NewRegistry 创建Consul注册中心
func NewRegistry(config *Config) (*Registry, error) {
	if config == nil {
		config = &Config{
			Address:            "127.0.0.1:8500",
			Scheme:             "http",
			WaitTime:           time.Second * 30,
			HealthCheckTimeout: time.Second * 5,
			HealthCheckTTL:     time.Second * 15,
		}
	}

	consulConfig := api.DefaultConfig()
	consulConfig.Address = config.Address
	consulConfig.Scheme = config.Scheme
	consulConfig.Token = config.Token
	consulConfig.Datacenter = config.Datacenter
	consulConfig.WaitTime = config.WaitTime

	client, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}

	return &Registry{
		client: client,
		config: config,
	}, nil
}

// Register 注册服务实例
func (r *Registry) Register(ctx context.Context, instance *registry.ServiceInstance) error {
	if instance == nil {
		return fmt.Errorf("service instance is nil")
	}

	// 构建健康检查
	check := &api.AgentServiceCheck{
		CheckID:                        instance.ID,
		TTL:                            r.config.HealthCheckTTL.String(),
		Timeout:                        r.config.HealthCheckTimeout.String(),
		DeregisterCriticalServiceAfter: "30s",
	}

	// 如果有HTTP端口，使用HTTP健康检查
	if instance.Metadata != nil && instance.Metadata["http_port"] != "" {
		httpPort := instance.Metadata["http_port"]
		check.HTTP = fmt.Sprintf("http://%s:%s/health", instance.Address, httpPort)
		check.Interval = "10s"
		check.TTL = ""
	}

	// 构建服务注册信息
	registration := &api.AgentServiceRegistration{
		ID:      instance.ID,
		Name:    instance.Name,
		Address: instance.Address,
		Port:    instance.Port,
		Tags:    instance.Tags,
		Meta:    instance.Metadata,
		Check:   check,
	}

	// 注册服务
	if err := r.client.Agent().ServiceRegister(registration); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	// 如果使用TTL健康检查，启动定期更新
	if check.TTL != "" {
		go r.keepAlive(instance.ID)
	}

	return nil
}

// Deregister 注销服务实例
func (r *Registry) Deregister(ctx context.Context, instanceID string) error {
	if err := r.client.Agent().ServiceDeregister(instanceID); err != nil {
		return fmt.Errorf("failed to deregister service: %w", err)
	}
	return nil
}

// Discover 发现服务实例列表
func (r *Registry) Discover(ctx context.Context, serviceName string) ([]*registry.ServiceInstance, error) {
	services, _, err := r.client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to discover service: %w", err)
	}

	instances := make([]*registry.ServiceInstance, 0, len(services))
	for _, service := range services {
		instance := &registry.ServiceInstance{
			ID:       service.Service.ID,
			Name:     service.Service.Service,
			Address:  service.Service.Address,
			Port:     service.Service.Port,
			Tags:     service.Service.Tags,
			Metadata: service.Service.Meta,
		}
		instances = append(instances, instance)
	}

	return instances, nil
}

// Watch 监听服务变化
func (r *Registry) Watch(ctx context.Context, serviceName string) (registry.Watcher, error) {
	return newWatcher(ctx, r.client, serviceName)
}

// HealthCheck 健康检查
func (r *Registry) HealthCheck(ctx context.Context, instanceID string) error {
	return r.client.Agent().UpdateTTL(instanceID, "", api.HealthPassing)
}

// keepAlive 保持服务健康状态
func (r *Registry) keepAlive(instanceID string) {
	ticker := time.NewTicker(r.config.HealthCheckTTL / 2)
	defer ticker.Stop()

	for range ticker.C {
		if err := r.client.Agent().UpdateTTL(instanceID, "", api.HealthPassing); err != nil {
			// 如果更新失败，服务可能已被注销，退出
			return
		}
	}
}

// watcher Consul服务监听器
type watcher struct {
	client      *api.Client
	serviceName string
	ctx         context.Context
	cancel      context.CancelFunc
	eventChan   chan []*registry.ServiceInstance
	errChan     chan error
}

// newWatcher 创建服务监听器
func newWatcher(ctx context.Context, client *api.Client, serviceName string) (*watcher, error) {
	watchCtx, cancel := context.WithCancel(ctx)

	w := &watcher{
		client:      client,
		serviceName: serviceName,
		ctx:         watchCtx,
		cancel:      cancel,
		eventChan:   make(chan []*registry.ServiceInstance, 1),
		errChan:     make(chan error, 1),
	}

	go w.watch()

	return w, nil
}

// watch 监听服务变化
func (w *watcher) watch() {
	var lastIndex uint64

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		queryOptions := &api.QueryOptions{
			WaitIndex: lastIndex,
			WaitTime:  time.Second * 30,
		}

		services, meta, err := w.client.Health().Service(w.serviceName, "", true, queryOptions)
		if err != nil {
			select {
			case w.errChan <- err:
			case <-w.ctx.Done():
				return
			}
			time.Sleep(time.Second)
			continue
		}

		// 如果索引没有变化，继续等待
		if lastIndex == meta.LastIndex {
			continue
		}

		lastIndex = meta.LastIndex

		// 转换为ServiceInstance
		instances := make([]*registry.ServiceInstance, 0, len(services))
		for _, service := range services {
			instance := &registry.ServiceInstance{
				ID:       service.Service.ID,
				Name:     service.Service.Service,
				Address:  service.Service.Address,
				Port:     service.Service.Port,
				Tags:     service.Service.Tags,
				Metadata: service.Service.Meta,
			}
			instances = append(instances, instance)
		}

		select {
		case w.eventChan <- instances:
		case <-w.ctx.Done():
			return
		}
	}
}

// Next 获取下一个服务变化事件
func (w *watcher) Next() ([]*registry.ServiceInstance, error) {
	select {
	case instances := <-w.eventChan:
		return instances, nil
	case err := <-w.errChan:
		return nil, err
	case <-w.ctx.Done():
		return nil, w.ctx.Err()
	}
}

// Stop 停止监听
func (w *watcher) Stop() error {
	w.cancel()
	return nil
}

// GetServiceAddress 获取服务地址（负载均衡）
func GetServiceAddress(instances []*registry.ServiceInstance) string {
	if len(instances) == 0 {
		return ""
	}
	// 简单轮询，实际应该使用更复杂的负载均衡策略
	instance := instances[0]
	return fmt.Sprintf("%s:%s", instance.Address, strconv.Itoa(instance.Port))
}
