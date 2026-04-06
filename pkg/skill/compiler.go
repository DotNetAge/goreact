package skill

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/DotNetAge/gochat/pkg/core"
)

// Compiler compiles skill templates into execution plans
type Compiler struct {
	templateDelims []string
	llmClient      core.Client
}

// NewCompiler creates a new Compiler
func NewCompiler() *Compiler {
	return &Compiler{
		templateDelims: []string{"{{", "}}"},
	}
}

// WithLLM sets the LLM client for entity extraction
func (c *Compiler) WithLLM(llm core.Client) *Compiler {
	c.llmClient = llm
	return c
}

// Compile compiles a skill into an execution plan
func (c *Compiler) Compile(ctx context.Context, skill *Skill) (*SkillExecutionPlan, error) {
	plan := NewSkillExecutionPlan(skill.Name)
	
	// Copy steps
	for _, step := range skill.Steps {
		plan.Steps = append(plan.Steps, step)
	}
	
	// Copy parameters
	for _, param := range skill.Parameters {
		plan.Parameters = append(plan.Parameters, ParameterSpec{
			Name:        param.Name,
			Type:        param.Type,
			Required:    param.Required,
			Default:     param.Default,
			Description: param.Description,
		})
	}
	
	// If skill has template, parse it for steps
	if skill.Template != "" {
		steps, err := c.parseTemplate(skill.Template)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template: %w", err)
		}
		plan.Steps = steps
	}
	
	plan.CompiledAt = time.Now()
	return plan, nil
}

// parseTemplate parses a skill template and extracts execution steps
func (c *Compiler) parseTemplate(templateContent string) ([]ExecutionStep, error) {
	steps := []ExecutionStep{}
	
	// Parse the template looking for tool invocations
	// Support SKILL.md format with steps defined in markdown
	
	lines := strings.Split(templateContent, "\n")
	stepIndex := 0
	inStepsSection := false
	
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		
		// Detect steps section
		if strings.HasPrefix(trimmedLine, "## Steps") || strings.HasPrefix(trimmedLine, "##步骤") {
			inStepsSection = true
			continue
		}
		
		// End steps section at next heading
		if inStepsSection && strings.HasPrefix(trimmedLine, "##") {
			inStepsSection = false
			continue
		}
		
		if !inStepsSection {
			continue
		}
		
		// Parse numbered steps
		if matched, _ := regexp.MatchString(`^\d+\.\s+.+`, trimmedLine); matched {
			// Extract step description
			re := regexp.MustCompile(`^\d+\.\s+(.+)$`)
			matches := re.FindStringSubmatch(trimmedLine)
			if len(matches) > 1 {
				description := matches[1]
				
				// Detect tool invocations in the step
				toolName := ""
				paramsTemplate := make(map[string]any)
				
				// Check for tool call pattern: tool_name(args)
				toolPattern := regexp.MustCompile(`(\w+)\s*\(([^)]*)\)`)
				toolMatches := toolPattern.FindStringSubmatch(description)
				if len(toolMatches) > 2 {
					toolName = toolMatches[1]
					// Parse parameters
					paramsStr := toolMatches[2]
					for _, param := range strings.Split(paramsStr, ",") {
						kv := strings.SplitN(strings.TrimSpace(param), "=", 2)
						if len(kv) == 2 {
							key := strings.TrimSpace(kv[0])
							value := strings.TrimSpace(kv[1])
							// Remove quotes if present
							value = strings.Trim(value, "\"'")
							paramsTemplate[key] = value
						}
					}
				}
				
				step := ExecutionStep{
					Index:          stepIndex,
					ToolName:       toolName,
					Description:    description,
					ParamsTemplate: paramsTemplate,
					Condition:      "",
					OnError:        "stop",
				}
				
				steps = append(steps, step)
				stepIndex++
			}
		}
	}
	
	// Also look for inline tool calls outside of numbered steps
	toolCallPattern := regexp.MustCompile("`([a-zA-Z_][a-zA-Z0-9_]*)\\s*\\(([^)]*)\\)`")
	matches := toolCallPattern.FindAllStringSubmatch(templateContent, -1)
	for i, match := range matches {
		if len(match) >= 3 {
			// Check if this tool call is already in steps
			found := false
			for _, step := range steps {
				if step.ToolName == match[1] {
					found = true
					break
				}
			}
			
			if !found && match[1] != "" {
				paramsTemplate := make(map[string]any)
				paramsStr := match[2]
				for _, param := range strings.Split(paramsStr, ",") {
					kv := strings.SplitN(strings.TrimSpace(param), "=", 2)
					if len(kv) == 2 {
						key := strings.TrimSpace(kv[0])
						value := strings.TrimSpace(kv[1])
						value = strings.Trim(value, "\"'")
						paramsTemplate[key] = value
					}
				}
				
				step := ExecutionStep{
					Index:          len(steps) + i,
					ToolName:       match[1],
					Description:    fmt.Sprintf("Execute %s", match[1]),
					ParamsTemplate: paramsTemplate,
					Condition:      "",
					OnError:        "stop",
				}
				
				steps = append(steps, step)
			}
		}
	}
	
	return steps, nil
}

// RenderParams renders parameter templates with the given context
func (c *Compiler) RenderParams(paramsTemplate map[string]any, context map[string]any) (map[string]any, error) {
	result := make(map[string]any)
	
	for key, value := range paramsTemplate {
		rendered, err := c.renderValue(value, context)
		if err != nil {
			return nil, fmt.Errorf("failed to render parameter %s: %w", key, err)
		}
		result[key] = rendered
	}
	
	return result, nil
}

// renderValue renders a single value
func (c *Compiler) renderValue(value any, context map[string]any) (any, error) {
	switch v := value.(type) {
	case string:
		return c.renderString(v, context)
	case map[string]any:
		result := make(map[string]any)
		for key, val := range v {
			rendered, err := c.renderValue(val, context)
			if err != nil {
				return nil, err
			}
			result[key] = rendered
		}
		return result, nil
	case []any:
		result := make([]any, len(v))
		for i, val := range v {
			rendered, err := c.renderValue(val, context)
			if err != nil {
				return nil, err
			}
			result[i] = rendered
		}
		return result, nil
	default:
		return value, nil
	}
}

// renderString renders a string template
func (c *Compiler) renderString(templateStr string, context map[string]any) (string, error) {
	tmpl, err := template.New("param").Delims(c.templateDelims[0], c.templateDelims[1]).Parse(templateStr)
	if err != nil {
		return templateStr, nil // Return as-is if not a valid template
	}
	
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, context); err != nil {
		return "", err
	}
	
	return buf.String(), nil
}

// RuntimeContextResolver resolves runtime context for skill execution
type RuntimeContextResolver struct {
	llmClient core.Client
}

// NewRuntimeContextResolver creates a new RuntimeContextResolver
func NewRuntimeContextResolver(llm core.Client) *RuntimeContextResolver {
	return &RuntimeContextResolver{llmClient: llm}
}

// Resolve resolves parameters from the current state and input
func (r *RuntimeContextResolver) Resolve(ctx context.Context, plan *SkillExecutionPlan, input string, state map[string]any) (map[string]any, error) {
	params := make(map[string]any)
	
	// If LLM is available, use it for entity extraction
	if r.llmClient != nil {
		entities, err := r.extractEntities(ctx, input, plan.Parameters)
		if err == nil {
			for k, v := range entities {
				params[k] = v
			}
		}
	}
	
	// Override with state values
	for _, spec := range plan.Parameters {
		// Try to get from state first
		if value, exists := state[spec.Name]; exists {
			params[spec.Name] = value
			continue
		}
		
		// Use default if available and not already set
		if _, ok := params[spec.Name]; !ok && spec.Default != nil {
			params[spec.Name] = spec.Default
		}
	}
	
	return params, nil
}

// extractEntities extracts entities from input using LLM
func (r *RuntimeContextResolver) extractEntities(ctx context.Context, input string, params []ParameterSpec) (map[string]any, error) {
	if r.llmClient == nil {
		return nil, fmt.Errorf("no LLM client available")
	}
	
	// Build extraction prompt
	prompt := "Extract the following parameters from this input:\n"
	for _, p := range params {
		prompt += fmt.Sprintf("- %s (%s): %s\n", p.Name, p.Type, p.Description)
	}
	prompt += "\nInput: " + input + "\n\nRespond in JSON format."
	
	resp, err := r.llmClient.Chat(ctx, []core.Message{
		core.NewUserMessage(prompt),
	})
	if err != nil {
		return nil, err
	}
	
	// Parse JSON response
	// Simplified: return the response content as a single value
	return map[string]any{
		"extracted": resp.Content,
	}, nil
}
