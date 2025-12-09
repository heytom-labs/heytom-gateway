package http

import (
	"github.com/google/wire"
	"github.com/heytom-labs/heytom-gateway/internal/config"
	"github.com/heytom-labs/heytom-gateway/internal/proto"
	"github.com/heytom-labs/heytom-gateway/internal/proxy"
	"github.com/heytom-labs/heytom-gateway/internal/registry"
)

// ProviderSet HTTP server provider set
var ProviderSet = wire.NewSet(
	ProvideServer,
	ProvideHTTPProxy,
)

// ProvideServer provides HTTP server instance
func ProvideServer(cfg *config.Config, httpProxy *proxy.HTTPProxy) *Server {
	server := New(cfg.Server.HTTPPort)
	server.SetHTTPProxy(httpProxy)
	return server
}

// ProvideHTTPProxy provides HTTP proxy instance
func ProvideHTTPProxy(cfg *config.Config, reg registry.Registry) (*proxy.HTTPProxy, error) {
	if !cfg.Registry.Enabled {
		return nil, nil
	}

	// Load protoset
	protoLoader, err := proto.NewDescriptorLoader(cfg.Proto.ProtoSetPath)
	if err != nil {
		return nil, err
	}

	// Load additional protosets if configured
	for _, ps := range cfg.Proto.ProtoSets {
		if ps.Path != "" {
			if err := protoLoader.LoadProtoset(ps.Path); err != nil {
				return nil, err
			}
		}
	}

	// Create HTTP proxy
	httpProxy, err := proxy.NewHTTPProxy(protoLoader, reg)
	if err != nil {
		return nil, err
	}

	// Start hot reload if enabled
	if cfg.Proto.HotReload.Enabled {
		hotReloadMgr := proto.NewHotReloadManager(
			protoLoader,
			&cfg.Proto.HotReload,
			cfg.Proto.ProtoSets,
		)

		// Set message cache clear callback
		hotReloadMgr.SetMessageCacheClearFunc(func() {
			httpProxy.ClearMessageCache()
		})

		// TODO: Start hot reload in app initialization
		// This should be done in the app.go main function
		// hotReloadMgr.Start(context.Background())
	}

	return httpProxy, nil
}
