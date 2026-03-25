package evo

import (
	"context"
	"fmt"

	"github.com/ray/goreact/pkg/actor"
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/memory"
	"github.com/ray/goreact/pkg/observer"
	"github.com/ray/goreact/pkg/pattern/mastersub"
	"github.com/ray/goreact/pkg/thinker"
)

// EvolutionPipeline 提供了一站式的自适应编排能力
type EvolutionPipeline struct {
	master     *mastersub.Master
	sub        mastersub.SubReactor
	compiler   *Compiler
	memoryBank memory.MemoryBank
	actor      actor.Actor
	observer   observer.Observer
	thinker    thinker.Thinker
	logger     core.Logger
}

func NewEvolutionPipeline(
	m *mastersub.Master,
	s mastersub.SubReactor,
	c *Compiler,
	mb memory.MemoryBank,
	a actor.Actor,
	o observer.Observer,
	t thinker.Thinker,
	l core.Logger,
) *EvolutionPipeline {
	return &EvolutionPipeline{
		master:     m,
		sub:        s,
		compiler:   c,
		memoryBank: mb,
		actor:      a,
		observer:   o,
		thinker:    t,
		logger:     l,
	}
}

// Execute 自动决策并执行任务（编译态优先，ReAct 降级备份）
func (p *EvolutionPipeline) Execute(ctx context.Context, intent string, inputParams map[string]any) (string, error) {
	// 1. 基于“语义化匹配”从肌肉记忆中召回最相关的编译态执行图 (CompiledAction)
	// 在 AI 时代，即使传入的是模糊的意图 (intent)，也能通过 RAG 召回精确的 Skill。
	p.logger.Info("Searching muscle memory for relevant compiled SOP (Semantic Matching)...", "intent", intent)

	rawAction, err := p.memoryBank.Muscle().LoadCompiledAction(ctx, intent)

	if err == nil && rawAction != nil {
		// 检查类型安全性
		if action, ok := rawAction.(*CompiledAction); ok {
			// 2. 存在编译态，启动快径执行 (AdaptiveRunner)
			runner := NewAdaptiveRunner(action, p.thinker, p.actor, p.observer, p.logger)
			
			p.logger.Info("Executing in Compiled State (Fast-Path)", "intent", intent)
			result, err := runner.Run(ctx, inputParams)
			if err == nil {
				return result, nil // 快径执行成功，直接返回
			}
			
			p.logger.Warn("Fast-Path execution failed. Falling back to Master-Sub ReAct.", "error", err)
		}
	}

	// 3. 降级执行：启动 Master-Sub 模式进行全量推理
	p.logger.Info("Executing in Source State (Full ReAct)", "intent", intent)
	orchestrator := mastersub.NewOrchestrator(p.master, p.sub, p.logger)
	
	// 这里将参数渲染到任务描述中
	goal := fmt.Sprintf("Execute Intent: %s with inputs: %v", intent, inputParams)
	taskResults, err := orchestrator.Run(ctx, goal)
	if err != nil {
		return "", fmt.Errorf("full ReAct execution failed: %w", err)
	}

	// 4. 编译进化：既然 ReAct 成功了，我们就把这次经验编译并固化
	p.logger.Info("Consolidating experience. Compiling Traces into Muscle Memory...", "intent", intent)
	
	// 收集所有子任务的 Traces
	allTraces := make([]core.Trace, 0)
	for _, res := range taskResults {
		allTraces = append(allTraces, res.Traces...)
	}

	// 调用编译器生成 SOP
	newAction, compileErr := p.compiler.Compile(ctx, intent, allTraces)
	if compileErr == nil {
		// 持久化到肌肉记忆 (泛型支持)
		p.memoryBank.Muscle().SaveCompiledAction(ctx, intent, newAction)
		p.logger.Info("SOP successfully compiled and persisted to Muscle Memory!", "intent", intent)
	} else {
		p.logger.Warn("Compilation failed, but execution was successful.", "error", compileErr)
	}

	// 返回最后一个任务的结果作为最终答案
	if len(taskResults) > 0 {
		return taskResults[len(taskResults)-1].Answer, nil
	}

	return "Task completed via ReAct", nil
}
