// Package prompt provides prompt building for the goreact framework.
package prompt

import (
	"time"
)

// Prompt represents a complete prompt for LLM interaction
type Prompt struct {
	// SystemPrompt is the system-level prompt defining agent identity
	SystemPrompt *SystemPrompt `json:"system_prompt" yaml:"system_prompt"`
	
	// NegativePrompts are constraints to suppress unwanted behaviors
	NegativePrompts []*NegativePrompt `json:"negative_prompts" yaml:"negative_prompts"`
	
	// RAGContext contains retrieved knowledge
	RAGContext *RAGContext `json:"rag_context" yaml:"rag_context"`
	
	// Tools are available tools for the agent
	Tools []*ToolDefinition `json:"tools" yaml:"tools"`
	
	// Examples are few-shot learning examples
	Examples []*Example `json:"examples" yaml:"examples"`
	
	// OutputFormat defines the expected output format
	OutputFormat *OutputFormat `json:"output_format" yaml:"output_format"`
	
	// UserQuery is the user's question/request
	UserQuery string `json:"user_query" yaml:"user_query"`
	
	// Sections are ordered sections for final assembly
	Sections []*PromptSection `json:"sections" yaml:"sections"`
	
	// Metadata contains additional information
	Metadata map[string]any `json:"metadata" yaml:"metadata"`
	
	// TokenCount is the estimated token count
	TokenCount int `json:"token_count" yaml:"token_count"`
}

// SystemPrompt defines the agent's identity and behavior
type SystemPrompt struct {
	// Role defines the agent's role
	Role string `json:"role" yaml:"role"`
	
	// Behavior defines expected behaviors
	Behavior string `json:"behavior" yaml:"behavior"`
	
	// Constraints are constraints on agent behavior
	Constraints string `json:"constraints" yaml:"constraints"`
	
	// Tools are available tools
	Tools []*ToolDefinition `json:"tools" yaml:"tools"`
	
	// OutputFormat defines the output format
	OutputFormat string `json:"output_format" yaml:"output_format"`
}

// NegativePrompt suppresses unwanted model behaviors
type NegativePrompt struct {
	// ID is the unique identifier
	ID string `json:"id" yaml:"id"`
	
	// Description explains what this constraint prevents
	Description string `json:"description" yaml:"description"`
	
	// Pattern is the constraint pattern
	Pattern string `json:"pattern" yaml:"pattern"`
	
	// Reason explains why this constraint exists
	Reason string `json:"reason" yaml:"reason"`
	
	// Alternative suggests what to do instead
	Alternative string `json:"alternative" yaml:"alternative"`
	
	// Severity indicates importance (critical, high, medium, low)
	Severity string `json:"severity" yaml:"severity"`
}

// NegativePromptGroup groups related negative prompts
type NegativePromptGroup struct {
	// ID is the group identifier
	ID string `json:"id" yaml:"id"`
	
	// Name is the group name
	Name string `json:"name" yaml:"name"`
	
	// Description explains the group's purpose
	Description string `json:"description" yaml:"description"`
	
	// Prompts are the negative prompts in this group
	Prompts []*NegativePrompt `json:"prompts" yaml:"prompts"`
	
	// Enabled indicates if this group is active
	Enabled bool `json:"enabled" yaml:"enabled"`
}

// RAGContext contains retrieved knowledge for prompt injection
type RAGContext struct {
	// Query is the original query
	Query string `json:"query" yaml:"query"`
	
	// Mode is the RAG mode (native, graph, hybrid)
	Mode RAGMode `json:"mode" yaml:"mode"`
	
	// Documents are retrieved documents
	Documents []*Document `json:"documents" yaml:"documents"`
	
	// GraphContext contains graph-based knowledge
	GraphContext *GraphContext `json:"graph_context" yaml:"graph_context"`
	
	// Metadata contains additional information
	Metadata map[string]any `json:"metadata" yaml:"metadata"`
	
	// TokenCount is the estimated token count
	TokenCount int `json:"token_count" yaml:"token_count"`
}

// RAGMode represents the RAG retrieval mode
type RAGMode int

const (
	// RAGModeNative uses vector similarity search
	RAGModeNative RAGMode = iota
	// RAGModeGraph uses graph-based retrieval
	RAGModeGraph
	// RAGModeHybrid combines vector and graph retrieval
	RAGModeHybrid
)

// Document represents a retrieved document
type Document struct {
	// ID is the document identifier
	ID string `json:"id" yaml:"id"`
	
	// Content is the document content
	Content string `json:"content" yaml:"content"`
	
	// Source is the document source
	Source string `json:"source" yaml:"source"`
	
	// Score is the relevance score
	Score float64 `json:"score" yaml:"score"`
	
	// Metadata contains additional information
	Metadata map[string]any `json:"metadata" yaml:"metadata"`
}

// GraphContext contains graph-based knowledge
type GraphContext struct {
	// Nodes are graph nodes
	Nodes []*GraphNode `json:"nodes" yaml:"nodes"`
	
	// Edges are graph edges
	Edges []*Edge `json:"edges" yaml:"edges"`
	
	// Paths are graph paths
	Paths []*Path `json:"paths" yaml:"paths"`
	
	// SubGraph is a subgraph extracted from the knowledge graph
	SubGraph *SubGraph `json:"sub_graph" yaml:"sub_graph"`
}

// SubGraph represents a subgraph extracted from the knowledge graph
type SubGraph struct {
	// Nodes are the nodes in the subgraph
	Nodes []*GraphNode `json:"nodes" yaml:"nodes"`
	
	// Edges are the edges in the subgraph
	Edges []*Edge `json:"edges" yaml:"edges"`
	
	// Root is the root node of the subgraph
	Root string `json:"root" yaml:"root"`
	
	// Depth is the depth of the subgraph extraction
	Depth int `json:"depth" yaml:"depth"`
}

// GraphNode represents a node in the knowledge graph
type GraphNode struct {
	// ID is the node identifier
	ID string `json:"id" yaml:"id"`
	
	// Type is the node type
	Type string `json:"type" yaml:"type"`
	
	// Label is the node label
	Label string `json:"label" yaml:"label"`
	
	// Properties are node properties
	Properties map[string]any `json:"properties" yaml:"properties"`
}

// Edge represents an edge in the knowledge graph
type Edge struct {
	// Source is the source node ID
	Source string `json:"source" yaml:"source"`
	
	// Target is the target node ID
	Target string `json:"target" yaml:"target"`
	
	// Relation is the edge relation type
	Relation string `json:"relation" yaml:"relation"`
	
	// Weight is the edge weight
	Weight float64 `json:"weight" yaml:"weight"`
}

// Path represents a path in the knowledge graph
type Path struct {
	// Nodes are the nodes in the path
	Nodes []string `json:"nodes" yaml:"nodes"`
	
	// Edges are the edges in the path
	Edges []string `json:"edges" yaml:"edges"`
}

// Example represents a few-shot learning example
type Example struct {
	// ID is the example identifier
	ID string `json:"id" yaml:"id"`
	
	// Question is the example question
	Question string `json:"question" yaml:"question"`
	
	// Thoughts are the reasoning steps
	Thoughts []string `json:"thoughts" yaml:"thoughts"`
	
	// Actions are the actions taken
	Actions []string `json:"actions" yaml:"actions"`
	
	// Observations are the results observed
	Observations []string `json:"observations" yaml:"observations"`
	
	// FinalAnswer is the final answer
	FinalAnswer string `json:"final_answer" yaml:"final_answer"`
	
	// Tags categorize the example
	Tags []string `json:"tags" yaml:"tags"`
	
	// Difficulty indicates complexity (1-5)
	Difficulty int `json:"difficulty" yaml:"difficulty"`
}

// ToolDefinition defines a tool for the prompt
type ToolDefinition struct {
	// Name is the tool name
	Name string `json:"name" yaml:"name"`
	
	// Description explains what the tool does
	Description string `json:"description" yaml:"description"`
	
	// Parameters defines the tool parameters
	Parameters map[string]any `json:"parameters" yaml:"parameters"`
	
	// Examples show tool usage
	Examples []string `json:"examples" yaml:"examples"`
}

// OutputFormat defines the expected output format
type OutputFormat struct {
	// ThoughtPrefix is the prefix for thought lines
	ThoughtPrefix string `json:"thought_prefix" yaml:"thought_prefix"`
	
	// ActionPrefix is the prefix for action lines
	ActionPrefix string `json:"action_prefix" yaml:"action_prefix"`
	
	// ObservationPrefix is the prefix for observation lines
	ObservationPrefix string `json:"observation_prefix" yaml:"observation_prefix"`
	
	// FinishAction is the action name for finishing
	FinishAction string `json:"finish_action" yaml:"finish_action"`
	
	// Format string for output
	Format string `json:"format" yaml:"format"`
}

// PromptSection represents a section in the assembled prompt
type PromptSection struct {
	// Type is the section type (system, rag, tools, examples, negative, question)
	Type string `json:"type" yaml:"type"`
	
	// Content is the section content
	Content string `json:"content" yaml:"content"`
	
	// Priority determines ordering
	Priority int `json:"priority" yaml:"priority"`
}

// BuildRequest represents a prompt build request
type BuildRequest struct {
	// Agent is the agent configuration
	Agent any `json:"agent" yaml:"agent"`
	
	// Input is the user input
	Input string `json:"input" yaml:"input"`
	
	// Tools are available tools
	Tools []*ToolDefinition `json:"tools" yaml:"tools"`
	
	// Skills are available skills
	Skills []any `json:"skills" yaml:"skills"`
	
	// Session is the current session
	Session any `json:"session" yaml:"session"`
	
	// RAGContext is pre-retrieved RAG context
	RAGContext *RAGContext `json:"rag_context" yaml:"rag_context"`
	
	// Permission is the user permission level
	Permission *Permission `json:"permission" yaml:"permission"`
	
	// TemplateID is the template to use
	TemplateID string `json:"template_id" yaml:"template_id"`
}

// Permission represents user permissions
type Permission struct {
	// IsAdmin indicates admin privileges
	IsAdmin bool `json:"is_admin" yaml:"is_admin"`
	
	// Permissions is the list of permissions
	Permissions []string `json:"permissions" yaml:"permissions"`
}

// PlanPromptRequest represents a planning prompt request
type PlanPromptRequest struct {
	// Agent is the agent configuration
	Agent any `json:"agent" yaml:"agent"`
	
	// Input is the user input
	Input string `json:"input" yaml:"input"`
	
	// Session is the current session
	Session any `json:"session" yaml:"session"`
	
	// Tools are available tools
	Tools []*ToolDefinition `json:"tools" yaml:"tools"`
}

// ThinkPromptRequest represents a thinking prompt request
type ThinkPromptRequest struct {
	// Agent is the agent configuration
	Agent any `json:"agent" yaml:"agent"`
	
	// Input is the user input
	Input string `json:"input" yaml:"input"`
	
	// Plan is the current execution plan
	Plan any `json:"plan" yaml:"plan"`
	
	// CurrentStep is the current step number
	CurrentStep int `json:"current_step" yaml:"current_step"`
	
	// Session is the current session
	Session any `json:"session" yaml:"session"`
	
	// Tools are available tools
	Tools []*ToolDefinition `json:"tools" yaml:"tools"`
}

// ReflectionPromptRequest represents a reflection prompt request
type ReflectionPromptRequest struct {
	// Agent is the agent configuration
	Agent any `json:"agent" yaml:"agent"`
	
	// Trajectory is the execution trajectory
	Trajectory any `json:"trajectory" yaml:"trajectory"`
	
	// Error is the error that occurred
	Error error `json:"-" yaml:"-"`
	
	// ErrorMessage is the error message
	ErrorMessage string `json:"error_message" yaml:"error_message"`
	
	// Session is the current session
	Session any `json:"session" yaml:"session"`
}

// ReplanPromptRequest represents a replanning prompt request
type ReplanPromptRequest struct {
	// OriginalGoal is the original goal
	OriginalGoal string `json:"original_goal" yaml:"original_goal"`
	
	// OriginalPlan is the original plan
	OriginalPlan any `json:"original_plan" yaml:"original_plan"`
	
	// CurrentStep is the current step
	CurrentStep int `json:"current_step" yaml:"current_step"`
	
	// FailureReason is why replanning is needed
	FailureReason string `json:"failure_reason" yaml:"failure_reason"`
	
	// CompletedSteps are the steps already completed
	CompletedSteps []any `json:"completed_steps" yaml:"completed_steps"`
	
	// Reflections are relevant reflections
	Reflections []*Reflection `json:"reflections" yaml:"reflections"`
}

// Reflection represents a reflection from failed execution
type Reflection struct {
	// FailureReason is why execution failed
	FailureReason string `json:"failure_reason" yaml:"failure_reason"`
	
	// Heuristic is the learned lesson
	Heuristic string `json:"heuristic" yaml:"heuristic"`
	
	// Suggestions are improvement suggestions
	Suggestions []string `json:"suggestions" yaml:"suggestions"`
}

// Trajectory represents the execution trajectory
type Trajectory struct {
	// Steps are the execution steps
	Steps []*TrajectoryStep `json:"steps" yaml:"steps"`
	
	// Success indicates if the execution was successful
	Success bool `json:"success" yaml:"success"`
	
	// FailurePoint is the step where execution failed
	FailurePoint int `json:"failure_point" yaml:"failure_point"`
}

// TrajectoryStep represents a single step in the execution trajectory
type TrajectoryStep struct {
	// Thought is the reasoning at this step
	Thought string `json:"thought" yaml:"thought"`
	
	// Action is the action taken
	Action string `json:"action" yaml:"action"`
	
	// Observation is the result observed
	Observation string `json:"observation" yaml:"observation"`
}

// Plan represents an execution plan
type Plan struct {
	// Goal is the plan goal
	Goal string `json:"goal" yaml:"goal"`
	
	// Steps are the plan steps
	Steps []*PlanStep `json:"steps" yaml:"steps"`
	
	// Success indicates if the plan was successful
	Success bool `json:"success" yaml:"success"`
}

// PlanStep represents a single step in the plan
type PlanStep struct {
	// Description is the step description
	Description string `json:"description" yaml:"description"`
	
	// Tool is the tool to use
	Tool string `json:"tool" yaml:"tool"`
	
	// Dependencies are the step dependencies
	Dependencies []string `json:"dependencies" yaml:"dependencies"`
	
	// Status is the step status
	Status string `json:"status" yaml:"status"`
}

// InjectionStrategy determines how to inject content
type InjectionStrategy int

const (
	// InjectionPrefix injects at the beginning
	InjectionPrefix InjectionStrategy = iota
	// InjectionInfix injects in the middle
	InjectionInfix
	// InjectionSuffix injects at the end
	InjectionSuffix
	// InjectionDynamic chooses position dynamically
	InjectionDynamic
)

// QuestionType represents the type of question
type QuestionType int

const (
	// QuestionTypeFactual asks for factual information
	QuestionTypeFactual QuestionType = iota
	// QuestionTypeProcedural asks for procedures
	QuestionTypeProcedural
	// QuestionTypeAnalytical requires analysis
	QuestionTypeAnalytical
	// QuestionTypeCreative requires creativity
	QuestionTypeCreative
)

// PromptSource identifies the source of a prompt component
type PromptSource int

const (
	// SourceNegativePrompt is from negative prompts
	SourceNegativePrompt PromptSource = iota
	// SourceSystemRole is from system role
	SourceSystemRole
	// SourceSkillPrompt is from skill prompt
	SourceSkillPrompt
	// SourceUserRequest is from user request
	SourceUserRequest
)

// Conflict represents a detected conflict
type Conflict struct {
	// Higher is the higher priority source
	Higher PromptSource `json:"higher" yaml:"higher"`
	
	// Lower is the lower priority source
	Lower PromptSource `json:"lower" yaml:"lower"`
	
	// Description explains the conflict
	Description string `json:"description" yaml:"description"`
	
	// Resolution is how it was resolved
	Resolution string `json:"resolution" yaml:"resolution"`
}

// DefaultOutputFormat returns the default ReAct output format
func DefaultOutputFormat() *OutputFormat {
	return &OutputFormat{
		ThoughtPrefix:    "Thought:",
		ActionPrefix:     "Action:",
		ObservationPrefix: "Observation:",
		FinishAction:     "Finish",
		Format: `Thought: [your reasoning]
Action: [tool_name[arguments] or Finish[answer]]`,
	}
}

// DefaultNegativePromptGroups returns default negative prompt groups
func DefaultNegativePromptGroups() []*NegativePromptGroup {
	return []*NegativePromptGroup{
		{
			ID:          "safety",
			Name:        "安全约束组",
			Description: "保障系统安全的基础约束",
			Enabled:     true,
			Prompts: []*NegativePrompt{
				{ID: "s1", Pattern: "不要泄露敏感信息", Reason: "防止数据泄露", Alternative: "使用脱敏数据", Severity: "critical"},
				{ID: "s2", Pattern: "不要执行危险操作", Reason: "防止系统损坏", Alternative: "请求人工确认", Severity: "critical"},
			},
		},
		{
			ID:          "format",
			Name:        "格式约束组",
			Description: "确保输出格式正确",
			Enabled:     true,
			Prompts: []*NegativePrompt{
				{ID: "f1", Pattern: "不要在 Action 之外输出额外解释", Reason: "保证解析正确", Alternative: "将解释放入 Thought", Severity: "high"},
				{ID: "f2", Pattern: "不要编造不存在的工具", Reason: "避免执行错误", Alternative: "使用可用工具列表中的工具", Severity: "high"},
			},
		},
		{
			ID:          "behavior",
			Name:        "行为约束组",
			Description: "约束 agent 行为",
			Enabled:     true,
			Prompts: []*NegativePrompt{
				{ID: "b1", Pattern: "不要跳过推理步骤", Reason: "确保推理完整", Alternative: "完整展示推理过程", Severity: "medium"},
				{ID: "b2", Pattern: "不要在不确定时猜测", Reason: "避免错误输出", Alternative: "使用工具获取准确信息", Severity: "medium"},
			},
		},
	}
}

// DefaultExamples returns default few-shot examples
func DefaultExamples() []*Example {
	return []*Example{
		{
			ID:       "success-001",
			Question: "北京今天天气如何？",
			Thoughts: []string{
				"我需要查询北京今天的天气信息",
				"我已经获取了天气信息，可以回答用户的问题",
			},
			Actions: []string{
				"weather[北京]",
				"Finish[北京今天天气晴朗，气温25°C]",
			},
			Observations: []string{
				"北京今天天气晴朗，气温25°C，空气质量良好",
			},
			FinalAnswer: "北京今天天气晴朗，气温25°C",
			Tags:        []string{"weather", "success"},
			Difficulty:  1,
		},
		{
			ID:       "recovery-001",
			Question: "苹果公司的股价是多少？",
			Thoughts: []string{
				"我需要查询苹果公司的股价",
				"第一次搜索没有结果，可能关键词不够准确",
				"我应该使用更精确的公司代码",
			},
			Actions: []string{
				"search[苹果股价]",
				"search[AAPL stock price]",
				"Finish[苹果公司(AAPL)当前股价为$178.50]",
			},
			Observations: []string{
				"未找到相关结果",
				"苹果公司(AAPL)当前股价为$178.50",
			},
			FinalAnswer: "苹果公司(AAPL)当前股价为$178.50",
			Tags:        []string{"finance", "recovery"},
			Difficulty:  2,
		},
	}
}

// EvolutionPromptConfig configures evolution paradigm prompts
type EvolutionPromptConfig struct {
	PlanTemplate              string `json:"plan_template" yaml:"plan_template"`
	ReflectionTemplate        string `json:"reflection_template" yaml:"reflection_template"`
	ReplanTemplate            string `json:"replan_template" yaml:"replan_template"`
	MaxReflections            int    `json:"max_reflections" yaml:"max_reflections"`
	MaxSimilarPlans           int    `json:"max_similar_plans" yaml:"max_similar_plans"`
	MaxTrajectorySteps        int    `json:"max_trajectory_steps" yaml:"max_trajectory_steps"`
	EnableReflectionInjection bool   `json:"enable_reflection_injection" yaml:"enable_reflection_injection"`
	EnablePlanInjection       bool   `json:"enable_plan_injection" yaml:"enable_plan_injection"`
}

// DefaultEvolutionConfig returns default evolution prompt config
func DefaultEvolutionConfig() *EvolutionPromptConfig {
	return &EvolutionPromptConfig{
		PlanTemplate:              "templates/plan.tmpl",
		ReflectionTemplate:        "templates/reflection.tmpl",
		ReplanTemplate:            "templates/replan.tmpl",
		MaxReflections:            5,
		MaxSimilarPlans:           3,
		MaxTrajectorySteps:        10,
		EnableReflectionInjection: true,
		EnablePlanInjection:       true,
	}
}

// PromptTemplateConfig configures prompt templates
type PromptTemplateConfig struct {
	SystemTemplate       string        `json:"system_template" yaml:"system_template"`
	OutputFormat         *OutputFormat `json:"output_format" yaml:"output_format"`
	MaxTokens            int           `json:"max_tokens" yaml:"max_tokens"`
	EnableRAG            bool          `json:"enable_rag" yaml:"enable_rag"`
	EnableFewShot        bool          `json:"enable_few_shot" yaml:"enable_few_shot"`
	NegativePromptGroups []string      `json:"negative_prompt_groups" yaml:"negative_prompt_groups"`
}

// DefaultPromptTemplateConfig returns default prompt template config
func DefaultPromptTemplateConfig() *PromptTemplateConfig {
	return &PromptTemplateConfig{
		SystemTemplate:       DefaultSystemTemplate,
		OutputFormat:         DefaultOutputFormat(),
		MaxTokens:            4096,
		EnableRAG:            true,
		EnableFewShot:        true,
		NegativePromptGroups: []string{"safety", "format", "behavior"},
	}
}

// Timestamp for metadata
type Timestamp struct {
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
}
