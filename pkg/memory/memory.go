package memory

import (
	"context"
)

// WorkingMemory 代表短期会话记忆，用于记录会话经验与临时授权。
// 支持基于时间衰减的洗牌机制。
type WorkingMemory interface {
	// RecallContext 根据当前意图召回短期会话上下文。
	RecallContext(ctx context.Context, sessionID, intent string) (string, error)
	// Update 记录或更新一条短期记忆，deltaWeight 用于调整其权重（如随时间衰减或因惩罚降低）。
	Update(ctx context.Context, sessionID, key string, deltaWeight float64) error
	// Store 保存具体的键值对（例如白名单授权 whitelist:xxx）。
	Store(ctx context.Context, sessionID, key string, value any) error
	// Retrieve 获取具体的键值对。
	Retrieve(ctx context.Context, sessionID, key string) (any, error)
}

// SemanticMemory 代表长期知识库（RAG/GraphRAG），提供只读的知识召回。
type SemanticMemory interface {
	// RecallKnowledge 根据当前意图从外部知识库中检索并召回相关背景知识。
	RecallKnowledge(ctx context.Context, intent string) (string, error)
}

// MuscleMemory 代表肌肉记忆，用于存储大模型在执行特定 Skill 过程中蒸馏出的成功经验。
// 它属于 NativeRAG 范畴，通常通过 SkillName 进行直接检索。
type MuscleMemory[T any] interface {
	// RecallExperience 召回特定技能的成功经验或避坑指南。
	RecallExperience(ctx context.Context, skillName string) (string, error)
	// DistillExperience 提炼并保存特定技能的经验教训。
	DistillExperience(ctx context.Context, skillName, newAction string) error
	
	// LoadCompiledAction 基于“语义化匹配”召回与当前意图最相关的“编译态”执行图（Evo 模式核心）
	LoadCompiledAction(ctx context.Context, intent string) (T, error)
	// SaveCompiledAction 将特定意图或技能的“编译态”执行图固化并进行知识索引（Evo 模式核心）
	SaveCompiledAction(ctx context.Context, intent string, sop T) error
}

// MemoryBank 将三种记忆模态组合在一起，成为 Agent 的专属记忆体。
type MemoryBank interface {
	Working() WorkingMemory
	Semantic() SemanticMemory
	Muscle() MuscleMemory[any] // 默认提供 any，但开发者可以定制
	// Compress 压缩或修剪旧的、不相关的记忆以节省空间。
	Compress(ctx context.Context, sessionID string) error
}
