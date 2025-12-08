package proxy

import (
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// ConnectionPool 连接池
type ConnectionPool struct {
	connections map[string]*grpc.ClientConn
	mu          sync.RWMutex
}

// NewConnectionPool 创建连接池
func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{
		connections: make(map[string]*grpc.ClientConn),
	}
}

// GetConnection 获取或创建连接
func (p *ConnectionPool) GetConnection(target string) (*grpc.ClientConn, error) {
	// 先尝试读取已有连接
	p.mu.RLock()
	if conn, ok := p.connections[target]; ok {
		// 检查连接状态
		state := conn.GetState()
		if state != connectivity.Shutdown && state != connectivity.TransientFailure {
			p.mu.RUnlock()
			return conn, nil
		}
	}
	p.mu.RUnlock()

	// 创建新连接
	p.mu.Lock()
	defer p.mu.Unlock()

	// 双重检查
	if conn, ok := p.connections[target]; ok {
		state := conn.GetState()
		if state != connectivity.Shutdown && state != connectivity.TransientFailure {
			return conn, nil
		}
		// 关闭旧连接
		conn.Close()
		delete(p.connections, target)
	}

	// 创建新连接
	conn, err := grpc.Dial(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		return nil, err
	}

	p.connections[target] = conn
	return conn, nil
}

// Close 关闭所有连接
func (p *ConnectionPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for target, conn := range p.connections {
		conn.Close()
		delete(p.connections, target)
	}
}

// RemoveConnection 移除指定连接
func (p *ConnectionPool) RemoveConnection(target string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, ok := p.connections[target]; ok {
		conn.Close()
		delete(p.connections, target)
	}
}
