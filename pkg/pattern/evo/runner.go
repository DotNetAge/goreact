package evo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/DotNetAge/goreact/pkg/actor"
	"github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/observer"
	"github.com/DotNetAge/goreact/pkg/prompt/builder"
	"github.com/DotNetAge/goreact/pkg/thinker"
)

// AdaptiveRunner 负责执行编译态图形，并在偏差时触发升级
type AdaptiveRunner struct {
	graph    *CompiledAction
	thinker  thinker.Thinker
	actor    actor.Actor
	observer observer.Observer
	logger   core.Logger
}

func NewAdaptiveRunner(g *CompiledAction, t thinker.Thinker, a actor.Actor, o observer.Observer, l core.Logger) *AdaptiveRunner {
	return &AdaptiveRunner{
		graph:    g,
		thinker:  t,
		actor:    a,
		observer: o,
		logger:   l,
	}
}

// Run 执行自适应快径
func (r *AdaptiveRunner) Run(ctx context.Context, inputParams map[string]any) (string, error) {
	r.logger.Info("Starting Adaptive Fast-Path", "skill", r.graph.SkillName)
	
	variableState := make(map[string]any)
	for k, v := range inputParams {
		variableState[k] = v
	}

	// 构造一个持久的 PipelineContext
	pctx := core.NewPipelineContext(ctx, "fast-path-"+r.graph.SkillName, "")

	// 循环执行编译态步骤
	for i, step := range r.graph.Steps {
		r.logger.Info("Executing Compiled Step", "step", i, "tool", step.ToolName)

		// 1. 渲染输入参数
		actionParams, err := r.renderStepInput(step.InputTemplate, variableState)
		if err != nil {
			return "", fmt.Errorf("failed to render step input: %w", err)
		}

		// 2. 模拟 Thinker 的产出：手动向 pctx 注入一个 Trace (Action)
		pctx.AppendTrace(&core.Trace{
			Step:    i + 1,
			Thought: "Fast-path execution of step " + step.ID,
			Action: &core.Action{
				Name:  step.ToolName,
				Input: actionParams,
			},
		})

		// 3. Actor 执行
		err = r.actor.Act(pctx)
		if err != nil {
			return r.escalate(pctx, i, fmt.Sprintf("Execution Error: %v", err), step, variableState)
		}

		// 4. Observer 处理
		err = r.observer.Observe(pctx)
		if err != nil {
			return r.escalate(pctx, i, fmt.Sprintf("Observation Error: %v", err), step, variableState)
		}

		// 5. 法官裁定
		lastTrace := pctx.LastTrace()
		if lastTrace == nil || lastTrace.Observation == nil {
			return r.escalate(pctx, i, "No observation found", step, variableState)
		}

		if !r.validate(lastTrace.Observation.Data, step) {
			r.logger.Warn("Step expectation mismatch! Escalating...", 
				"step", i, 
				"actual", lastTrace.Observation.Data, 
				"expected", step.ExpectedObservation)
			return r.escalate(pctx, i, lastTrace.Observation.Data, step, variableState)
		}

		// 匹配成功，更新状态变量
		variableState[fmt.Sprintf("step_%d_output", i)] = lastTrace.Observation.Data
	}

	return "Compiled execution successful", nil
}

// validate 判定 Observation 是否满足步骤期望
func (r *AdaptiveRunner) validate(actual string, step ActionStep) bool {
	if step.ValidationRule != "" {
		matched, err := regexp.MatchString(step.ValidationRule, actual)
		if err == nil && matched {
			return true
		}
	}
	return strings.Contains(actual, step.ExpectedObservation)
}

// escalate 触发升级流程
func (r *AdaptiveRunner) escalate(pctx *core.PipelineContext, stepIndex int, actual string, step ActionStep, state map[string]any) (string, error) {
	pb := builder.New().
		WithSystemTemplate(`You are a Detective Agent (Thinker). 
A linear execution path has FAILED at step {{.step_index}}. 
Your goal is to investigate why the Actual result does not match the Expected Fingerprint.

ACTUAL OBSERVATION:
{{.actual}}

EXPECTED FINGERPRINT:
{{.expected}}

STEP DESCRIPTION:
{{.description}}

CURRENT STATE:
{{.state}}

Decide if this is an 'Environment Change' (the fingerprint needs update) or an 'Execution Failure'. 
Then, provide the next ReAct Action to fix the situation.`).
		WithVariable("step_index", stepIndex).
		WithVariable("actual", actual).
		WithVariable("expected", step.ExpectedObservation).
		WithVariable("description", step.Description).
		WithVariable("state", state)

	builtPrompt := pb.Build()

	pctx.Input = builtPrompt.System
	
	err := r.thinker.Think(pctx)
	if err != nil {
		return "", fmt.Errorf("detective thinking failed: %w", err)
	}

	return pctx.FinalResult, nil
}

// renderStepInput 将 Go Template 渲染为具体的 Tool 参数
func (r *AdaptiveRunner) renderStepInput(tmplStr string, state map[string]any) (map[string]any, error) {
	tmpl, err := template.New("input").Parse(tmplStr)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, state); err != nil {
		return nil, err
	}

	var result map[string]any
	inputBytes := buf.Bytes()
	if json.Valid(inputBytes) {
		if err := json.Unmarshal(inputBytes, &result); err == nil {
			return result, nil
		}
	}

	return map[string]any{"query": string(inputBytes)}, nil
}
