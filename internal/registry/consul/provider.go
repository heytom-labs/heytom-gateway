package consul

import (
	"github.com/heytom-labs/heytom-gateway/internal/config"
	"github.com/heytom-labs/heytom-gateway/internal/registry"
)

func init() {
	// 注册Consul工厂到注册中心
	registry.RegisterFactory("consul", NewConsulRegistry)
}

// NewConsulRegistry 创建Consul注册中心实例
func NewConsulRegistry(cfg *config.Config) (registry.Registry, error) {
	return NewRegistry(&Config{
		Address:            cfg.Registry.Address,
		Scheme:             "http",
		HealthCheckTimeout: cfg.Registry.HealthCheckTimeout,
		HealthCheckTTL:     cfg.Registry.HealthCheckTTL,
	})
}
