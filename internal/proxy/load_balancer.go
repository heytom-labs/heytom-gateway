package proxy

import (
	"fmt"
	"math/rand"
	"sync/atomic"

	"github.com/heytom-labs/heytom-gateway/internal/registry"
)

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	Select(instances []*registry.ServiceInstance) *registry.ServiceInstance
}

// RoundRobinLoadBalancer 轮询负载均衡器
type RoundRobinLoadBalancer struct {
	counter uint64
}

// NewRoundRobinLoadBalancer 创建轮询负载均衡器
func NewRoundRobinLoadBalancer() *RoundRobinLoadBalancer {
	return &RoundRobinLoadBalancer{}
}

// Select 选择实例
func (lb *RoundRobinLoadBalancer) Select(instances []*registry.ServiceInstance) *registry.ServiceInstance {
	if len(instances) == 0 {
		return nil
	}

	index := atomic.AddUint64(&lb.counter, 1)
	return instances[int(index)%len(instances)]
}

// RandomLoadBalancer 随机负载均衡器
type RandomLoadBalancer struct{}

// NewRandomLoadBalancer 创建随机负载均衡器
func NewRandomLoadBalancer() *RandomLoadBalancer {
	return &RandomLoadBalancer{}
}

// Select 选择实例
func (lb *RandomLoadBalancer) Select(instances []*registry.ServiceInstance) *registry.ServiceInstance {
	if len(instances) == 0 {
		return nil
	}

	return instances[rand.Intn(len(instances))]
}

// WeightedLoadBalancer 加权负载均衡器
type WeightedLoadBalancer struct {
	counter uint64
}

// NewWeightedLoadBalancer 创建加权负载均衡器
func NewWeightedLoadBalancer() *WeightedLoadBalancer {
	return &WeightedLoadBalancer{}
}

// Select 选择实例（基于权重）
func (lb *WeightedLoadBalancer) Select(instances []*registry.ServiceInstance) *registry.ServiceInstance {
	if len(instances) == 0 {
		return nil
	}

	// 计算总权重
	totalWeight := 0
	for _, instance := range instances {
		weight := getWeight(instance)
		totalWeight += weight
	}

	if totalWeight == 0 {
		// 如果没有权重，使用轮询
		index := atomic.AddUint64(&lb.counter, 1)
		return instances[int(index)%len(instances)]
	}

	// 加权选择
	index := atomic.AddUint64(&lb.counter, 1)
	offset := int(index) % totalWeight
	currentWeight := 0

	for _, instance := range instances {
		weight := getWeight(instance)
		currentWeight += weight
		if offset < currentWeight {
			return instance
		}
	}

	return instances[0]
}

// getWeight 从实例元数据中获取权重
func getWeight(instance *registry.ServiceInstance) int {
	if instance.Metadata == nil {
		return 1
	}

	if weightStr, ok := instance.Metadata["weight"]; ok {
		var weight int
		if _, err := fmt.Sscanf(weightStr, "%d", &weight); err == nil && weight > 0 {
			return weight
		}
	}

	return 1
}
