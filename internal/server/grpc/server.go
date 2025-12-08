package grpc

import (
	"fmt"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/heytom-labs/heytom-gateway/internal/proxy"
	"github.com/heytom-labs/heytom-gateway/internal/registry"
)

// Server gRPC服务器结构体
type Server struct {
	grpcServer *grpc.Server
	address    string
	proxy      *proxy.GRPCProxy
}

// New 创建gRPC服务器实例
func New(address string) *Server {
	return &Server{
		address: address,
	}
}

// SetRegistry 设置注册中心（用于依赖注入）
func (s *Server) SetRegistry(reg registry.Registry) {
	if reg != nil {
		s.proxy = proxy.NewGRPCProxy(reg)
	}
}

// Initialize 初始化gRPC服务器
func (s *Server) Initialize() {
	// 创建gRPC服务器实例，设置未知服务处理器
	s.grpcServer = grpc.NewServer(
		grpc.UnknownServiceHandler(s.handleUnknownService),
	)

	// 注册健康检查服务
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(s.grpcServer, healthServer)
}

// handleUnknownService 处理未知服务的请求（动态转发）
func (s *Server) handleUnknownService(srv any, stream grpc.ServerStream) error {
	// 1. 解析服务名和方法名
	serviceName, methodName, err := ParseServiceAndMethod(stream)

	fmt.Printf("parse serviceName:%s,methodName:%s", serviceName, methodName)
	if err != nil {
		return fmt.Errorf("parse service method error: %w", err)
	}

	// 2. 检查是否配置了代理
	if s.proxy == nil {
		return fmt.Errorf("proxy not configured, cannot forward request to service: %s", serviceName)
	}

	// 3. 使用代理转发请求
	ctx := stream.Context()
	return s.proxy.ProxyStream(ctx, serviceName, methodName, stream)
}

// ParseServiceAndMethod 从流中解析服务名和方法名
func ParseServiceAndMethod(stream grpc.ServerStream) (serviceName, methodName string, err error) {
	// 获取完整方法名，格式: /package.Service/Method
	fullMethod, ok := grpc.MethodFromServerStream(stream)
	fmt.Println(fullMethod)
	if !ok {
		return "", "", fmt.Errorf("failed to get method from stream")
	}

	// 移除开头的斜杠
	fullMethod = strings.TrimPrefix(fullMethod, "/")

	// 找到第一个斜杠
	idx := strings.Index(fullMethod, "/")
	if idx == -1 {
		// 没有找到斜杠，整个字符串作为服务方法
		return "", fullMethod, nil
	}

	// 分割：第一个部分作为路由前缀，剩余部分作为服务方法
	serviceName = fullMethod[:idx]
	methodName = fullMethod[idx+1:]

	return serviceName, methodName, nil
}

// Start 启动gRPC服务器
func (s *Server) Start() error {
	// 如果还没有初始化，先初始化
	if s.grpcServer == nil {
		s.Initialize()
	}

	lis, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	return s.grpcServer.Serve(lis)
}

// Stop 停止gRPC服务器
func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}

// GetGRPCServer 获取底层gRPC服务器实例
// 用于注册其他服务
func (s *Server) GetGRPCServer() *grpc.Server {
	if s.grpcServer == nil {
		s.Initialize()
	}
	return s.grpcServer
}
