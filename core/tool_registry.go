package core

type ToolRegistry interface {
	Register(tool FuncTool) error
	Get(name string) (FuncTool, bool)
	All() []FuncTool
	FindAvailable(filter *ToolFilter) []FuncTool
}

type ToolFilter struct {
	Terms        string        // 匹配语义内容
	Security     SecurityLevel // 匹配安全等级
	Keywords     []string      // 匹配关键词
	AllowedNames []string      // 匹配工具名称
}
