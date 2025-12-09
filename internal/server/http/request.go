package http

import (
	"fmt"
	"strings"
)

// HTTPRequest HTTP 请求信息
type HTTPRequest struct {
	Tenant      string // 租户标识
	ServiceName string // 完整的 protobuf 服务名 (package.ServiceName)
	MethodName  string // 方法名
	Body        []byte // 请求体
}

// ParseHTTPRequest 解析 HTTP 请求路径
// 路径格式支持两种:
//  1. 带 tenant: /rpc/{tenant}/{full.proto.Service}/{Method}
//     例如: /rpc/tenantA/order.OrderService/Create
//  2. 不带 tenant: /rpc/{full.proto.Service}/{Method}
//     例如: /rpc/order.OrderService/Create
func ParseHTTPRequest(path string, body []byte) (*HTTPRequest, error) {
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")

	// 需要至少 3 部分:
	// rpc, service, method
	// 或 rpc, tenant, service, method
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid path format, expected /rpc/{service}/{method} or /rpc/{tenant}/{service}/{method}, got /%s", path)
	}

	if parts[0] != "rpc" {
		return nil, fmt.Errorf("invalid path, expected /rpc prefix, got /%s", parts[0])
	}

	methodName := parts[len(parts)-1]
	if methodName == "" {
		return nil, fmt.Errorf("method name cannot be empty")
	}

	serviceName := parts[len(parts)-2]
	if serviceName == "" {
		return nil, fmt.Errorf("service name cannot be empty")
	}
	var tenant string
	if len(parts) > 3 {
		tenant = parts[1]
	}

	return &HTTPRequest{
		Tenant:      tenant,
		ServiceName: serviceName,
		MethodName:  methodName,
		Body:        body,
	}, nil
}
