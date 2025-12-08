package registry

import (
	"fmt"

	"github.com/google/wire"
	"github.com/heytom-labs/heytom-gateway/internal/config"
)

// ProviderSet 注册中心Provider集合
var ProviderSet = wire.NewSet(
	ProvideRegistry,
)

// RegistryFactory 注册中心工厂函数类型
type RegistryFactory func(*config.Config) (Registry, error)

// registryFactories 注册中心工厂映射
var registryFactories = make(map[string]RegistryFactory)

// RegisterFactory 注册注册中心工厂
func RegisterFactory(registryType string, factory RegistryFactory) {
	registryFactories[registryType] = factory
}

// ProvideRegistry 提供注册中心实例
func ProvideRegistry(cfg *config.Config) (Registry, error) {
	if !cfg.Registry.Enabled {
		return nil, nil
	}

	factory, ok := registryFactories[cfg.Registry.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported registry type: %s", cfg.Registry.Type)
	}

	return factory(cfg)
}
