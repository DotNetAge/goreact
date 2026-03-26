package skill

import (
	"context"

	"github.com/DotNetAge/goreact/pkg/memory"
)

// Compiler 知识结晶器接口
// 负责从 Agent 的肌肉记忆和短期会话记忆中提取高价值经验，并反写为持久化的技能（SKILL.md）。
type Compiler interface {
	// RefineCompiledSkill 现有技能打磨。
	// 扫描积累的同一任务下的高权重“肌肉记忆”，利用大模型对历史经验进行二次总结蒸馏，覆写现有的 SKILL.md。
	RefineCompiledSkill(ctx context.Context, skillName string, muscleMemory memory.MuscleMemory[any]) error

	// DiscoverNewSkill 新技能涌现。
	// 分析长期沉淀的“短期会话记忆”，寻找缺乏专属 Skill 的长尾工作流，自动撰写全新的 SKILL.md 并注册。
	DiscoverNewSkill(ctx context.Context, workingMemory memory.WorkingMemory) (*Skill, error)
}
