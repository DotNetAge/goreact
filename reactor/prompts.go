package reactor

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	gochatcore "github.com/DotNetAge/gochat/core"

	"github.com/DotNetAge/goreact/core"
)

// promptTemplates is the embedded filesystem containing prompt template files.
// Templates are stored as .tmpl files under the prompts/ directory, making them
// easy to review and edit independently of Go source code.
//
//go:embed prompts/*.tmpl
var promptTemplates embed.FS

// promptFuncMap defines custom template functions used across all prompts.
var promptFuncMap = template.FuncMap{
	"jsonMarshal": func(v any) string {
		b, _ := json.Marshal(v)
		return string(b)
	},
}

// templateNames maps the embed path to the parsed template name used for Lookup.
// Go's template.ParseFS strips the directory prefix, so "prompts/summary_prompt.tmpl"
// becomes associated name "summary_prompt.tmpl".
const (
	tmplIntent = "intent_prompt.tmpl"
	tmplThink  = "think_prompt.tmpl"
	// tmplSkillSelect = "skill_select_prompt.tmpl"
	tmplSystem  = "default_system_prompt.tmpl"
	tmplSummary = "summary_prompt.tmpl"
)

// intentPromptTemplate is parsed once at init from the embedded .tmpl file.
var intentPromptTemplate = template.Must(
	template.New("intent_prompt").Funcs(promptFuncMap).ParseFS(promptTemplates, "prompts/intent_prompt.tmpl"),
)

// thinkPromptTemplate is parsed once at init from the embedded .tmpl file.
var thinkPromptTemplate = template.Must(
	template.New("think_prompt").Funcs(promptFuncMap).ParseFS(promptTemplates, "prompts/think_prompt.tmpl"),
)

// skillSelectPromptTemplate is parsed once at init from the embedded .tmpl file (Phase 1).
var skillSelectPromptTemplate = template.Must(
	template.New("skill_select").Funcs(promptFuncMap).ParseFS(promptTemplates, "prompts/skill_select_prompt.tmpl"),
)

// defaultSystemPromptTemplate is parsed once at init from the embedded .tmpl file.
var defaultSystemPromptTemplate = template.Must(
	template.New("default_system_prompt").Funcs(promptFuncMap).ParseFS(promptTemplates, "prompts/default_system_prompt.tmpl"),
)

// summaryPromptTemplate is parsed once at init from the embedded .tmpl file.
var summaryPromptTemplate = template.Must(
	template.New("summary_prompt").Funcs(promptFuncMap).ParseFS(promptTemplates, "prompts/summary_prompt.tmpl"),
)

// intentPromptData holds template variables for the intent classification prompt.
type intentPromptData struct {
	IntentTypes string
	Input       string
	Context     string
}

// thinkPromptData holds template variables for the Think phase prompt (Phase 2).
// When HasActiveSkill is true, the template renders skill-guided instructions.
type thinkPromptData struct {
	IntentSection string
	MemorySection string
	Input         string

	HasActiveSkill          bool
	ActiveSkillName         string
	ActiveSkillDesc         string
	ActiveSkillInstructions string
	FilteredToolList        string // comma-separated tool names available to the active skill
	ResourceBasePath        string // P3: base path for scripts/, references/, assets/
}

// skillSelectPromptData holds template variables for the Skill Selection prompt (Phase 1).
// Capabilities are injected via SystemPrompt (skillsSection), not rendered in this template.
type skillSelectPromptData struct {
	IntentSection string
	Input         string
}

// systemPromptData holds template variables for the default agent system prompt.
type systemPromptData struct {
	Name        string
	Domain      string
	Description string
}

// summaryPromptData holds template variables for the task summary prompt.
type summaryPromptData struct {
	Input             string
	Answer            string
	Iterations        int
	ToolsUsed         string
	Duration          string
	TerminationReason string
}

// renderIntentPrompt renders the intent prompt using the embedded Go template.
func renderIntentPrompt(data intentPromptData) (string, error) {
	t := intentPromptTemplate.Lookup(tmplIntent)
	if t == nil {
		return "", template.ExecError{Name: tmplIntent, Err: nil}
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderThinkPrompt renders the Think prompt using the embedded Go template.
func renderThinkPrompt(data thinkPromptData) (string, error) {
	t := thinkPromptTemplate.Lookup(tmplThink)
	if t == nil {
		return "", template.ExecError{Name: tmplThink, Err: nil}
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderSkillSelectPrompt renders the Skill Selection prompt (Phase 1) using the embedded Go template.
// func renderSkillSelectPrompt(data skillSelectPromptData) (string, error) {
// 	t := skillSelectPromptTemplate.Lookup(tmplSkillSelect)
// 	if t == nil {
// 		return "", template.ExecError{Name: tmplSkillSelect, Err: nil}
// 	}
// 	var buf bytes.Buffer
// 	if err := t.Execute(&buf, data); err != nil {
// 		return "", err
// 	}
// 	return buf.String(), nil
// }

// RenderDefaultSystemPrompt renders the default agent system prompt using the embedded template.
// It accepts the agent's name, domain, and description as template variables.
func RenderDefaultSystemPrompt(name, domain, description string) (string, error) {
	t := defaultSystemPromptTemplate.Lookup(tmplSystem)
	if t == nil {
		return "", template.ExecError{Name: tmplSystem, Err: nil}
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, systemPromptData{
		Name:        name,
		Domain:      domain,
		Description: description,
	}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderSummaryPrompt renders the task summary prompt using the embedded template.
func renderSummaryPrompt(data summaryPromptData) (string, error) {
	t := summaryPromptTemplate.Lookup(tmplSummary)
	if t == nil {
		return "", template.ExecError{Name: tmplSummary, Err: nil}
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// BuildSummaryToolsUsed extracts unique tool names from step history into a comma-separated string.
func BuildSummaryToolsUsed(steps []Step) string {
	seen := make(map[string]bool)
	var tools []string
	for _, step := range steps {
		if step.Action.Type == ActionTypeToolCall && step.Action.Target != "" {
			if !seen[step.Action.Target] {
				seen[step.Action.Target] = true
				tools = append(tools, step.Action.Target)
			}
		}
	}
	if len(tools) == 0 {
		return "none"
	}
	return strings.Join(tools, ", ")
}

// ToolInfosToLLMTools converts goreact core.ToolInfo slice into gochat core.Tool slice
// for native function calling via LLM's Tools parameter.
// Each ToolInfo's Parameters are converted to JSON Schema format.
func ToolInfosToLLMTools(infos []core.ToolInfo) []gochatcore.Tool {
	if len(infos) == 0 {
		return nil
	}
	tools := make([]gochatcore.Tool, 0, len(infos))
	for _, info := range infos {
		params := buildJSONSchemaParams(info.Parameters)
		tools = append(tools, gochatcore.Tool{
			Name:        info.Name,
			Description: info.Description,
			Parameters:  params,
		})
	}
	return tools
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

// BuildSkillsSystemPrompt builds a system-prompt-level skills section.
// Skills contain domain-specific behavioral instructions and should be injected
// into the System Prompt layer (not User Prompt) so they define agent capabilities.
func BuildSkillsSystemPrompt(skills []*core.Skill) string {
	if len(skills) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n<available_capabilities>\n")
	sb.WriteString("The following specialized capabilities (skills) are active for this session.\n")
	sb.WriteString("Each skill provides domain-specific instructions — reference them when planning your approach.\n\n")
	for _, s := range skills {
		sb.WriteString(fmt.Sprintf("- **%s**", s.Name))
		if s.Description != "" {
			sb.WriteString(fmt.Sprintf(": %s", s.Description))
		}
		if s.AllowedTools != "" {
			sb.WriteString(fmt.Sprintf(" [tools: %s]", s.AllowedTools))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("</available_capabilities>\n")
	return sb.String()
}

// BuildCapabilitiesList builds a compact L1 summary of skills for Phase 1 selection prompt.
// Format: "- **name**: description [tools: tool1, tool2]"
func BuildCapabilitiesList(skills []*core.Skill) string {
	if len(skills) == 0 {
		return "(no specialized capabilities available)"
	}
	var sb strings.Builder
	for _, s := range skills {
		sb.WriteString(fmt.Sprintf("- **%s**", s.Name))
		if s.Description != "" {
			sb.WriteString(fmt.Sprintf(": %s", s.Description))
		}
		if s.AllowedTools != "" {
			sb.WriteString(fmt.Sprintf(" [tools: %s]", s.AllowedTools))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// EstimateTokensForTools estimates the total token count for a set of LLM tools
// by measuring their JSON serialization size.
func EstimateTokensForTools(tools []gochatcore.Tool, estimateFn func(string) int) int64 {
	if estimateFn == nil || len(tools) == 0 {
		return 0
	}
	var total int64
	for _, t := range tools {
		b, _ := json.Marshal(t)
		total += int64(estimateFn(string(b)))
	}
	return total
}

// ---------------------------------------------------------------------------
// Skill Activation (Two-Phase Think)
// ---------------------------------------------------------------------------

// ActivatedSkillContext holds the loaded L2 instructions and filtered tools
// for a selected skill, produced by Reactor.ActivateSkill().
type ActivatedSkillContext struct {
	Skill            *core.Skill       // The selected skill (with full L2 instructions)
	Instructions     string            // L2 instructions text (for injection into Phase 2 prompt)
	FilteredTools    []gochatcore.Tool // Tools filtered by skill.AllowedTools (LLM native format)
	FilteredInfos    []core.ToolInfo   // Same tools in goreact ToolInfo format (for display in prompt)
	ResourceBasePath string            // 🆕 P3: skill.RootDir — base path for scripts/, references/, assets/
}

// ActivateSkill loads a skill's L2 instructions and filters available tools
// by its allowed-tools list. Returns an activation context ready for Phase 2 thinking.
//
// If skillName is empty or not found, returns nil with no error (no-skill mode).
// If found but has no allowed-tools restriction, all registered tools are included.
func (r *Reactor) ActivateSkill(skillName string, allToolInfos []core.ToolInfo) (*ActivatedSkillContext, error) {
	if skillName == "" {
		return nil, nil
	}

	chosen, err := r.skillRegistry.GetSkill(skillName)
	if err != nil {
		return nil, fmt.Errorf("skill %q not found: %w", skillName, err)
	}

	filteredInfos := filterToolsByAllowed(allToolInfos, chosen.AllowedTools)
	llmTools := ToolInfosToLLMTools(filteredInfos)

	return &ActivatedSkillContext{
		Skill:            chosen,
		Instructions:     chosen.Instructions,
		FilteredTools:    llmTools,
		FilteredInfos:    filteredInfos,
		ResourceBasePath: chosen.RootDir,
	}, nil
}

// filterToolsByAllowed filters tool infos to only those whose names appear in
// the comma-separated allowedTools string. If allowedTools is empty, returns all.
func filterToolsByAllowed(infos []core.ToolInfo, allowedTools string) []core.ToolInfo {
	if allowedTools == "" || allowedTools == "*" {
		return infos
	}

	allowed := make(map[string]bool)
	for _, name := range strings.Split(allowedTools, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			allowed[name] = true
		}
	}

	var filtered []core.ToolInfo
	for _, info := range infos {
		if allowed[info.Name] {
			filtered = append(filtered, info)
		}
	}
	return filtered
}

// BuildThinkPrompt constructs the Think phase prompt (Phase 2) using Go template.
// It includes classified intent, memory records, user input, and optionally
// an activated skill's L2 instructions with filtered tool list.
func BuildThinkPrompt(input string, intent *Intent, memoryRecords []core.MemoryRecord, actCtx *ActivatedSkillContext) string {
	intentSection := "(no intent)"
	if intent != nil {
		b, _ := json.Marshal(intent)
		intentSection = string(b)
	}

	memorySection := ""
	if len(memoryRecords) > 0 {
		memorySection = core.FormatMemoryRecords(memoryRecords)
	}

	data := thinkPromptData{
		IntentSection: intentSection,
		MemorySection: memorySection,
		Input:         input,
	}

	if actCtx != nil && actCtx.Skill != nil {
		data.HasActiveSkill = true
		data.ActiveSkillName = actCtx.Skill.Name
		data.ActiveSkillDesc = actCtx.Skill.Description
		data.ActiveSkillInstructions = actCtx.Instructions
		data.ResourceBasePath = actCtx.ResourceBasePath

		var toolNames []string
		for _, t := range actCtx.FilteredInfos {
			toolNames = append(toolNames, t.Name)
		}
		data.FilteredToolList = strings.Join(toolNames, ", ")
	}

	result, err := renderThinkPrompt(data)
	if err != nil {
		return fmt.Sprintf("think prompt render error: %v", err)
	}
	return result
}

// BuildSkillSelectPrompt constructs the Skill Selection prompt (Phase 1) using Go template.
// It presents L1 summaries of all available skills and asks LLM to choose one.
func BuildSkillSelectPrompt(input string, intent *Intent, skills []*core.Skill) string {
	intentSection := "(no intent)"
	if intent != nil {
		b, _ := json.Marshal(intent)
		intentSection = string(b)
	}

	result, err := renderSkillSelectPrompt(skillSelectPromptData{
		IntentSection: intentSection,
		Input:         input,
	})
	if err != nil {
		return fmt.Sprintf("skill select prompt render error: %v", err)
	}
	return result
}
