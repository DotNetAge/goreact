package thinker

import (
	"fmt"
	"strings"
	"time"

	"github.com/DotNetAge/gochat/pkg/core"
	reactCore "github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/memory"
	"github.com/ray/goreact/pkg/prompt/builder"
	"github.com/ray/goreact/pkg/prompt/compression"
	"github.com/ray/goreact/pkg/prompt/counter"
	"github.com/ray/goreact/pkg/prompt/formatter"
	"github.com/ray/goreact/pkg/thinker/parser"
	"github.com/ray/goreact/pkg/thinker/prompt"
	"github.com/ray/goreact/pkg/tools"
)

// Default Thinker Implementation.
type defaultThinker struct {
	llmClient   core.Client
	modelName   string
	toolManager tools.Manager
	memoryBank  memory.MemoryBank
	sysTemplate string // Changed to raw string for builder
}

// Option configures a defaultThinker
type Option func(*defaultThinker)

func WithModel(model string) Option {
	return func(t *defaultThinker) {
		t.modelName = model
	}
}

func WithToolManager(mgr tools.Manager) Option {
	return func(t *defaultThinker) {
		t.toolManager = mgr
	}
}

func WithMemoryBank(bank memory.MemoryBank) Option {
	return func(t *defaultThinker) {
		t.memoryBank = bank
	}
}

func WithSystemPrompt(tpl string) Option {
	return func(t *defaultThinker) {
		if tpl != "" {
			t.sysTemplate = tpl
		}
	}
}

func Default(client core.Client, opts ...Option) Thinker {
	t := &defaultThinker{
		llmClient:   client,
		modelName:   "gpt-4",
		sysTemplate: prompt.ReActSystemPrompt,
		toolManager: tools.NewSimpleManager(),
	}

	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (t *defaultThinker) Think(ctx *reactCore.PipelineContext) error {
	ctx.Logger.Info("Thinker is reasoning...", "step", ctx.CurrentStep)
	start := time.Now()

	// Sub-pipeline Step 0: Self-Maintenance Commands (Short-circuit)
	if strings.HasPrefix(ctx.Input, "/clear") {
		ctx.Traces = nil
		ctx.IsFinished = true
		ctx.FinalResult = "Context cleared. I'm ready for a fresh start."
		ctx.FinishReason = "ContextReset"
		return nil
	}

	// Sub-pipeline Step 1: Intent & Mode Resolution (Codewords)
	mode, template := t.resolveMode(ctx.Input)

	// Sub-pipeline Step 2: Tool Discovery
	availableTools, err := t.toolManager.ListAvailableTools(ctx, ctx.Input)
	if err != nil {
		return fmt.Errorf("failed to list available tools: %w", err)
	}

	// Sub-pipeline Step 3: Prompt Synthesis using pkg/prompt/builder
	pb := t.createPromptBuilder(ctx, availableTools, template)

	// If it's a forced compress command, we can apply more aggressive strategy
	if strings.HasPrefix(ctx.Input, "/compress") {
		pb.WithCompression(compression.NewSlidingWindowStrategy(1)) // Aggressive!
	}

	builtPrompt := pb.Build()

	// Sub-pipeline Step 4: LLM Execution
	messages := []core.Message{
		{Role: core.RoleSystem, Content: []core.ContentBlock{{Type: core.ContentTypeText, Text: builtPrompt.System}}},
		{Role: core.RoleUser, Content: []core.ContentBlock{{Type: core.ContentTypeText, Text: builtPrompt.User}}},
	}

	llmOpts := []core.Option{
		core.WithModel(t.modelName),
		core.WithUsageCallback(func(usage core.Usage) {
			ctx.TotalTokens.Add(usage.PromptTokens, usage.CompletionTokens)
		}),
		core.WithAttachments(ctx.Attachments...),
	}

	rawResponse, err := t.callLLM(ctx, messages, llmOpts)
	if err != nil {
		return err
	}

	ctx.Metrics.RecordTimer("thinker_latency", time.Since(start), nil)

	// Sub-pipeline Step 5: Mode-specific Parsing
	return t.processOutput(ctx, mode, rawResponse)
}

// resolveMode detects "Codewords" like /plan or /specs
func (t *defaultThinker) resolveMode(input string) (string, string) {
	if strings.HasPrefix(input, "/plan") {
		return "plan", prompt.PlanningSystemPrompt
	}
	if strings.HasPrefix(input, "/specs") {
		return "specs", prompt.SpecsSystemPrompt
	}
	if strings.HasPrefix(input, "/json") {
		return "json", t.sysTemplate
	}
	return "react", t.sysTemplate
}

// createPromptBuilder internal helper to setup the builder
func (t *defaultThinker) createPromptBuilder(ctx *reactCore.PipelineContext, currentTools []tools.Tool, sysTpl string) *builder.FluentPromptBuilder {
	pb := builder.New().
		WithSystemTemplate(sysTpl).
		WithTask(ctx.Input).
		WithMaxTokens(4000).
		WithTokenCounter(counter.NewUniversalEstimator("mixed")).
		WithCompression(compression.NewSlidingWindowStrategy(10))

	// 1. Transform tools to ToolDesc for the formatter
	var toolDescs []formatter.ToolDesc
	var toolNames []string
	for _, tool := range currentTools {
		toolDescs = append(toolDescs, formatter.ToolDesc{
			Name:        tool.Name(),
			Description: tool.Description(),
		})
		toolNames = append(toolNames, tool.Name())
	}
	pb.WithTools(toolDescs)
	pb.WithVariable("ToolNames", strings.Join(toolNames, ", "))

	// 2. Format ReAct Traces into Builder History
	var history []builder.Turn
	for _, trace := range ctx.Traces {
		assistantContent := ""
		if trace.Thought != "" {
			assistantContent += fmt.Sprintf("Thought: %s\n", trace.Thought)
		}
		if trace.Action != nil {
			assistantContent += fmt.Sprintf("Action: %s\nActionInput: %v\n", trace.Action.Name, trace.Action.Input)
		}
		if assistantContent != "" {
			history = append(history, builder.Turn{Role: "assistant", Content: assistantContent})
		}

		if trace.Observation != nil {
			obsText := fmt.Sprintf("Observation (Status: %v):\n%s", trace.Observation.IsSuccess, trace.Observation.Data)
			history = append(history, builder.Turn{Role: "user", Content: obsText})
		}
	}
	pb.WithHistory(history)

	// 3. Inject Memories
	if t.memoryBank != nil {
		memories := make(map[string]string)
		if w, _ := t.memoryBank.Working().RecallContext(ctx, ctx.SessionID, ctx.Input); w != "" {
			memories["Working"] = w
		}
		if s, _ := t.memoryBank.Semantic().RecallKnowledge(ctx, ctx.Input); s != "" {
			memories["Semantic"] = s
		}
		pb.WithVariable("Memories", memories)
	}
	return pb
}

func (t *defaultThinker) callLLM(ctx *reactCore.PipelineContext, messages []core.Message, opts []core.Option) (string, error) {
	stream, err := t.llmClient.ChatStream(ctx, messages, opts...)
	if err != nil {
		return "", err
	}
	defer stream.Close()

	var sb strings.Builder
	for stream.Next() {
		ev := stream.Event()
		if ev.Type == core.EventContent || ev.Type == core.EventThinking {
			sb.WriteString(ev.Content)
			if ctx.OnThoughtStream != nil {
				ctx.OnThoughtStream(ev.Content)
			}
		}
	}
	return sb.String(), nil
}

func (t *defaultThinker) processOutput(ctx *reactCore.PipelineContext, mode, raw string) error {
	if mode == "plan" {
		tasks, err := parser.ParsePlan(raw)
		if err == nil {
			ctx.Logger.Info("Plan parsed into structured tasks", "count", len(tasks))
			// Here we could dynamically build a Sub-Pipeline and attach it to context
			// For now, we store them in PlanSteps for the Driving Force to consume
			for _, task := range tasks {
				ctx.PlanSteps = append(ctx.PlanSteps, task.Task)
			}
			ctx.IsFinished = false       // Not finished! We need to execute the plan.
			ctx.Input = ctx.PlanSteps[0] // Switch input to the first task
			ctx.CurrentPlan = 0
			return nil
		}

		// Fallback if parsing fails
		ctx.IsFinished = true
		ctx.FinalResult = raw
		return nil
	}

	if mode == "json" || mode == "specs" {
		ctx.IsFinished = true
		ctx.FinalResult = raw
		ctx.FinishReason = "DirectOutput"
		return nil
	}

	trace, finalAnswer, isFinished, parseErr := parser.ParseLLMOutput(raw)
	if parseErr != nil {
		// Auto-recovery reflection
		ctx.AppendTrace(&reactCore.Trace{
			Step:    ctx.CurrentStep,
			Thought: "(Format Error Reflection)",
			Observation: &reactCore.Observation{
				Data:      fmt.Sprintf("SYSTEM ERROR: Format mismatch. Err: %v.", parseErr),
				IsSuccess: false,
			},
		})
		return nil
	}

	if isFinished {
		ctx.IsFinished = true
		ctx.FinalResult = finalAnswer
		ctx.FinishReason = "TaskComplete"
	}

	if trace != nil {
		trace.Step = ctx.CurrentStep
		ctx.AppendTrace(trace)
	}

	return nil
}
