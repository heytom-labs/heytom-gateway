package proto

import (
	"github.com/google/wire"
	"github.com/heytom-labs/heytom-gateway/internal/config"
)

// ProviderSet gRPC服务器Provider集合
var ProviderSet = wire.NewSet(
	ProvideHotReloadManager,
)

// ProvideServer 提供gRPC服务器实例
func ProvideHotReloadManager(loader *DescriptorLoader,
	cfg *config.ProtoHotReloadConfig,
	protosets []config.ProtoSetInfo,
) *HotReloadManager {
	srv := NewHotReloadManager(loader, cfg, protosets)
	return srv
}
