# HeyTom Gateway

一个基于 Go 语言的高性能 API 网关，支持 HTTP 到 gRPC 的协议转换、服务发现、负载均衡和动态路由。

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-blue)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

<img width="1888" height="1036" alt="image" src="https://github.com/user-attachments/assets/cf3f074f-fcbf-4e0d-83ff-07adbc67a616" />

## 核心特性

### 🚀 协议支持
- **HTTP/REST API** - 接收 HTTP 请求并转换为 gRPC 调用
- **gRPC** - 原生 gRPC 协议支持，透明代理转发
- **双向流** - 支持 gRPC 双向流式传输

### 🔍 服务发现
- **Consul 集成** - 自动服务注册与发现
- **健康检查** - 实时监控后端服务健康状态
- **动态路由** - 根据服务名自动发现并路由到后端实例

### ⚖️ 负载均衡
- **轮询（Round Robin）** - 默认策略，均匀分配请求
- **随机（Random）** - 随机选择后端实例
- **加权（Weighted）** - 基于权重的智能分配

### 🔌 连接管理
- **连接池** - 自动管理和复用后端连接
- **健康检测** - 自动检测并移除失效连接
- **优雅关闭** - 支持优雅的服务关闭和重启


## 快速开始

### 前置要求

- Go 1.25+
- Docker & Docker Compose（用于运行 Consul）
- Make

### 安装

```bash
# 克隆项目
git clone https://github.com/heytom-labs/heytom-gateway.git
cd heytom-gateway

# 安装依赖
go mod tidy

# 安装 Wire（依赖注入工具）
go install github.com/google/wire/cmd/wire@latest
```

### 配置

编辑 `configs/config.json`：

```json
{
  "server": {
    "http_port": ":8080",
    "grpc_port": ":9090",
    "host": "127.0.0.1"
  },
  "registry": {
    "enabled": true,
    "type": "consul",
    "address": "127.0.0.1:8500",
    "service_name": "heytom-gateway",
    "service_id": "heytom-gateway-1",
    "tags": ["gateway", "api"],
    "health_check_timeout": 5000000000,
    "health_check_ttl": 15000000000
  }
}
```

### 运行

```bash
# 使用 Make
make run

# 或直接使用 Go
go run ./cmd/gateway
```

### 测试

```bash
# 测试 HTTP 服务
curl http://localhost:8080/

# 测试健康检查
curl http://localhost:8080/health
```

## 贡献

欢迎贡献代码！请遵循以下步骤：

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 联系方式

- 项目主页: https://github.com/heytom-labs/heytom-gateway
- 问题反馈: https://github.com/heytom-labs/heytom-gateway/issues

## 致谢

- [gRPC](https://grpc.io/) - 高性能 RPC 框架
- [Consul](https://www.consul.io/) - 服务发现和配置
- [Wire](https://github.com/google/wire) - 依赖注入工具
