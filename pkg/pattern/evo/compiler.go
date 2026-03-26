package evo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/thinker"
)

const CompilationPrompt = `You are a Code Compiler for AI Agents. 
Your task is to convert a sequence of successful ReAct Traces into a structured "CompiledAction".

GUIDELINES:
1. IDENTIFY INVARIANTS: Look at the Action name and input. Identify which parts are dynamic parameters (e.g., a file path, a search query) and replace them with Go template syntax like {{.path}}.
2. EXTRACT FINGERPRINTS: Look at the Observation. Identify the MINIMUM necessary evidence that proves this step succeeded. This is the "ExpectedObservation".
3. NO HALLUCINATION: Only use information present in the Traces.

Traces to Compile:
%s

Output ONLY a JSON object matching the CompiledAction structure.`

// Compiler 负责将执行痕迹编译为肌肉记忆
type Compiler struct {
	thinker thinker.Thinker
}

func NewCompiler(t thinker.Thinker) *Compiler {
	return &Compiler{thinker: t}
}

// Compile 从执行痕迹中生成编译态图形
func (c *Compiler) Compile(ctx context.Context, skillName string, traces []core.Trace) (*CompiledAction, error) {
	// 1. 将 Trace 序列化为文本，供 LLM 分析
	tracesJSON, _ := json.MarshalIndent(traces, "", "  ")
	
	// 2. 构造引导式 Prompt
	input := "/json " + fmt.Sprintf(CompilationPrompt, string(tracesJSON))
	
	// 3. 调用 Thinker (利用我们之前建立的 /json 直通协议)
	pctx := core.NewPipelineContext(ctx, "compilation-"+skillName, input)
	err := c.thinker.Think(pctx)
	if err != nil {
		return nil, fmt.Errorf("compilation thinking failed: %w", err)
	}

	// 4. 解析生成的图形
	var graph CompiledAction
	err = json.Unmarshal([]byte(pctx.FinalResult), &graph)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compiled graph JSON: %w. Raw: %s", err, pctx.FinalResult)
	}

	graph.SkillName = skillName
	return &graph, nil
}
