package examples_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/DotNetAge/gochat/pkg/client/base"
	"github.com/DotNetAge/gochat/pkg/client/ollama"
	"github.com/DotNetAge/gochat/pkg/client/openai"
	"github.com/ray/goreact/pkg/actor"
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/engine"
	"github.com/ray/goreact/pkg/observer"
	"github.com/ray/goreact/pkg/terminator"
	"github.com/ray/goreact/pkg/thinker"
	"github.com/ray/goreact/pkg/tools"
	"github.com/ray/goreact/pkg/tools/builtin"
)

// setupTools returns a ready-to-use Tool Manager with our demo tools.
func setupTools() tools.Manager {
	toolMgr := tools.NewSimpleManager()
	// NOTE: In a real debug session, you can put breakpoints inside
	// builtin.DateTime.Execute or builtin.Calculator.Execute to see the Actor hitting them!
	toolMgr.Register(builtin.NewCalculator())
	toolMgr.Register(builtin.NewDateTime())
	return toolMgr
}

// TestOllamaAgent runs the agent using your local Ollama Qwen3.5:2b model.
// Tip: Put a breakpoint inside `pkg/thinker/default_thinker.go` at the `Think` method.
func TestOllamaAgent(t *testing.T) {
	// 1. Client setup with infinite underlying timeout
	config := ollama.Config{
		Config: base.Config{
			Model:   "qwen3.5:.8b",
			BaseURL: "http://localhost:11434",
			Timeout: 7200 * time.Second, // Bypass gochat's 60s hard limit
		},
	}
	client, err := ollama.New(config)
	if err != nil {
		t.Fatalf("Failed to create ollama client: %v", err)
	}

	toolMgr := setupTools()

	// 2. Reactor setup
	agent := engine.NewReactor(
		engine.WithThinker(thinker.Default(client,
			thinker.WithModel("qwen3.5:0.8b"),
			thinker.WithToolManager(toolMgr),
		)),
		engine.WithActor(actor.Default(actor.WithToolManager(toolMgr))),
		engine.WithObserver(observer.Default()),
		engine.WithTerminator(terminator.Default()),
	)

	// 3. Execution Context
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	userInput := "如果今天是2026年3月17日，请先计算100天后是几号，然后把那个日期再往后加15天。"
	fmt.Printf("\n[Ollama Test] User: %s\n\n", userInput)

	// 4. Run the Agent
	reactCtx, err := agent.Run(ctx, "test-ollama", userInput, core.WithThoughtStream(func(chunk string) {
		// This prints the LLM's thought process as it generates it
		fmt.Print(chunk)
	}))

	if err != nil {
		t.Fatalf("\n❌ Agent Error: %v", err)
	}

	fmt.Printf("\n\n✅ Final Answer: %s\n", reactCtx.FinalResult)
	fmt.Printf("📊 Tokens: Prompt: %d | Completion: %d | Total: %d\n",
		reactCtx.TotalTokens.PromptTokens, reactCtx.TotalTokens.CompletionTokens, reactCtx.TotalTokens.TotalTokens)
}

// TestQwenCloudAgent runs the agent using DashScope (Aliyun) Qwen-3.5-Flash model.
func TestQwenCloudAgent(t *testing.T) {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		t.Skip("DASHSCOPE_API_KEY is not set. Skipping cloud test. To run this, set the env variable.")
	}

	// 1. Client setup with infinite underlying timeout
	config := openai.Config{
		Config: base.Config{
			APIKey:  apiKey,
			BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
			Model:   "qwen3.5-flash",
			Timeout: 7200 * time.Second, // Bypass gochat's 30s hard limit
		},
	}
	client, err := openai.New(config)
	if err != nil {
		t.Fatalf("Failed to create openai client: %v", err)
	}

	toolMgr := setupTools()

	// 2. Reactor setup
	agent := engine.NewReactor(
		engine.WithThinker(thinker.Default(client,
			thinker.WithModel("qwen3.5-flash"),
			thinker.WithToolManager(toolMgr),
		)),
		engine.WithActor(actor.Default(actor.WithToolManager(toolMgr))),
		engine.WithObserver(observer.Default()),
		engine.WithTerminator(terminator.Default()),
	)

	// 3. Execution Context
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	userInput := "如果今天是2026年3月17日，请先计算100天后是几号，然后把那个日期再往后加15天。"
	fmt.Printf("\n[Cloud Test] User: %s\n\n", userInput)

	// 4. Run the Agent
	reactCtx, err := agent.Run(ctx, "test-cloud", userInput, core.WithThoughtStream(func(chunk string) {
		fmt.Print(chunk)
	}))

	if err != nil {
		t.Fatalf("\n❌ Agent Error: %v", err)
	}

	fmt.Printf("\n\n✅ Final Answer: %s\n", reactCtx.FinalResult)
	fmt.Printf("📊 Tokens: Prompt: %d | Completion: %d | Total: %d\n",
		reactCtx.TotalTokens.PromptTokens, reactCtx.TotalTokens.CompletionTokens, reactCtx.TotalTokens.TotalTokens)
}
