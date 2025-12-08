package registry

import "context"

// ServiceInstance 服务实例信息
type ServiceInstance struct {
	ID       string            // 服务实例唯一ID
	Name     string            // 服务名称
	Version  string            // 服务版本
	Address  string            // 服务地址
	Port     int               // 服务端口
	Metadata map[string]string // 元数据
	Tags     []string          // 标签
}

// Registry 服务注册发现接口
type Registry interface {
	// Register 注册服务实例
	Register(ctx context.Context, instance *ServiceInstance) error

	// Deregister 注销服务实例
	Deregister(ctx context.Context, instanceID string) error

	// Discover 发现服务实例列表
	Discover(ctx context.Context, serviceName string) ([]*ServiceInstance, error)

	// Watch 监听服务变化
	Watch(ctx context.Context, serviceName string) (Watcher, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context, instanceID string) error
}

// Watcher 服务监听器
type Watcher interface {
	// Next 获取下一个服务变化事件
	Next() ([]*ServiceInstance, error)

	// Stop 停止监听
	Stop() error
}
