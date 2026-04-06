// Package main demonstrates the basic usage of the goreact framework.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/DotNetAge/goreact/pkg/agent"
	"github.com/DotNetAge/goreact/pkg/common"
	"github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/memory"
	"github.com/DotNetAge/goreact/pkg/reactor"
	"github.com/DotNetAge/goreact/pkg/tool"
)

// SimpleAgent is a basic agent implementation
type SimpleAgent struct {
	*agent.BaseAgent
	reactor *reactor.Reactor
	memory  *memory.Memory
}

// NewSimpleAgent creates a new simple agent
func NewSimpleAgent(name string) *SimpleAgent {
	config := &agent.Config{
		Name:       name,
		Domain:     "general",
		Model:      "gpt-4",
		MaxSteps:   10,
		MaxRetries: 3,
		Timeout:    30 * time.Second,
	}
	
	return &SimpleAgent{
		BaseAgent: agent.NewBaseAgent(config),
		memory:    memory.NewMemory(),
	}
}

// Ask executes a question
func (a *SimpleAgent) Ask(ctx context.Context, question string, files ...string) (*agent.Result, error) {
	// Create reactor if not exists
	if a.reactor == nil {
		a.reactor = reactor.NewReactor()
		a.reactor.WithPlanner(reactor.NewBasePlanner())
		a.reactor.WithThinker(reactor.NewBaseThinker(nil, nil)) // nil LLM uses fallback
		a.reactor.WithActor(reactor.NewBaseActor(nil))
		a.reactor.WithObserver(reactor.NewBaseObserver(nil))
	}
	
	// Execute the reactor
	result, err := a.reactor.Execute(ctx, question,
		reactor.WithMaxSteps(10),
		reactor.WithMaxRetries(3),
	)
	if err != nil {
		return nil, err
	}
	
	// Convert to agent result
	agentResult := &agent.Result{
		Answer:     result.Answer,
		Status:     result.Status,
		TokenUsage: result.TokenUsage,
		Duration:   result.Duration,
	}
	
	if result.State != nil {
		agentResult.SessionName = result.State.SessionName
		agentResult.Trajectory = result.State.Trajectory
	}
	
	if result.PendingQuestion != nil {
		agentResult.PendingQuestion = &agent.PendingQuestion{
			ID:            result.PendingQuestion.Name,
			Type:          result.PendingQuestion.Type,
			Question:      result.PendingQuestion.Question,
			Options:       result.PendingQuestion.Options,
			DefaultAnswer: result.PendingQuestion.DefaultAnswer,
		}
	}
	
	return agentResult, nil
}

// Resume resumes a paused session
func (a *SimpleAgent) Resume(ctx context.Context, sessionName string, answer string) (*agent.Result, error) {
	if a.reactor == nil {
		return nil, fmt.Errorf("no active session to resume")
	}
	
	state := a.reactor.State()
	if state == nil {
		return nil, fmt.Errorf("no state to resume")
	}
	
	result, err := a.reactor.Resume(ctx, state, answer)
	if err != nil {
		return nil, err
	}
	
	return &agent.Result{
		Answer:   result.Answer,
		Status:   result.Status,
		Duration: result.Duration,
	}, nil
}

// AskStream executes with streaming
func (a *SimpleAgent) AskStream(ctx context.Context, question string, files ...string) (<-chan any, error) {
	if a.reactor == nil {
		a.reactor = reactor.NewReactor()
	}
	
	return a.reactor.ExecuteStream(ctx, question)
}

func main() {
	ctx := context.Background()
	
	// Register built-in tools
	if err := tool.RegisterBuiltins(); err != nil {
		log.Fatalf("Failed to register tools: %v", err)
	}
	
	// Create an agent
	agent := NewSimpleAgent("assistant")
	
	// Example 1: Simple question
	fmt.Println("=== Example 1: Simple Question ===")
	result, err := agent.Ask(ctx, "What is the weather today?")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Answer: %s\n", result.Answer)
		fmt.Printf("Status: %s\n", result.Status)
		fmt.Printf("Duration: %v\n", result.Duration)
	}
	
	// Example 2: Task execution
	fmt.Println("\n=== Example 2: Task Execution ===")
	result, err = agent.Ask(ctx, "Read the file /etc/hosts and tell me its contents")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Answer: %s\n", result.Answer)
		fmt.Printf("Status: %s\n", result.Status)
	}
	
	// Example 3: Working with state
	fmt.Println("\n=== Example 3: Working with State ===")
	state := core.NewState("test-session", "Test input", 5, 3)
	state.AddThought(core.NewThought("Analyzing input", "Need to determine intent", "act", 0.8))
	state.AddAction(core.NewAction(common.ActionTypeToolCall, "read", map[string]any{"file_path": "/tmp/test.txt"}))
	state.AddObservation(core.NewObservation("File content", "read", true))
	
	fmt.Printf("Session: %s\n", state.SessionName)
	fmt.Printf("Current Step: %d/%d\n", state.CurrentStep, state.MaxSteps)
	fmt.Printf("Thoughts: %d\n", len(state.Thoughts))
	fmt.Printf("Actions: %d\n", len(state.Actions))
	fmt.Printf("Observations: %d\n", len(state.Observations))
	
	// Example 4: Memory operations
	fmt.Println("\n=== Example 4: Memory Operations ===")
	mem := memory.NewMemory()
	
	// Access different memory types
	sessions := mem.Sessions()
	shortTerms := mem.ShortTerms()
	longTerms := mem.LongTerms()
	
	fmt.Printf("Session Accessor: %T\n", sessions)
	fmt.Printf("ShortTerm Accessor: %T\n", shortTerms)
	fmt.Printf("LongTerm Accessor: %T\n", longTerms)
	
	fmt.Println("\n=== Framework Demo Completed ===")
}
