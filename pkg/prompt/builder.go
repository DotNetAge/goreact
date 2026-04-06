package prompt

import (
	"context"
	"fmt"
	"strings"
)

// PromptBuilder interface defines the contract for prompt builders
type PromptBuilder interface {
	Build(ctx context.Context, req *BuildRequest) (*Prompt, error)
	BuildPlanPrompt(ctx context.Context, req *PlanPromptRequest) (*Prompt, error)
	BuildThinkPrompt(ctx context.Context, req *ThinkPromptRequest) (*Prompt, error)
	BuildReflectionPrompt(ctx context.Context, req *ReflectionPromptRequest) (*Prompt, error)
	BuildReplanPrompt(ctx context.Context, req *ReplanPromptRequest) (*Prompt, error)
}

// Builder builds prompts for different contexts
type Builder struct {
	templates          map[string]string
	config             *PromptTemplateConfig
	evolutionConfig    *EvolutionPromptConfig
	negativeManager    *NegativePromptManager
	exampleSelector    *ExampleSelector
	contextManager     *ContextManager
	ragInjector        *RAGInjector
	conflictResolver   *ConflictResolver
	questionAnalyzer   *QuestionAnalyzer
}

// NewBuilder creates a new Builder
func NewBuilder(config *PromptTemplateConfig) *Builder {
	if config == nil {
		config = DefaultPromptTemplateConfig()
	}

	contextManager := NewContextManager(config.MaxTokens)
	
	return &Builder{
		templates:        make(map[string]string),
		config:           config,
		evolutionConfig:  DefaultEvolutionConfig(),
		negativeManager:  NewNegativePromptManager(),
		exampleSelector:  NewExampleSelector(3),
		contextManager:   contextManager,
		ragInjector:      NewRAGInjector(contextManager),
		conflictResolver: NewConflictResolver(),
		questionAnalyzer: NewQuestionAnalyzer(),
	}
}

// WithEvolutionConfig sets the evolution config
func (b *Builder) WithEvolutionConfig(config *EvolutionPromptConfig) *Builder {
	b.evolutionConfig = config
	return b
}

// Build builds a complete prompt from the request
func (b *Builder) Build(ctx context.Context, req *BuildRequest) (*Prompt, error) {
	prompt := &Prompt{
		SystemPrompt:   &SystemPrompt{},
		NegativePrompts: make([]*NegativePrompt, 0),
		Tools:          make([]*ToolDefinition, 0),
		Examples:       make([]*Example, 0),
		Sections:       make([]*PromptSection, 0),
		Metadata:       make(map[string]any),
	}

	// 1. Build system prompt
	b.buildSystemPrompt(prompt, req)

	// 2. Inject tools
	b.injectTools(prompt, req)

	// 3. Inject RAG context if needed
	if b.config.EnableRAG && b.questionAnalyzer.ShouldInjectRAG(req.Input) {
		if req.RAGContext != nil {
			b.injectRAGContext(prompt, req.RAGContext)
		}
	}

	// 4. Inject negative prompts
	b.injectNegativePrompts(prompt, req.Permission)

	// 5. Inject few-shot examples
	if b.config.EnableFewShot {
		b.injectFewShotExamples(prompt, req.Input)
	}

	// 6. Set output format
	b.setOutputFormat(prompt)

	// 7. Set user query
	prompt.UserQuery = req.Input
	b.addSection(prompt, &PromptSection{Type: "question", Content: req.Input, Priority: 10})

	// 8. Detect and resolve conflicts
	conflicts := b.conflictResolver.DetectConflicts(prompt)
	if len(conflicts) > 0 {
		resolution := b.conflictResolver.Resolve(conflicts)
		if resolution.HasConflicts() {
			b.addSection(prompt, &PromptSection{
				Type:     "reinforcement",
				Content:  resolution.GetOverrideDirective(),
				Priority: 90,
			})
		}
	}

	// 9. Manage context window
	if err := b.contextManager.Manage(prompt); err != nil {
		return nil, err
	}

	return prompt, nil
}

// BuildPlanPrompt builds a planning prompt
func (b *Builder) BuildPlanPrompt(ctx context.Context, req *PlanPromptRequest) (*Prompt, error) {
	prompt := &Prompt{
		SystemPrompt: &SystemPrompt{
			Role: "规划助手",
			Behavior: `你的职责是为给定任务创建详细的执行计划。

将任务分解为清晰、有序的步骤。对于每个步骤，指定：
1. 要执行的操作
2. 预期结果
3. 与之前步骤的依赖关系

以 JSON 数组格式输出步骤列表。`,
		},
		Tools:    req.Tools,
		Sections: make([]*PromptSection, 0),
	}

	// Build user section
	userSection := fmt.Sprintf(`## 任务
%s

## 可用工具
%s

## 创建执行计划`, req.Input, b.formatTools(req.Tools))

	prompt.UserQuery = userSection

	return prompt, nil
}

// BuildThinkPrompt builds a thinking prompt
func (b *Builder) BuildThinkPrompt(ctx context.Context, req *ThinkPromptRequest) (*Prompt, error) {
	prompt := &Prompt{
		SystemPrompt: &SystemPrompt{
			Role: "推理引擎",
			Behavior: `分析当前状态并决定下一步行动。

你的响应必须是 JSON 格式：
{
  "content": "你的完整思考",
  "reasoning": "你的推理过程",
  "decision": "act" 或 "answer",
  "confidence": 0.0-1.0,
  "action": {
    "type": "tool_call|skill_invoke|delegate",
    "target": "目标名称",
    "params": {},
    "reasoning": "为什么选择此行动"
  },
  "finalAnswer": "如果 decision 是 answer，则提供最终答案"
}`,
		},
		Tools:    req.Tools,
		Sections: make([]*PromptSection, 0),
	}

	// Build history section
	// History would come from session in production

	userSection := fmt.Sprintf(`## 当前任务
%s

## 执行计划
%v

## 当前步骤
%d

## 可用工具
%s

## 做出决定`, req.Input, req.Plan, req.CurrentStep, b.formatTools(req.Tools))

	prompt.UserQuery = userSection

	return prompt, nil
}

// BuildReflectionPrompt builds a reflection prompt
func (b *Builder) BuildReflectionPrompt(ctx context.Context, req *ReflectionPromptRequest) (*Prompt, error) {
	prompt := &Prompt{
		SystemPrompt: &SystemPrompt{
			Role: "反思引擎",
			Behavior: `分析失败的执行并提供见解。

你的响应必须是 JSON 格式：
{
  "failure_reason": "执行失败的原因",
  "analysis": "出错环节的详细分析",
  "heuristic": "总结的可复用经验",
  "suggestions": ["建议1", "建议2", ...],
  "score": 0.0-1.0
}`,
		},
		Sections: make([]*PromptSection, 0),
	}

	errorMsg := req.ErrorMessage
	if req.Error != nil {
		errorMsg = req.Error.Error()
	}

	userSection := fmt.Sprintf(`## 原始任务
分析执行轨迹

## 执行轨迹
%v

## 错误信息
%s

## 分析问题并提出改进建议`, req.Trajectory, errorMsg)

	prompt.UserQuery = userSection

	return prompt, nil
}

// BuildReplanPrompt builds a replanning prompt
func (b *Builder) BuildReplanPrompt(ctx context.Context, req *ReplanPromptRequest) (*Prompt, error) {
	prompt := &Prompt{
		SystemPrompt: &SystemPrompt{
			Role: "重规划引擎",
			Behavior: `根据当前执行情况重新规划。

## 重规划要求
1. 保留已成功的步骤
2. 调整失败步骤的执行方式
3. 必要时添加新的步骤
4. 确保新计划能够达成目标

## 输出格式
Step {current_step}: [调整后的步骤] -> [工具名]
Step {current_step+1}: [新步骤] -> [工具名]
...`,
		},
		Sections: make([]*PromptSection, 0),
	}

	// Format reflections
	reflectionsStr := ""
	if len(req.Reflections) > 0 {
		for i, r := range req.Reflections {
			reflectionsStr += fmt.Sprintf("%d. %s\n", i+1, r.Heuristic)
		}
	}

	userSection := fmt.Sprintf(`## 原始目标
%s

## 原始计划
%v

## 当前进度
已完成 %d 步

## 失败原因
%s

## 反思建议
%s

## 已完成步骤结果
%v

## 创建新的执行计划`, 
		req.OriginalGoal,
		req.OriginalPlan,
		req.CurrentStep,
		req.FailureReason,
		reflectionsStr,
		req.CompletedSteps)

	prompt.UserQuery = userSection

	return prompt, nil
}

// buildSystemPrompt builds the system prompt section
func (b *Builder) buildSystemPrompt(prompt *Prompt, req *BuildRequest) {
	template := b.templates["system"]
	if template == "" {
		template = DefaultSystemTemplate
	}

	prompt.SystemPrompt = &SystemPrompt{
		Role:         "智能助手",
		Behavior:     template,
		Constraints:  "",
		OutputFormat: b.config.OutputFormat.Format,
	}

	b.addSection(prompt, &PromptSection{
		Type:     "system",
		Content:  template,
		Priority: 100,
	})
}

// injectTools injects tools into the prompt
func (b *Builder) injectTools(prompt *Prompt, req *BuildRequest) {
	if len(req.Tools) == 0 {
		return
	}

	prompt.Tools = req.Tools

	toolSection := "# 可用工具\n\n"
	for _, tool := range req.Tools {
		toolSection += b.formatTool(tool)
	}

	b.addSection(prompt, &PromptSection{
		Type:     "tools",
		Content:  toolSection,
		Priority: 80,
	})
}

// formatTool formats a single tool definition
func (b *Builder) formatTool(tool *ToolDefinition) string {
	result := "### " + tool.Name + "\n"
	result += "- 描述: " + tool.Description + "\n"
	if len(tool.Examples) > 0 {
		result += "- 示例: " + strings.Join(tool.Examples, ", ") + "\n"
	}
	return result + "\n"
}

// formatTools formats multiple tools
func (b *Builder) formatTools(tools []*ToolDefinition) string {
	var result []string
	for _, tool := range tools {
		result = append(result, tool.Name+": "+tool.Description)
	}
	return strings.Join(result, "\n")
}

// injectRAGContext injects RAG context into the prompt
func (b *Builder) injectRAGContext(prompt *Prompt, context *RAGContext) {
	if context == nil {
		return
	}

	// Optimize context for token limits
	context = b.ragInjector.OptimizeContext(context, b.config.MaxTokens/4)

	// Get recommended injection strategy
	strategy := b.questionAnalyzer.GetRecommendedStrategy(prompt.UserQuery)
	
	// Inject using RAGInjector
	b.ragInjector.Inject(prompt, context, strategy)
}

// injectNegativePrompts injects negative prompts based on permission
func (b *Builder) injectNegativePrompts(prompt *Prompt, permission *Permission) {
	// Get filtered prompts
	prompts := b.negativeManager.FilterByPermission(permission)
	prompt.NegativePrompts = prompts

	if len(prompts) == 0 {
		return
	}

	// Format and add as section
	negSection := b.negativeManager.FormatPrompts(prompts)
	b.addSection(prompt, &PromptSection{
		Type:     "negative",
		Content:  negSection,
		Priority: 95,
	})
}

// injectFewShotExamples injects few-shot examples
func (b *Builder) injectFewShotExamples(prompt *Prompt, query string) {
	// Determine difficulty and example count
	difficulty := b.questionAnalyzer.GetDifficulty(query)
	exampleCount := b.questionAnalyzer.GetRecommendedExampleCount(query)

	// Select examples
	opts := SelectOptions{
		Difficulty: difficulty,
	}
	examples := b.exampleSelector.Select(query, opts)

	// Limit examples
	if len(examples) > exampleCount {
		examples = examples[:exampleCount]
	}

	if len(examples) == 0 {
		return
	}

	prompt.Examples = examples

	// Format and add as section
	exampleSection := FormatExamples(examples)
	b.addSection(prompt, &PromptSection{
		Type:     "examples",
		Content:  exampleSection,
		Priority: 40,
	})
}

// setOutputFormat sets the output format
func (b *Builder) setOutputFormat(prompt *Prompt) {
	prompt.OutputFormat = b.config.OutputFormat

	formatSection := "# 输出格式\n\n" + b.config.OutputFormat.Format
	b.addSection(prompt, &PromptSection{
		Type:     "format",
		Content:  formatSection,
		Priority: 30,
	})
}

// addSection adds a section to the prompt
func (b *Builder) addSection(prompt *Prompt, section *PromptSection) {
	prompt.Sections = append(prompt.Sections, section)
}

// String returns the assembled prompt as a string
func (p *Prompt) String() string {
	var sb strings.Builder

	// Sort sections by priority (higher first)
	sorted := make([]*PromptSection, len(p.Sections))
	copy(sorted, p.Sections)
	
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Priority > sorted[i].Priority {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Write sections
	for _, section := range sorted {
		sb.WriteString(section.Content)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// InjectReflections injects reflections into the prompt
func (b *Builder) InjectReflections(prompt *Prompt, reflections []string) *Prompt {
	if len(reflections) == 0 {
		return prompt
	}

	var sb strings.Builder
	sb.WriteString("\n## Previous Learnings\n")
	for _, r := range reflections {
		sb.WriteString("- ")
		sb.WriteString(r)
		sb.WriteString("\n")
	}
	injection := sb.String()

	b.addSection(prompt, &PromptSection{
		Type:     "reflections",
		Content:  injection,
		Priority: 60,
	})

	return prompt
}

// InjectNegativePrompt injects negative prompts (constraints)
func (b *Builder) InjectNegativePrompt(prompt *Prompt, negatives []string) *Prompt {
	if len(negatives) == 0 {
		return prompt
	}

	var sb strings.Builder
	sb.WriteString("\n## Behavior Constraints\n")
	for _, n := range negatives {
		sb.WriteString("- ")
		sb.WriteString(n)
		sb.WriteString("\n")
	}
	injection := sb.String()

	b.addSection(prompt, &PromptSection{
		Type:     "negative_custom",
		Content:  injection,
		Priority: 94,
	})

	return prompt
}

// Default templates
const DefaultSystemTemplate = `你是一个智能助手，能够通过思考和行动来帮助用户完成任务。

你可以使用各种工具和技能来实现目标。
在行动之前请仔细思考，从错误中学习。

## 工作方式
1. 思考：分析当前情况，确定下一步行动
2. 行动：选择合适的工具执行操作
3. 观察：查看执行结果，更新理解
4. 重复以上步骤直到任务完成`
