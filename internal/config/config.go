package config

import (
	"time"
)

// Config 应用配置结构
type Config struct {
	Server   ServerConfig   `json:"server"`
	Registry RegistryConfig `json:"registry"`
	Proto    ProtoConfig    `json:"proto"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	HTTPPort string `json:"http_port"`
	GRPCPort string `json:"grpc_port"`
	Host     string `json:"host"` // 服务主机地址
}

// RegistryConfig 注册中心配置
type RegistryConfig struct {
	Enabled            bool          `json:"enabled"`              // 是否启用注册中心
	Type               string        `json:"type"`                 // 注册中心类型: consul, etcd, nacos
	Address            string        `json:"address"`              // 注册中心地址
	ServiceName        string        `json:"service_name"`         // 服务名称
	ServiceID          string        `json:"service_id"`           // 服务实例ID
	Tags               []string      `json:"tags"`                 // 服务标签
	HealthCheckTimeout time.Duration `json:"health_check_timeout"` // 健康检查超时
	HealthCheckTTL     time.Duration `json:"health_check_ttl"`     // 健康检查TTL
}

// ProtoConfig Protobuf 配置
type ProtoConfig struct {
	ProtoSetPath string               `json:"protoset_path"` // 主 protoset 文件路径
	ProtoSets    []ProtoSetInfo       `json:"protosets"`     // 不同服务的 protoset 列表
	HotReload    ProtoHotReloadConfig `json:"hot_reload"`    // 热更新配置
}

// ProtoSetInfo single protoset information
type ProtoSetInfo struct {
	ServiceName string `json:"service_name"` // Microservice name
	Path        string `json:"path"`         // Local file path
	URL         string `json:"url"`          // Download URL (artifact repository)
}

// ProtoHotReloadConfig hot reload configuration
type ProtoHotReloadConfig struct {
	Enabled     bool   `json:"enabled"`      // Enable hot reload
	CheckPeriod int64  `json:"check_period"` // Check period (seconds)
	AuthToken   string `json:"auth_token"`   // Auth token for artifact repository
}
