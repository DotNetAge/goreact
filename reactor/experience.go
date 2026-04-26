package reactor

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// orchestrationToolNames identifies tools that spawn or manage subagents/teams.
var orchestrationToolNames = map[string]bool{
	// Task tools
	"task_create":     true,
	"task_result":     true,
	// SubAgent tools
	"subagent":        true,
	"subagent_result": true,
	// Team tools
	"team_create":  true,
	"team_delete":  true,
	"team_status":  true,
	"wait_team":    true,
	// Communication
	"send_message":      true,
	"receive_messages":  true,
}

// saveExperience saves the execution trace as an experience memory record
// when the task completed successfully (has answer, no critical errors).
// This is called at the end of Run, after the execution summary is emitted.
//
// The experience is structured as:
//   - Title (problem): the user's original input / task description
//   - Tags: extracted keywords from the problem for semantic matching
//   - Content (solution): JSON with analysis, tools used, subagents, steps, and answer
//   - Meta: typed *core.ExperienceData for direct access by Memory implementations
//
// Future runs with similar problems will Retrieve this experience during
// the Think phase, allowing the LLM to skip redundant analysis.
func (r *Reactor) saveExperience(ctx *ReactContext, result *RunResult) {
	// Skip if no memory configured
	if r.memory == nil {
		return
	}

	// Only save on successful completion (has answer, not from error/cancellation)
	if result.Answer == "" {
		return
	}
	if strings.Contains(result.TerminationReason, "error") {
		return
	}

	// Skip trivially short tasks (single iteration with no tool calls = no reusable experience)
	if result.TotalIterations <= 1 {
		hasToolCall := false
		for _, step := range ctx.History {
			if step.Action.Type == ActionTypeToolCall {
				hasToolCall = true
				break
			}
		}
		if !hasToolCall {
			return
		}
	}

	// Build experience data
	exp := buildExperienceData(ctx.Input, ctx.History, result)

	// Serialize to JSON for Content field
	contentBytes, err := json.Marshal(exp)
	if err != nil {
		return // non-fatal: don't break the run
	}

	// Extract tags from intent and input for semantic matching
	tags := extractExperienceTags(ctx.Input, ctx.Intent)

	record := core.MemoryRecord{
		Type:    core.MemoryTypeExperience,
		Title:   ctx.Input,            // Problem description (semantic index)
		Content: string(contentBytes), // Solution (analysis + steps) as JSON
		Meta:    exp,                  // Typed metadata for Memory implementations
		Scope:   core.MemoryScopeTeam, // Experiences are shared across the team
		Tags:    tags,
	}

	// Store asynchronously — don't block the Run return
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// non-fatal: experience storage panic should not crash the reactor
			}
		}()
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = r.memory.Store(bgCtx, record)
	}()

	// Emit event so external listeners can observe experience saves
	ctx.EmitEvent(core.ExperienceSaved, core.ExperienceSavedData{
		Problem:    ctx.Input,
		Iterations: result.TotalIterations,
		ToolsUsed:  exp.Tools,
	})
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

// extractExperienceTags generates tags from the user input and intent
// to improve semantic matching when recalling this experience later.
func extractExperienceTags(input string, intent *Intent) []string {
	var tags []string
	seen := make(map[string]bool)

	addTag := func(tag string) {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if len(tag) >= 2 && !seen[tag] {
			tags = append(tags, tag)
			seen[tag] = true
		}
	}

	// From intent
	if intent != nil {
		if intent.Topic != "" {
			addTag(intent.Topic)
		}
		if intent.Type != "" {
			addTag(string(intent.Type))
		}
		for k := range intent.Entities {
			addTag(k)
		}
	}

	// From input: extract meaningful words (simple heuristic)
	words := strings.Fields(input)
	for _, w := range words {
		w = strings.ToLower(strings.TrimSpace(w))
		if len(w) >= 3 {
			addTag(w)
		}
	}

	// Limit tags
	if len(tags) > 10 {
		tags = tags[:10]
	}

	return tags
}
