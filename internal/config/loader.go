package config

import (
	"encoding/json"
	"os"
)

// LoadConfig 从文件加载配置
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// GetDefaultConfig 返回默认配置
func GetDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			HTTPPort: ":8080",
			GRPCPort: ":9091",
			Host:     "127.0.0.1",
		},
		Registry: RegistryConfig{
			Enabled:            false,
			Type:               "consul",
			Address:            "127.0.0.1:8500",
			ServiceName:        "heytom-gateway",
			ServiceID:          "heytom-gateway-1",
			Tags:               []string{"gateway", "api"},
			HealthCheckTimeout: 5000000000,  // 5s
			HealthCheckTTL:     15000000000, // 15s
		},
	}
}
