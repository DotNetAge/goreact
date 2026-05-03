package reactor

import (
	"encoding/json"

	gochatcore "github.com/DotNetAge/gochat/core"

	"github.com/DotNetAge/goreact/core"
)

// DefaultBehavioralRules returns the built-in behavioral rules.
func DefaultBehavioralRules() string {
	return `1. Language Consistency: Always respond in the same language as the user's input.
2. Concise & Precise: Answer directly to the point, avoid redundancy without sacrificing completeness.
3. Tool-first: When a tool can significantly improve answer quality, proactively use it instead of relying solely on memory.
4. Honest & Transparent: Explicitly state uncertainty, never fabricate facts; proactively ask when more information is needed.
5. Safety Boundaries: Do not execute destructive operations that risk data loss or security breaches; high-risk operations require user consent.
6. Context Awareness: Maintain understanding of prior conversation context, leverage context rather than asking users to repeat information.
7. Memory-driven: Prefer known facts from memory; when memory conflicts with prior knowledge, defer to memory.`
}

// ToolInfosToLLMTools converts ToolInfo slice into gochat Tool slice
// with full JSON Schema parameters for native function calling.
func ToolInfosToLLMTools(infos []core.ToolInfo) []gochatcore.Tool {
	if len(infos) == 0 {
		return nil
	}
	tools := make([]gochatcore.Tool, 0, len(infos))
	for _, info := range infos {
		params := buildJSONSchemaParams(info.Parameters)
		tools = append(tools, gochatcore.Tool{
			Name:        info.Name,
			Description: toolDescription(info),
			Parameters:  params,
		})
	}
	return tools
}

// toolDescription returns the best description: Prompt if non-empty, else Description.
func toolDescription(info core.ToolInfo) string {
	if info.Prompt != "" {
		return info.Prompt
	}
	return info.Description
}

// buildJSONSchemaParams converts core.Parameter slice into JSON Schema RawMessage.
func buildJSONSchemaParams(params []core.Parameter) json.RawMessage {
	if len(params) == 0 {
		return json.RawMessage(`{"type":"object","properties":{}}`)
	}
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
		"required":   []string{},
	}
	props := schema["properties"].(map[string]any)
	required := schema["required"].([]string)
	for _, p := range params {
		prop := map[string]any{
			"type":        paramTypeToSchema(p.Type),
			"description": p.Description,
		}
		if len(p.Enum) > 0 {
			prop["enum"] = p.Enum
		}
		if p.Default != nil {
			prop["default"] = p.Default
		}
		props[p.Name] = prop
		if p.Required {
			required = append(required, p.Name)
		}
	}
	schema["required"] = required
	b, _ := json.Marshal(schema)
	return b
}

// paramTypeToSchema maps goreact parameter types to JSON Schema types.
func paramTypeToSchema(t string) string {
	switch t {
	case "integer", "int", "int64", "int32":
		return "integer"
	case "number", "float64", "float32":
		return "number"
	case "boolean", "bool":
		return "boolean"
	case "array", "[]string", "[]int":
		return "array"
	case "object", "map":
		return "object"
	default:
		return "string"
	}
}

// BuildAgentCoordinationGuidance returns the system prompt section for agent orchestration tools.
func BuildAgentCoordinationGuidance() string {
	return `## Agent Coordination

Agent coordination has two purposes: (a) handing off tasks that fall outside your role to a specialist, and (b) parallelizing large workloads by dispatching independent sub-tasks to multiple agents simultaneously.

Do NOT use these tools for tasks you can handle directly. Your first responsibility is to complete the work yourself.

### When to delegate to another agent
- The user asks for something that is not in your area of expertise (e.g. you are a code reviewer and they ask for legal advice).
- The task requires a specialized capability you do not have access to.
- The user explicitly requests that another agent handle the task.

In those cases, use FindAgent to find a matching agent, then Delegate.

### When to parallelize by spawning multiple agents
- The current task involves many independent sub-tasks that could run in parallel (e.g. reviewing 10 files, researching 5 topics, testing 3 configurations).
- You estimate that the total task would take significantly longer if done sequentially — dispatching sub-tasks to agents with the same capabilities as yourself can reduce wall-clock time.
- Each sub-task is self-contained and does not depend on results from other sub-tasks.

In those cases, call Delegate multiple times in the same Act phase with different sub-tasks — they will run in parallel. Use CollectResults to gather all outcomes.

### When to create a new agent
- A specialized task type repeats frequently, and no existing agent covers it.
- The user asks you to define a new expert role with a custom system prompt.

When creating an agent, call ModelList to see available models and SkillList to see available skills. Select the model and skills that match the new agent's role. Skills go in the skills array parameter; the model name goes in the model parameter. If no skill fits, describe the capability in the agent's introduction instead.

### When to rank an agent
- After a delegated task completes and you have evaluated the result.
- Scoring helps the system learn which agents perform well for which kinds of tasks.`
}
