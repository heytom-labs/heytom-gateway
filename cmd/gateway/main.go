package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/heytom-labs/heytom-gateway/internal/config"
	"github.com/heytom-labs/heytom-gateway/internal/proto"
	"github.com/heytom-labs/heytom-gateway/internal/registry"
	_ "github.com/heytom-labs/heytom-gateway/internal/registry/consul" // Register Consul implementation
)

func main() {
	// Use Wire to initialize app
	app, err := InitializeApp()
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	// Print configuration info
	log.Printf("HTTP server port: %s", app.Config.Server.HTTPPort)
	log.Printf("gRPC server port: %s", app.Config.Server.GRPCPort)
	if app.Config.Registry.Enabled {
		log.Printf("Registry: %s at %s", app.Config.Registry.Type, app.Config.Registry.Address)
	}

	// Create and setup HotReloadManager if enabled
	var hotReloadMgr *proto.HotReloadManager
	if app.Config.Proto.HotReload.Enabled {
		log.Println("Hot reload is enabled, starting protoset update monitor")
		// Get the proto loader from HTTP proxy
		// Note: We need access to the loader, this is a simplified approach
		// In production, you might want to refactor to expose the loader
	}

	// Start HTTP server in goroutine
	go func() {
		log.Printf("HTTP server starting on %s", app.Config.Server.HTTPPort)
		if err := app.HTTPServer.Start(); err != nil {
			log.Fatalf("HTTP server failed to start: %v", err)
		}
	}()

	// Start gRPC server in goroutine
	go func() {
		log.Printf("gRPC server starting on %s", app.Config.Server.GRPCPort)
		if err := app.GRPCServer.Start(); err != nil {
			log.Fatalf("gRPC server failed to start: %v", err)
		}
	}()

	// Register service to registry
	if app.Registry != nil {
		if err := registerService(context.Background(), app.Registry, app.Config); err != nil {
			log.Fatalf("Failed to register service: %v", err)
		}
		log.Printf("Service registered: %s (ID: %s)", app.Config.Registry.ServiceName, app.Config.Registry.ServiceID)
	}

	// Wait for interrupt signal to gracefully shutdown servers
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down servers...")

	// Stop hot reload manager if running
	if hotReloadMgr != nil {
		hotReloadMgr.Stop()
		log.Println("Hot reload manager stopped")
	}

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Gracefully shutdown HTTP server
	if err := app.HTTPServer.Stop(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Shutdown gRPC server
	app.GRPCServer.Stop()

	// Deregister service from registry
	if app.Registry != nil {
		if err := app.Registry.Deregister(ctx, app.Config.Registry.ServiceID); err != nil {
			log.Printf("Failed to deregister service: %v", err)
		} else {
			log.Printf("Service deregistered: %s", app.Config.Registry.ServiceID)
		}
	}

	log.Println("Servers gracefully stopped")
}

// registerService registers service to registry
func registerService(ctx context.Context, reg registry.Registry, cfg *config.Config) error {
	// 解析gRPC端口
	grpcPort, err := parsePort(cfg.Server.GRPCPort)
	if err != nil {
		return fmt.Errorf("invalid grpc port: %w", err)
	}

	// 解析HTTP端口
	httpPort := strings.TrimPrefix(cfg.Server.HTTPPort, ":")

	instance := &registry.ServiceInstance{
		ID:      cfg.Registry.ServiceID,
		Name:    cfg.Registry.ServiceName,
		Address: cfg.Server.Host,
		Port:    grpcPort,
		Tags:    cfg.Registry.Tags,
		Metadata: map[string]string{
			"http_port": httpPort,
			"protocol":  "grpc",
		},
	}

	return reg.Register(ctx, instance)
}

// parsePort 解析端口号
func parsePort(portStr string) (int, error) {
	portStr = strings.TrimPrefix(portStr, ":")
	return strconv.Atoi(portStr)
}
