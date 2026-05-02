package core

// ToolRegistry manages the registration and discovery of FuncTool instances.
// This is a DYNAMIC registry: tools can be registered/unregistered at runtime
// based on context (e.g., permission level, active skills). It is distinct from
// MCPToolRegistry which handles static MCP tool definitions.
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
