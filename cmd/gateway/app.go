package main

import (
	"github.com/heytom-labs/heytom-gateway/internal/config"
	"github.com/heytom-labs/heytom-gateway/internal/proto"
	"github.com/heytom-labs/heytom-gateway/internal/registry"
	"github.com/heytom-labs/heytom-gateway/internal/server/grpc"
	"github.com/heytom-labs/heytom-gateway/internal/server/http"
)

// App Application structure
type App struct {
	Config           *config.Config
	HTTPServer       *http.Server
	GRPCServer       *grpc.Server
	Registry         registry.Registry
	HotReloadManager *proto.HotReloadManager // Optional hot reload manager
}
