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
	"github.com/heytom-labs/heytom-gateway/internal/registry"
	_ "github.com/heytom-labs/heytom-gateway/internal/registry/consul" // 注册Consul实现
)

func main() {
	// 使用Wire初始化应用
	app, err := InitializeApp()
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	// 打印配置信息
	log.Printf("HTTP server port: %s", app.Config.Server.HTTPPort)
	log.Printf("gRPC server port: %s", app.Config.Server.GRPCPort)
	if app.Config.Registry.Enabled {
		log.Printf("Registry: %s at %s", app.Config.Registry.Type, app.Config.Registry.Address)
	}

	// 使用goroutine启动HTTP服务器
	go func() {
		log.Printf("HTTP server starting on %s", app.Config.Server.HTTPPort)
		if err := app.HTTPServer.Start(); err != nil {
			log.Fatalf("HTTP server failed to start: %v", err)
		}
	}()

	// 使用goroutine启动gRPC服务器
	go func() {
		log.Printf("gRPC server starting on %s", app.Config.Server.GRPCPort)
		if err := app.GRPCServer.Start(); err != nil {
			log.Fatalf("gRPC server failed to start: %v", err)
		}
	}()

	// 注册服务到注册中心
	if app.Registry != nil {
		if err := registerService(context.Background(), app.Registry, app.Config); err != nil {
			log.Fatalf("Failed to register service: %v", err)
		}
		log.Printf("Service registered: %s (ID: %s)", app.Config.Registry.ServiceName, app.Config.Registry.ServiceID)
	}

	// 等待中断信号以优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down servers...")

	// 创建关闭上下文，带超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 优雅关闭HTTP服务器
	if err := app.HTTPServer.Stop(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// 关闭gRPC服务器
	app.GRPCServer.Stop()

	// 从注册中心注销服务
	if app.Registry != nil {
		if err := app.Registry.Deregister(ctx, app.Config.Registry.ServiceID); err != nil {
			log.Printf("Failed to deregister service: %v", err)
		} else {
			log.Printf("Service deregistered: %s", app.Config.Registry.ServiceID)
		}
	}

	log.Println("Servers gracefully stopped")
}

// registerService 注册服务到注册中心
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
