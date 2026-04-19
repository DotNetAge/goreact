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
			sb.WriteString("## Skill: " + s.Name + "\n" + s.Instructions + "\n")
		}
		sb.WriteString("</activated_skills>\n")
		return sb.String()
	},
}

// intentPromptTemplate is parsed once at init from the embedded .tmpl file.
var intentPromptTemplate = template.Must(
	template.New("intent_prompt").Funcs(promptFuncMap).ParseFS(promptTemplates, "prompts/intent_prompt.tmpl"),
)

// thinkPromptTemplate is parsed once at init from the embedded .tmpl file.
var thinkPromptTemplate = template.Must(
	template.New("think_prompt").Funcs(promptFuncMap).ParseFS(promptTemplates, "prompts/think_prompt.tmpl"),
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
	Input         string
}

// renderIntentPrompt renders the intent prompt using the embedded Go template.
func renderIntentPrompt(data intentPromptData) (string, error) {
	t := intentPromptTemplate.Lookup("prompts/intent_prompt.tmpl")
	if t == nil {
		return "", template.ExecError{Name: "prompts/intent_prompt.tmpl", Err: nil}
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderThinkPrompt renders the Think prompt using the embedded Go template.
func renderThinkPrompt(data thinkPromptData) (string, error) {
	t := thinkPromptTemplate.Lookup("prompts/think_prompt.tmpl")
	if t == nil {
		return "", template.ExecError{Name: "prompts/think_prompt.tmpl", Err: nil}
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
