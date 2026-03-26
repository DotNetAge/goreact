package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/DotNetAge/gochat/pkg/client/base"
	"github.com/DotNetAge/gochat/pkg/client/openai"
	"github.com/DotNetAge/goreact/pkg/actor"
	"github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/engine"
	"github.com/DotNetAge/goreact/pkg/observer"
	"github.com/DotNetAge/goreact/pkg/terminator"
	"github.com/DotNetAge/goreact/pkg/thinker"
	"github.com/DotNetAge/goreact/pkg/tools"
	"github.com/DotNetAge/goreact/pkg/tools/builtin"
)

func main() {
	fmt.Println("🚀 Starting GoReAct Cloud Agent (Qwen-3.5-Flash via DashScope)...")

	// 1. Get API Key from environment
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		log.Fatal("Environment variable DASHSCOPE_API_KEY is not set")
	}

	// 2. Initialize OpenAI-Compatible Client for DashScope
	config := openai.Config{
		Config: base.Config{
			APIKey:  apiKey,
			BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/",
			Model:   "qwen3.5-flash",
			Timeout: 7200 * time.Second, // 绕过 http.Client 级别的拦截，交给 context

		},
	}
	client, err := openai.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// 3. Prepare Tools
	toolMgr := tools.NewSimpleManager()
	toolMgr.Register(builtin.NewCalculator(), builtin.NewDateTime())
	fmt.Println("🛠️  Equipped Tools: [Calculator, DateTime]")

	// 4. Build Reactor
	agent := engine.NewReactor(
		engine.WithThinker(thinker.Default(client,
			thinker.WithModel("qwen3.5-flash"),
			thinker.WithToolManager(toolMgr),
		)),
		engine.WithActor(actor.Default(
			actor.WithToolManager(toolMgr),
		)),
		engine.WithObserver(observer.Default()),
		engine.WithTerminator(terminator.Default()),
	)

	// 5. Context with reasonable cloud timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 6. User Task
	userInput := "如果今天是2026年3月17日，请先计算100天后是几号，然后把那个日期再往后加15天。"
	fmt.Printf("\n👤 User: %s\n\n🧠 Thinking Log:\n", userInput)

	// 7. Run with real-time Thought Stream
	reactCtx, err := agent.Run(ctx, "cloud-session-001", userInput,
		core.WithThoughtStream(func(chunk string) {
			fmt.Print(chunk)
		}),
	)

	if err != nil {
		log.Fatalf("\n❌ Agent Error: %v", err)
	}

	// 8. Final Results
	fmt.Printf("\n\n✅ Final Answer: %s\n", reactCtx.FinalResult)
	fmt.Printf("📊 Token Audit: Prompt: %d | Completion: %d | Total: %d\n",
		reactCtx.TotalTokens.PromptTokens,
		reactCtx.TotalTokens.CompletionTokens,
		reactCtx.TotalTokens.TotalTokens)
	fmt.Printf("⏱️  Execution Time: %v\n", time.Since(reactCtx.StartTime))
}
