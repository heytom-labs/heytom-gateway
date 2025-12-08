package http

import (
	"context"
	"fmt"
	"net/http"
)

// Server HTTP服务器结构体
type Server struct {
	httpServer *http.Server
}

// New 创建HTTP服务器实例
func New(address string) *Server {
	mux := http.NewServeMux()

	// 默认路由
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Hello from HTTP Server!")
	})

	// 健康检查路由
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "HTTP Server is healthy")
	})

	return &Server{
		httpServer: &http.Server{
			Addr:    address,
			Handler: mux,
		},
	}
}

// Start 启动HTTP服务器
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// StartTLS 启动HTTPS服务器
func (s *Server) StartTLS(certFile, keyFile string) error {
	return s.httpServer.ListenAndServeTLS(certFile, keyFile)
}

// Stop 停止HTTP服务器
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
