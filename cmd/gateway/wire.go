//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/heytom-labs/heytom-gateway/internal/config"
	"github.com/heytom-labs/heytom-gateway/internal/proto"
	"github.com/heytom-labs/heytom-gateway/internal/registry"
	"github.com/heytom-labs/heytom-gateway/internal/server/grpc"
	"github.com/heytom-labs/heytom-gateway/internal/server/http"
)

// InitializeApp 初始化应用程序
func InitializeApp() (*App, error) {
	wire.Build(
		config.ProviderSet,
		http.ProviderSet,
		grpc.ProviderSet,
		registry.ProviderSet,
		proto.ProviderSet,
		wire.Struct(new(App), "*"),
	)
	return &App{}, nil
}
