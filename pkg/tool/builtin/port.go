package builtin

import (
	"fmt"
	"net"
)

// Port 端口查询工具
type Port struct{}

// NewPort 创建端口查询工具
func NewPort() *Port {
	return &Port{}
}

// Name 返回工具名称
func (p *Port) Name() string {
	return "port"
}

// Description 返回工具描述
func (p *Port) Description() string {
	return "端口查询工具，检查端口是否被占用"
}

// Execute 执行端口查询操作
func (p *Port) Execute(params map[string]interface{}) (interface{}, error) {
	port, ok := params["port"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'port' parameter")
	}

	address, ok := params["address"].(string)
	if !ok {
		address = "localhost"
	}

	// 检查端口是否被占用
	addr := fmt.Sprintf("%s:%d", address, int(port))
	listener, err := net.Listen("tcp", addr)

	result := map[string]interface{}{
		"port":     int(port),
		"address":  address,
		"inUse":    err != nil,
	}

	if err != nil {
		result["error"] = err.Error()
	} else {
		listener.Close()
	}

	return result, nil
}
