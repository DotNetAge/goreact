package evo

import (
	"github.com/DotNetAge/goreact/pkg/core"
)

// CompiledAction 代表技能的“编译态”
// 它是一个已经被蒸馏、固化并带指纹验证的动作集合。
// 在执行层面，它被视为一个复合 Action。
type CompiledAction struct {
	// SkillName 原始技能名称
	SkillName string `json:"skill_name"`
	// InputSchema 该动作所需的参数 Schema（用于验证输入）
	InputSchema core.JSONSchema `json:"input_schema,omitempty"`
	// Steps 编译后的确定性步骤序列
	Steps []ActionStep `json:"steps"`
	// SuccessCriteria 最终成功的判定准则（自然语言描述）
	SuccessCriteria string `json:"success_criteria,omitempty"`
}

// ActionStep 表示编译态中的一个确定原子动作。
// 它是从原始 ReAct Trace 中蒸馏出的“标准动作”。
type ActionStep struct {
	// ID 步骤唯一标识
	ID string `json:"id"`
	// ToolName 要调用的工具名称
	ToolName string `json:"tool_name"`
	// InputTemplate 输入参数模板（支持 Go Template 语法，可引用 variableState）
	InputTemplate string `json:"input_template"`
	// ExpectedObservation 预期观察指纹（用于法官裁定环境是否发生变化）
	ExpectedObservation string `json:"expected_observation"`
	// ValidationRule 验证指纹的正则规则（可选）
	ValidationRule string `json:"validation_rule,omitempty"`
	// Description 步骤的人类可读描述
	Description string `json:"description"`
}
