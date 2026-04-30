package orchestration

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"

	"github.com/DotNetAge/goreact/core"
)

// promptTemplates is the embedded filesystem containing orchestration prompt template files.
// Templates are stored as .tmpl files under the prompts/ directory.
//
//go:embed prompts/*.tmpl
var promptTemplates embed.FS

// Template file name constants for Lookup.
const (
	tmplRouting              = "routing_prompt.tmpl"
	tmplCapabilityExtraction = "capability_extraction_prompt.tmpl"
	tmplBodyGeneration       = "body_generation_prompt.tmpl"
	tmplWBSDecomposition      = "wbs_decomposition_prompt.tmpl"
)

// --- Parsed templates (init once) ---

var routingPromptTemplate = template.Must(
	template.New("routing_prompt").Funcs(templateFuncMap).ParseFS(promptTemplates, "prompts/routing_prompt.tmpl"),
)

var capabilityExtractionPromptTemplate = template.Must(
	template.New("capability_extraction_prompt").Funcs(templateFuncMap).ParseFS(promptTemplates, "prompts/capability_extraction_prompt.tmpl"),
)

var bodyGenerationPromptTemplate = template.Must(
	template.New("body_generation_prompt").Funcs(templateFuncMap).ParseFS(promptTemplates, "prompts/body_generation_prompt.tmpl"),
)

var wbsDecompositionPromptTemplate = template.Must(
	template.New("wbs_decomposition_prompt").Funcs(templateFuncMap).ParseFS(promptTemplates, "prompts/wbs_decomposition_prompt.tmpl"),
)

// templateFuncMap defines custom template functions used across all orchestration prompts.
var templateFuncMap = template.FuncMap{}

// ===========================================================================
// Template Data Types
// ===========================================================================

// routingPromptData holds template variables for the LLM Router system prompt (Design §6.3).
type routingPromptData struct {
	Agents            []agentMetadataView
	TaskDescription   string
	DesiredCapability string
}

// agentMetadataView is a lightweight view of agent metadata for the router prompt.
type agentMetadataView struct {
	Name        string
	Description string
	State       string
	ScoreFormatted string
	TaskCount   int64
}

// capabilityExtractionPromptData holds template variables for AgentFactory's
// capability extraction prompt (Design §12.2.1 Step 1).
type capabilityExtractionPromptData struct {
	TaskDescription string
}

// bodyGenerationPromptData holds template variables for AgentFactory's body/system-prompt
// generation prompt (Design §12.2.1 Step 2).
type bodyGenerationPromptData struct {
	Name          string
	Description   string
	Capabilities  string // comma-joined
	TaskExample   string
	// MatchedSkills contains skills selected for this agent based on its domain/capability scope.
	// Skills' metadata (name + description) are pre-loaded into system prompts to free up context window.
	MatchedSkills []*core.Skill
}

// wbsDecompositionPromptData holds template variables for WBS decomposition judgment
// prompt (Design §11.2, inserted at Think Phase Step B).
type wbsDecompositionPromptData struct {
	TaskDescription      string
	AvailableCapabilities string
}

// ===========================================================================
// Render Functions
// ===========================================================================

// renderRoutingPrompt renders the LLM Router system prompt using the embedded Go template.
func renderRoutingPrompt(data routingPromptData) (string, error) {
	t := routingPromptTemplate.Lookup(tmplRouting)
	if t == nil {
		return "", template.ExecError{Name: tmplRouting, Err: nil}
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderCapabilityExtractionPrompt renders the capability extraction prompt using the embedded template.
func renderCapabilityExtractionPrompt(data capabilityExtractionPromptData) (string, error) {
	t := capabilityExtractionPromptTemplate.Lookup(tmplCapabilityExtraction)
	if t == nil {
		return "", template.ExecError{Name: tmplCapabilityExtraction, Err: nil}
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderBodyGenerationPrompt renders the body generation prompt using the embedded template.
func renderBodyGenerationPrompt(data bodyGenerationPromptData) (string, error) {
	t := bodyGenerationPromptTemplate.Lookup(tmplBodyGeneration)
	if t == nil {
		return "", template.ExecError{Name: tmplBodyGeneration, Err: nil}
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderWBSDecompositionPrompt renders the WBS decomposition judgment prompt using the embedded template.
func renderWBSDecompositionPrompt(data wbsDecompositionPromptData) (string, error) {
	t := wbsDecompositionPromptTemplate.Lookup(tmplWBSDecomposition)
	if t == nil {
		return "", template.ExecError{Name: tmplWBSDecomposition, Err: nil}
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// toAgentViews converts []*core.AgentRuntimeMeta to []agentMetadataView for template rendering.
func toAgentViews(agents []*core.AgentRuntimeMeta) []agentMetadataView {
	if len(agents) == 0 {
		return nil
	}
	views := make([]agentMetadataView, len(agents))
	for i, a := range agents {
		views[i] = agentMetadataView{
			Name:           a.Name(),
			Description:    a.Description(),
			State:          string(a.State),
			ScoreFormatted: formatScore(a.Score),
			TaskCount:      a.TaskCount,
		}
	}
	return views
}

func formatScore(score float64) string {
	return fmt.Sprintf("%.2f", score)
}
