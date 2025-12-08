package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/heytom-labs/heytom-gateway/internal/registry"
)

// GRPCProxy gRPC代理
type GRPCProxy struct {
	registry    registry.Registry
	connPool    *ConnectionPool
	loadBalance LoadBalancer
}

// NewGRPCProxy 创建gRPC代理
func NewGRPCProxy(reg registry.Registry) *GRPCProxy {
	return &GRPCProxy{
		registry:    reg,
		connPool:    NewConnectionPool(),
		loadBalance: NewRoundRobinLoadBalancer(),
	}
}

// ProxyStream 代理流式请求
func (p *GRPCProxy) ProxyStream(ctx context.Context, serviceName, fullMethod string, stream grpc.ServerStream) error {
	// 1. 从注册中心发现服务实例
	instances, err := p.registry.Discover(ctx, serviceName)
	if err != nil {
		return status.Errorf(codes.Unavailable, "failed to discover service %s: %v", serviceName, err)
	}

	if len(instances) == 0 {
		return status.Errorf(codes.Unavailable, "no available instances for service: %s", serviceName)
	}

	// 2. 负载均衡选择实例
	instance := p.loadBalance.Select(instances)
	if instance == nil {
		return status.Errorf(codes.Unavailable, "failed to select instance for service: %s", serviceName)
	}

	target := fmt.Sprintf("%s:%d", instance.Address, instance.Port)
	log.Printf("Proxying request to service: %s, method: %s, target: %s", serviceName, fullMethod, target)

	// 3. 获取或创建到后端服务的连接
	conn, err := p.connPool.GetConnection(target)
	if err != nil {
		return status.Errorf(codes.Unavailable, "failed to connect to backend %s: %v", target, err)
	}

	methodNams := strings.Split(fullMethod, "/")

	// 5. 创建客户端流
	clientCtx := metadata.NewOutgoingContext(ctx, metadata.MD{})
	clientStream, err := conn.NewStream(clientCtx, &grpc.StreamDesc{
		StreamName:    methodNams[1],
		ServerStreams: true,
		ClientStreams: true,
	}, fullMethod)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create client stream: %v", err)
	}

	// 6. 双向转发流数据
	return p.forwardStream(stream, clientStream)
}

// forwardStream 双向转发流数据
func (p *GRPCProxy) forwardStream(serverStream grpc.ServerStream, clientStream grpc.ClientStream) error {
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// 服务端 -> 客户端
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// 从服务端接收消息
			msg := &DynamicMessage{}
			if err := serverStream.RecvMsg(msg); err != nil {
				if err == io.EOF {
					clientStream.CloseSend()
					return
				}
				errChan <- fmt.Errorf("server recv error: %w", err)
				return
			}

			// 发送到客户端
			if err := clientStream.SendMsg(msg); err != nil {
				errChan <- fmt.Errorf("client send error: %w", err)
				return
			}
		}
	}()

	// 客户端 -> 服务端
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// 从客户端接收消息
			msg := &DynamicMessage{}
			if err := clientStream.RecvMsg(msg); err != nil {
				if err == io.EOF {
					return
				}
				errChan <- fmt.Errorf("client recv error: %w", err)
				return
			}

			// 发送到服务端
			if err := serverStream.SendMsg(msg); err != nil {
				errChan <- fmt.Errorf("server send error: %w", err)
				return
			}
		}
	}()

	// 等待转发完成或出错
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// 返回第一个错误
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// DynamicMessage 动态消息类型，用于转发任意protobuf消息
type DynamicMessage struct {
	data []byte
}

// Reset 实现 proto.Message 接口
func (m *DynamicMessage) Reset() {
	m.data = nil
}

// String 实现 proto.Message 接口
func (m *DynamicMessage) String() string {
	return string(m.data)
}

// ProtoMessage 实现 proto.Message 接口
func (m *DynamicMessage) ProtoMessage() {}

// Marshal 序列化
func (m *DynamicMessage) Marshal() ([]byte, error) {
	return m.data, nil
}

// Unmarshal 反序列化
func (m *DynamicMessage) Unmarshal(data []byte) error {
	m.data = data
	return nil
}
