package model

// DefaultModelManager 默认模型管理器
type DefaultModelManager struct {
	models map[string]Model
}

// NewDefaultModelManager 创建默认模型管理器
func NewDefaultModelManager() *DefaultModelManager {
	return &DefaultModelManager{
		models: make(map[string]Model),
	}
}

// RegisterModel 注册模型
func (m *DefaultModelManager) RegisterModel(model Model) {
	m.models[model.Name()] = model
}

// GetModel 获取模型
func (m *DefaultModelManager) GetModel(name string) Model {
	return m.models[name]
}

// SelectModel 根据任务和需求选择模型
func (m *DefaultModelManager) SelectModel(task string, requirements []Capability) Model {
	// 简单的模型选择逻辑
	// 遍历所有模型，找到满足所有需求的模型
	var bestModel Model
	var bestScore int

	for _, model := range m.models {
		modelCaps := model.GetCapabilities()
		score := 0
		
		// 检查模型是否满足所有需求
		allRequirementsMet := true
		for _, req := range requirements {
			found := false
			for _, cap := range modelCaps {
				if cap.Name == req.Name && cap.Level >= req.Level {
					score += cap.Level
					found = true
					break
				}
			}
			if !found {
				allRequirementsMet = false
				break
			}
		}

		if allRequirementsMet && score > bestScore {
			bestModel = model
			bestScore = score
		}
	}

	// 如果没有找到满足所有需求的模型，返回第一个模型
	if bestModel == nil && len(m.models) > 0 {
		for _, model := range m.models {
			bestModel = model
			break
		}
	}

	return bestModel
}
