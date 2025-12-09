package proxy

import (
	"context"
	"fmt"
	"log"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	protopkg "github.com/heytom-labs/heytom-gateway/internal/proto"
	"github.com/heytom-labs/heytom-gateway/internal/registry"
)

// HTTPProxy HTTP to gRPC proxy
type HTTPProxy struct {
	protoLoader  *protopkg.DescriptorLoader
	registry     registry.Registry
	connPool     *ConnectionPool
	loadBalance  LoadBalancer
	fileResolver *protoregistry.Files
	msgCache     map[string]proto.Message // Message cache
	msgCacheMu   sync.RWMutex             // Message cache lock
}

// NewHTTPProxy 创建 HTTP 代理
func NewHTTPProxy(protoLoader *protopkg.DescriptorLoader, reg registry.Registry) (*HTTPProxy, error) {
	// 初始化文件注册表
	fileResolver := &protoregistry.Files{}

	// 注册所有 protobuf 文件描述符
	for _, fileProto := range protoLoader.GetFileDescriptorSet().File {
		fd, err := protodesc.NewFile(fileProto, fileResolver)
		if err != nil {
			return nil, fmt.Errorf("failed to create file descriptor: %w", err)
		}
		if err := fileResolver.RegisterFile(fd); err != nil {
			return nil, fmt.Errorf("failed to register file: %w", err)
		}
	}

	return &HTTPProxy{
		protoLoader:  protoLoader,
		registry:     reg,
		connPool:     NewConnectionPool(),
		loadBalance:  NewRoundRobinLoadBalancer(),
		fileResolver: fileResolver,
		msgCache:     make(map[string]proto.Message),
	}, nil
}

// ProxyHTTPRequest 代理 HTTP 请求到 gRPC
func (p *HTTPProxy) ProxyHTTPRequest(ctx context.Context, serviceName, methodName string, jsonBody []byte) ([]byte, error) {
	// 1. 查找方法描述符
	methodDesc := p.protoLoader.FindMethodDescriptor(serviceName, methodName)
	if methodDesc == nil {
		return nil, status.Errorf(codes.NotFound, "method not found: %s/%s", serviceName, methodName)
	}

	// 2. 获取输入消息的完整名称
	inputType := methodDesc.GetInputType()
	if inputType == "" {
		return nil, status.Errorf(codes.Internal, "method input type not specified")
	}

	// 3. 从 JSON 创建请求消息
	requestMsg, err := p.jsonToProtobuf(jsonBody, inputType)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to unmarshal request: %v", err)
	}

	// 4. 从注册中心发现服务实例
	instances, err := p.registry.Discover(ctx, serviceName)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to discover service %s: %v", serviceName, err)
	}

	if len(instances) == 0 {
		return nil, status.Errorf(codes.Unavailable, "no available instances for service: %s", serviceName)
	}

	// 5. 负载均衡选择实例
	instance := p.loadBalance.Select(instances)
	if instance == nil {
		return nil, status.Errorf(codes.Unavailable, "failed to select instance for service: %s", serviceName)
	}

	target := fmt.Sprintf("%s:%d", instance.Address, instance.Port)
	log.Printf("Proxying HTTP request to service: %s, method: %s, target: %s", serviceName, methodName, target)

	// 6. 获取或创建连接
	conn, err := p.connPool.GetConnection(target)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to connect to backend %s: %v", target, err)
	}

	// 7. 调用 gRPC 方法（一元 RPC）
	fullMethod := "/" + serviceName + "/" + methodName
	return p.invokeUnary(ctx, conn, fullMethod, requestMsg, methodDesc)
}

// invokeUnary 调用一元 RPC
func (p *HTTPProxy) invokeUnary(ctx context.Context, conn *grpc.ClientConn, fullMethod string, requestMsg proto.Message, methodDesc *descriptorpb.MethodDescriptorProto) ([]byte, error) {
	outputType := methodDesc.GetOutputType()
	if outputType == "" {
		return nil, status.Errorf(codes.Internal, "method output type not specified")
	}

	// 创建响应消息
	responseMsg, err := p.createDynamicMessage(outputType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create response message: %v", err)
	}

	// 执行 RPC
	clientCtx := metadata.NewOutgoingContext(ctx, metadata.MD{})
	err = conn.Invoke(clientCtx, fullMethod, requestMsg, responseMsg)
	if err != nil {
		return nil, err
	}

	// 将响应转换为 JSON
	return protojson.Marshal(responseMsg)
}

// jsonToProtobuf 将 JSON 转换为 Protobuf 消息
func (p *HTTPProxy) jsonToProtobuf(jsonData []byte, messageType string) (proto.Message, error) {
	msg, err := p.createDynamicMessage(messageType)
	if err != nil {
		return nil, err
	}

	if err := protojson.Unmarshal(jsonData, msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return msg, nil
}

// createDynamicMessage creates dynamic message from message type name
func (p *HTTPProxy) createDynamicMessage(messageType string) (proto.Message, error) {
	// Check cache
	p.msgCacheMu.RLock()
	if cached, ok := p.msgCache[messageType]; ok {
		p.msgCacheMu.RUnlock()
		// Return a new instance
		return proto.Clone(cached), nil
	}
	p.msgCacheMu.RUnlock()

	// Find full MessageDescriptor from registered files
	msgFullDesc := p.findFullMessageDescriptor(messageType)
	if msgFullDesc == nil {
		return nil, fmt.Errorf("message descriptor not found: %s", messageType)
	}

	// Create dynamic message
	msg := dynamicpb.NewMessage(msgFullDesc)
	p.msgCacheMu.Lock()
	p.msgCache[messageType] = msg
	p.msgCacheMu.Unlock()

	return msg, nil
}

// findFullMessageDescriptor finds the full message descriptor from the registry
func (p *HTTPProxy) findFullMessageDescriptor(fullName string) protoreflect.MessageDescriptor {
	// Iterate through all file descriptors to find the matching message
	for _, fileProto := range p.protoLoader.GetFileDescriptorSet().File {
		fd, err := protodesc.NewFile(fileProto, p.fileResolver)
		if err != nil {
			continue
		}

		// Search in messages
		msgs := fd.Messages()
		for i := 0; i < msgs.Len(); i++ {
			msg := msgs.Get(i)
			if string(msg.FullName()) == fullName {
				return msg // Return the message descriptor directly
			}
		}
	}
	return nil
}

// ClearMessageCache clears the message cache (for hot reload)
func (p *HTTPProxy) ClearMessageCache() {
	p.msgCacheMu.Lock()
	defer p.msgCacheMu.Unlock()
	p.msgCache = make(map[string]proto.Message)
}
