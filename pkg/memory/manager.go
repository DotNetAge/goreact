package memory

import (
	"context"
	"sync"
)

// DefaultMemoryBank 是 MemoryBank 的一个简单基础实现，
// 它通过内部 KV 存储来模拟三种不同模式的记忆。
type DefaultMemoryBank struct {
	working  *defaultWorkingMemory
	semantic *defaultSemanticMemory
	muscle   *defaultMuscleMemory[any]
}

func NewDefaultMemoryBank() *DefaultMemoryBank {
	return &DefaultMemoryBank{
		working:  &defaultWorkingMemory{data: make(map[string]any)},
		semantic: &defaultSemanticMemory{},
		muscle:   &defaultMuscleMemory[any]{data: make(map[string]any)},
	}
}

func (b *DefaultMemoryBank) Working() WorkingMemory { return b.working }
func (b *DefaultMemoryBank) Semantic() SemanticMemory { return b.semantic }
func (b *DefaultMemoryBank) Muscle() MuscleMemory[any] { return b.muscle }

func (b *DefaultMemoryBank) Compress(ctx context.Context, sessionID string) error {
	return nil
}

// --- 内部实现 ---

type defaultWorkingMemory struct {
	data  map[string]any
	mutex sync.RWMutex
}

func (w *defaultWorkingMemory) RecallContext(ctx context.Context, sessionID, intent string) (string, error) {
	return "", nil
}

func (w *defaultWorkingMemory) Update(ctx context.Context, sessionID, key string, deltaWeight float64) error {
	return nil
}

func (w *defaultWorkingMemory) Store(ctx context.Context, sessionID, key string, value any) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.data[sessionID+":"+key] = value
	return nil
}

func (w *defaultWorkingMemory) Retrieve(ctx context.Context, sessionID, key string) (any, error) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.data[sessionID+":"+key], nil
}

type defaultSemanticMemory struct{}

func (s *defaultSemanticMemory) RecallKnowledge(ctx context.Context, intent string) (string, error) {
	return "No semantic knowledge found.", nil
}

func (s *defaultSemanticMemory) QueryGraph(ctx context.Context, query string, depth int) (any, error) {
	// 在默认实现中，我们模拟返回一个空的结果。
	// 实际应用中，开发者会注入支持分布式图检索的 Client（如 Nebula, Neo4j）。
	return nil, nil
}

type defaultMuscleMemory[T any] struct {
	data  map[string]T
	mutex sync.RWMutex
}

func (m *defaultMuscleMemory[T]) RecallExperience(ctx context.Context, skillName string) (string, error) {
	return "", nil
}

func (m *defaultMuscleMemory[T]) DistillExperience(ctx context.Context, skillName, newAction string) error {
	return nil
}

func (m *defaultMuscleMemory[T]) LoadCompiledAction(ctx context.Context, intent string) (T, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	// 注意：这里是 NativeRAG 的极简实现（精确查表）。
	// 在生产环境中接入 GoRAG 后，这里应该是向量语义召回逻辑。
	return m.data[intent], nil
}

func (m *defaultMuscleMemory[T]) SaveCompiledAction(ctx context.Context, intent string, sop T) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	// 在高级实现中，这应该触发一个 Index() 操作。
	m.data[intent] = sop
	return nil
}
