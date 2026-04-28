package core

func ToToolInfos(tools []FuncTool) []ToolInfo {
	if len(tools) == 0 {
		return nil
	}
	infos := make([]ToolInfo, len(tools))
	for i, t := range tools {
		infos[i] = *t.Info()
	}
	return infos
}
