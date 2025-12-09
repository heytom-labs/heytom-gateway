package http

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/heytom-labs/heytom-gateway/internal/proxy"
)

// Server HTTP服务器结构体
type Server struct {
	httpServer *http.Server
	httpProxy  *proxy.HTTPProxy
}

// New 创建HTTP服务器实例
func New(address string) *Server {
	mux := http.NewServeMux()

	return &Server{
		httpServer: &http.Server{
			Addr:    address,
			Handler: mux,
		},
	}
}

// SetHTTPProxy 设置HTTP代理器（依赖注入）
func (s *Server) SetHTTPProxy(proxy *proxy.HTTPProxy) {
	s.httpProxy = proxy
}

// Start 启动HTTP服务器
func (s *Server) Start() error {
	// 定义库底路由处理器
	s.httpServer.Handler = http.HandlerFunc(s.handleRequest)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "HTTP Server is healthy")
	})
	mux.HandleFunc("/", s.handleRequest)
	s.httpServer.Handler = mux

	return s.httpServer.ListenAndServe()
}

// handleRequest 处理HTTP请求
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/health" {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "HTTP Server is healthy")
		return
	}
	if s.httpProxy == nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "HTTP proxy not configured")
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "Only POST method is allowed")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to read request body: %v", err)
		return
	}
	defer r.Body.Close()

	// 解析HTTP请求
	httpReq, err := ParseHTTPRequest(r.URL.Path, body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid request: %v", err)
		return
	}

	// 调用HTTP代理
	response, err := s.httpProxy.ProxyHTTPRequest(r.Context(), httpReq.ServiceName, httpReq.MethodName, body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "RPC call failed: %v", err)
		return
	}

	// 返回响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

// StartTLS 启动HTTPS服务器
func (s *Server) StartTLS(certFile, keyFile string) error {
	// 定义库底路由处理器
	s.httpServer.Handler = http.HandlerFunc(s.handleRequest)
	return s.httpServer.ListenAndServeTLS(certFile, keyFile)
}

// Stop 停止HTTP服务器
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
