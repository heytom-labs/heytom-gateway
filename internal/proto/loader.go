package proto

import (
	"fmt"
	"os"
	"sync"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// DescriptorLoader 用于加载和管理 protobuf 描述符
type DescriptorLoader struct {
	mu      sync.RWMutex
	fileSet *descriptorpb.FileDescriptorSet
}

// NewDescriptorLoader 创建描述符加载器
func NewDescriptorLoader(protosetPath string) (*DescriptorLoader, error) {
	data, err := os.ReadFile(protosetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read protoset file: %w", err)
	}

	fileSet := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(data, fileSet); err != nil {
		return nil, fmt.Errorf("failed to unmarshal protoset file: %w", err)
	}

	return &DescriptorLoader{
		fileSet: fileSet,
	}, nil
}

// LoadProtoset 加载单个 protoset 文件
func (d *DescriptorLoader) LoadProtoset(protosetPath string) error {
	data, err := os.ReadFile(protosetPath)
	if err != nil {
		return fmt.Errorf("failed to read protoset file: %w", err)
	}

	fileSet := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(data, fileSet); err != nil {
		return fmt.Errorf("failed to unmarshal protoset file: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// 合并文件描述符集
	d.fileSet.File = append(d.fileSet.File, fileSet.File...)
	return nil
}

// LoadProtosetData 加载 protoset 数据（从制品库或其他源）
func (d *DescriptorLoader) LoadProtosetData(data []byte) error {
	fileSet := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(data, fileSet); err != nil {
		return fmt.Errorf("failed to unmarshal protoset data: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// 合并文件描述符集
	d.fileSet.File = append(d.fileSet.File, fileSet.File...)
	return nil
}

// ReplaceProtoset 替换整个 protoset（用于热更新）
func (d *DescriptorLoader) ReplaceProtoset(protosetPath string) error {
	data, err := os.ReadFile(protosetPath)
	if err != nil {
		return fmt.Errorf("failed to read protoset file: %w", err)
	}

	fileSet := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(data, fileSet); err != nil {
		return fmt.Errorf("failed to unmarshal protoset file: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// 替换整个文件集
	d.fileSet = fileSet
	return nil
}

// ReplaceProtosetData 替换整个 protoset 数据（用于热更新）
func (d *DescriptorLoader) ReplaceProtosetData(data []byte) error {
	fileSet := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(data, fileSet); err != nil {
		return fmt.Errorf("failed to unmarshal protoset data: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// 替换整个文件集
	d.fileSet = fileSet
	return nil
}

// GetFileDescriptor 获取文件描述符
func (d *DescriptorLoader) GetFileDescriptor(name string) *descriptorpb.FileDescriptorProto {
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, file := range d.fileSet.File {
		if file.GetName() == name {
			return file
		}
	}
	return nil
}

// FindServiceDescriptor 查找服务描述符
// fullName 格式: package.ServiceName 例如 order.OrderService
func (d *DescriptorLoader) FindServiceDescriptor(fullName string) *descriptorpb.ServiceDescriptorProto {
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, file := range d.fileSet.File {
		// 构建完整的服务名 = 包名 + 服务名
		for _, service := range file.Service {
			fullServiceName := file.GetPackage() + "." + service.GetName()
			if fullServiceName == fullName {
				return service
			}
		}
	}
	return nil
}

// FindMethodDescriptor 查找方法描述符
// serviceName 格式: package.ServiceName
// methodName 格式: MethodName
func (d *DescriptorLoader) FindMethodDescriptor(serviceName, methodName string) *descriptorpb.MethodDescriptorProto {
	service := d.FindServiceDescriptor(serviceName)
	if service == nil {
		return nil
	}

	for _, method := range service.Method {
		if method.GetName() == methodName {
			return method
		}
	}
	return nil
}

// FindMessageDescriptor 查找消息描述符
// fullName 格式: package.MessageName 或 package.OuterMessage.InnerMessage
func (d *DescriptorLoader) FindMessageDescriptor(fullName string) *descriptorpb.DescriptorProto {
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, file := range d.fileSet.File {
		msg := d.findMessageInFile(file, fullName, file.GetPackage())
		if msg != nil {
			return msg
		}
	}
	return nil
}

// findMessageInFile 在文件中查找消息描述符
func (d *DescriptorLoader) findMessageInFile(file *descriptorpb.FileDescriptorProto, fullName, pkgPrefix string) *descriptorpb.DescriptorProto {
	for _, msg := range file.MessageType {
		fullMsgName := pkgPrefix + "." + msg.GetName()
		if fullMsgName == fullName {
			return msg
		}
		// 检查嵌套消息
		if nested := d.findNestedMessage(msg, fullName, fullMsgName); nested != nil {
			return nested
		}
	}
	return nil
}

// findNestedMessage 查找嵌套消息
func (d *DescriptorLoader) findNestedMessage(msg *descriptorpb.DescriptorProto, fullName, prefix string) *descriptorpb.DescriptorProto {
	for _, nested := range msg.NestedType {
		fullNestedName := prefix + "." + nested.GetName()
		if fullNestedName == fullName {
			return nested
		}
		if found := d.findNestedMessage(nested, fullName, fullNestedName); found != nil {
			return found
		}
	}
	return nil
}

// GetFileDescriptorSet 获取完整的 FileDescriptorSet
func (d *DescriptorLoader) GetFileDescriptorSet() *descriptorpb.FileDescriptorSet {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.fileSet
}
