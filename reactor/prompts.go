package reactor

import (
	"bytes"
	"embed"
	"encoding/json"
	"strings"
	"text/template"

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
	"skillSection": func(skills []*core.Skill) string {
		if len(skills) == 0 {
			return ""
		}
		var sb strings.Builder
		sb.WriteString("\n<activated_skills>\n")
		for _, s := range skills {
			sb.WriteString("<skill name=\"" + s.Name + "\"")
			if s.AllowedTools != "" {
				sb.WriteString(" allowed-tools=\"" + s.AllowedTools + "\"")
			}
			if s.Source != "" {
				sb.WriteString(" source=\"" + s.Source + "\"")
			}
			sb.WriteString(">\n")
			if s.Description != "" {
				sb.WriteString("<description>" + s.Description + "</description>\n")
			}
			if s.Instructions != "" {
				sb.WriteString("<instructions>\n" + s.Instructions + "\n</instructions>\n")
			}
			sb.WriteString("</skill>\n")
		}
		sb.WriteString("</activated_skills>\n")
		return sb.String()
	},
}

// templateNames maps the embed path to the parsed template name used for Lookup.
// Go's template.ParseFS strips the directory prefix, so "prompts/summary_prompt.tmpl"
// becomes associated name "summary_prompt.tmpl".
const (
	tmplIntent   = "intent_prompt.tmpl"
	tmplThink    = "think_prompt.tmpl"
	tmplSystem   = "default_system_prompt.tmpl"
	tmplSummary  = "summary_prompt.tmpl"
)

// intentPromptTemplate is parsed once at init from the embedded .tmpl file.
var intentPromptTemplate = template.Must(
	template.New("intent_prompt").Funcs(promptFuncMap).ParseFS(promptTemplates, "prompts/intent_prompt.tmpl"),
)

// thinkPromptTemplate is parsed once at init from the embedded .tmpl file.
var thinkPromptTemplate = template.Must(
	template.New("think_prompt").Funcs(promptFuncMap).ParseFS(promptTemplates, "prompts/think_prompt.tmpl"),
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

// thinkPromptData holds template variables for the Think phase prompt.
type thinkPromptData struct {
	IntentSection string
	ToolSection   string
	Skills        []*core.Skill
	MemorySection string // relevant memory records for hallucination suppression
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
