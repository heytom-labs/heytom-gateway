package config

import (
	"log"

	"github.com/google/wire"
)

// ProviderSet 配置Provider集合
var ProviderSet = wire.NewSet(
	ProvideConfig,
)

// ProvideConfig 提供配置实例
func ProvideConfig() *Config {
	cfg, err := LoadConfig("configs/config.json")
	if err != nil {
		log.Printf("Failed to load config: %v, using default config", err)
		return GetDefaultConfig()
	}
	return cfg
}
