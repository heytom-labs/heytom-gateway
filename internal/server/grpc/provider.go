package grpc

import (
	"github.com/google/wire"
	"github.com/heytom-labs/heytom-gateway/internal/config"
	"github.com/heytom-labs/heytom-gateway/internal/registry"
)

// ProviderSet gRPC服务器Provider集合
var ProviderSet = wire.NewSet(
	ProvideServer,
)

// ProvideServer 提供gRPC服务器实例
func ProvideServer(cfg *config.Config, reg registry.Registry) *Server {
	srv := New(cfg.Server.GRPCPort)
	srv.SetRegistry(reg)
	return srv
}
