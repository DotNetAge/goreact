package reactor

import (
	"strings"

	"github.com/DotNetAge/goreact/core"
)

// orchestrationToolNames identifies tools that spawn or manage subagents/teams.
var orchestrationToolNames = map[string]bool{
	// Task tools
	"task_create": true,
	"task_result": true,
	// SubAgent tools
	"subagent":        true,
	"subagent_result": true,
	// Team tools
	"team_create": true,
	"team_delete": true,
	"team_status": true,
	"wait_team":   true,
	// Communication
	"send_message":     true,
	"receive_messages": true,
}

// buildExperienceCandidate evaluates whether a completed T-A-O execution produced
// reusable experience, and if so returns a structured *core.ExperienceData.
// Returns nil when the execution is not worth persisting (no answer, errors, trivial).
//
// The Reactor does NOT store experience — it only produces candidates.
// The caller (Agent or client code) decides whether and how to persist,
// allowing custom evaluation logic (quality scoring, dedup, format conversion).
func (r *Reactor) buildExperienceCandidate(ctx *ReactContext, result *RunResult) *core.ExperienceData {
	if result.Answer == "" {
		return nil
	}
	if strings.Contains(result.TerminationReason, "error") {
		return nil
	}

	if result.TotalIterations <= 1 {
		hasToolCall := false
		for _, step := range ctx.History {
			if step.Action.Type == ActionTypeToolCall {
				hasToolCall = true
				break
			}
		}
		if !hasToolCall {
			return nil
		}
	}

	exp := buildExperienceData(ctx.Input, ctx.History, result)

	ctx.EmitEvent(core.ExperienceSaved, core.ExperienceSavedData{
		Problem:    ctx.Input,
		Iterations: result.TotalIterations,
		ToolsUsed:  exp.Tools,
	})

	return exp
}

// buildExperienceData extracts the reusable analysis from a completed T-A-O execution.
func buildExperienceData(input string, history []Step, result *RunResult) *core.ExperienceData {
	exp := &core.ExperienceData{
		Problem:   input,
		Answer:    truncate(result.Answer, 500),
		TokenCost: result.TokensUsed,
	}

	// Collect unique tools, subagents, and build compact step summaries
	exp.Tools = collectUniqueToolNames(history)
	var reasoningParts []string

	for _, step := range history {
		// Collect reasoning from Think phases
		if step.Thought.Reasoning != "" {
			reasoningParts = append(reasoningParts, step.Thought.Reasoning)
		}

		// Track orchestration tools (task_create, subagent)
		if step.Action.Type == ActionTypeToolCall && step.Action.Target != "" && orchestrationToolNames[step.Action.Target] {
			sa := core.ExperienceSubAgent{
				Tool:    step.Action.Target,
				Success: step.Observation.Error == "",
			}
			if step.Action.Params != nil {
				if name, ok := step.Action.Params["name"].(string); ok {
					sa.Name = name
				} else if desc, ok := step.Action.Params["description"].(string); ok {
					sa.Name = desc
				}
				if prompt, ok := step.Action.Params["prompt"].(string); ok {
					sa.Prompt = truncate(prompt, 200)
				}
			}
			if step.Action.Target == "subagent_result" && step.Action.Params != nil {
				if taskID, ok := step.Action.Params["task_id"].(string); ok {
					sa.Name = taskID
				}
			}
			exp.SubAgents = append(exp.SubAgents, sa)
		}

		// Build compact step summary
		es := core.ExperienceStep{
			HasError: step.Observation.Error != "",
		}
		if step.Thought.Reasoning != "" {
			es.Thought = truncate(step.Thought.Reasoning, 200)
		}
		if step.Action.Type == ActionTypeToolCall {
			es.Action = step.Action.Target
		}
		if step.Observation.Result != "" {
			es.Result = truncate(step.Observation.Result, 200)
		}
		exp.Steps = append(exp.Steps, es)
	}

	// Combine all reasoning into the analysis field (most valuable for reuse)
	exp.Analysis = truncate(strings.Join(reasoningParts, "\n---\n"), 2000)

	return exp
}
