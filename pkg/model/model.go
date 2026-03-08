package model

// Model 模型接口
type Model interface {
	// Name 返回模型名称
	Name() string
	// Type 返回模型类型
	Type() string
	// Execute 执行模型推理
	Execute(prompt string, options map[string]interface{}) (string, error)
	// GetCapabilities 返回模型能力
	GetCapabilities() []Capability
}

// Capability 模型能力
type Capability struct {
	Name        string
	Description string
	Level       int // 能力等级，1-5
}

// ModelManager 模型管理器接口
type ModelManager interface {
	// RegisterModel 注册模型
	RegisterModel(model Model)
	// GetModel 获取模型
	GetModel(name string) Model
	// SelectModel 根据任务和需求选择模型
	SelectModel(task string, requirements []Capability) Model
}
