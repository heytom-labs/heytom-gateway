package http

import (
	"github.com/google/wire"
	"github.com/heytom-labs/heytom-gateway/internal/config"
)

// ProviderSet HTTP服务器Provider集合
var ProviderSet = wire.NewSet(
	ProvideServer,
)

// ProvideServer 提供HTTP服务器实例
func ProvideServer(cfg *config.Config) *Server {
	return New(cfg.Server.HTTPPort)
}
