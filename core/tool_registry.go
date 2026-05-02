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
	Terms        string        // semantic matching terms
	Security     SecurityLevel // matching security level
	Keywords     []string      // matching keywords
	AllowedNames []string      // matching tool names
}
