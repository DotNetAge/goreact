package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/DotNetAge/gochat/pkg/client/base"
	"github.com/DotNetAge/gochat/pkg/client/ollama"
	"github.com/ray/goreact/pkg/actor"
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/engine"
	"github.com/ray/goreact/pkg/observer"
	"github.com/ray/goreact/pkg/terminator"
	"github.com/ray/goreact/pkg/thinker"
	"github.com/ray/goreact/pkg/tools"
	"github.com/ray/goreact/pkg/tools/builtin"
)

func main() {
	fmt.Println("🚀 Starting GoReAct Local Agent (Ollama + Qwen3.5:2b)...")

	// 1. Initialize the GoChat Ollama Client
	config := ollama.Config{
		Config: base.Config{
			Model:   "qwen3.5:2b",
			Timeout: 36000 * time.Second, // Override gochat's default 60s timeout for local models

			BaseURL: "http://localhost:11434",
		},
	}
	client, err := ollama.New(config)
	if err != nil {
		log.Fatalf("Failed to connect to Ollama: %v", err)
	}

	// 2. Prepare the Tools Manager
	toolMgr := tools.NewSimpleManager()
	toolMgr.Register(builtin.NewCalculator())
	toolMgr.Register(builtin.NewDateTime())
	
	fmt.Println("🛠️  Equipped Tools: [Calculator, DateTime]")

	// 3. Assemble the Reactor
	agent := engine.NewReactor(
		engine.WithThinker(thinker.Default(client, 
			thinker.WithModel("qwen3.5:2b"),
			thinker.WithToolManager(toolMgr),
		)),
		engine.WithActor(actor.Default(
			actor.WithToolManager(toolMgr),
		)),
		engine.WithObserver(observer.Default()),
		engine.WithTerminator(terminator.Default()),
	)

	// 4. Set a VERY EXTREME timeout for slow local model inference & cold start
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second) // 10 mins
	defer cancel()

	// 5. User Input
	userInput := "如果今天是2026年3月17日，那么100天后是几号？请先计算日期，再把那个日期加15天。"
	fmt.Printf("\n👤 User: %s\n\n🧠 Thinking Log (Models may take several minutes to load on slow PCs)... \n", userInput)

	// 6. Run the Engine with Thought Stream injected into Context setup
	reactCtx, err := agent.Run(ctx, "session-001", userInput, 
		core.WithThoughtStream(func(chunk string) {
			fmt.Print(chunk) // Dynamic typing effect
		}),
	)

	if err != nil {
		log.Fatalf("\n❌ Agent Error: %v", err)
	}

	// 7. Output Final Results & Audit
	fmt.Printf("\n\n✅ Final Answer: %s\n", reactCtx.FinalResult)
	fmt.Printf("📊 Token Audit: Prompt: %d | Completion: %d | Total: %d\n", 
		reactCtx.TotalTokens.PromptTokens, 
		reactCtx.TotalTokens.CompletionTokens, 
		reactCtx.TotalTokens.TotalTokens)
	fmt.Printf("⏱️  Time Spent: %v\n", time.Since(reactCtx.StartTime))
}
