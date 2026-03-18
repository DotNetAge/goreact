<div align="center">

# ⚛️ GoReAct

**A High-Performance, Extensible ReAct (Reasoning + Acting) Framework for Go.**

[![Go Report Card](https://goreportcard.com/badge/github.com/ray/goreact)](https://goreportcard.com/report/github.com/ray/goreact)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

[**English**](./README.md) | [**中文说明**](./README_zh-CN.md)

</div>

---

## 💡 What is GoReAct?

**GoReAct** is a lightweight, production-ready framework that implements the **ReAct (Reasoning + Acting)** pattern. It empowers developers to build autonomous AI agents that can iteratively reason about complex tasks, formulate plans, and execute actions using a robust toolkit.

Instead of relying on fragile, complex distributed P2P agent networks, GoReAct champions the **Agent-as-a-Tool (AAAT)** philosophy. Everything—from a simple calculator to an entire sub-agent—is just a tool within a unified, fractal-like architecture.

## ✨ Key Features

- 🏗️ **Clean Architecture**: Strict separation of concerns via the **Thinker**, **Actor**, **Observer**, and **Terminator** pipeline.
- 🧰 **Agent-as-a-Tool (AAAT)**: Seamlessly wrap entire autonomous sub-agents as tools for a supervisor agent. Infinite horizontal scaling without the complexity.
- 🔌 **Middleware Ecosystem**: Web-inspired middleware for the *Think* phase (Logging, Caching, Rate Limiting, RAG injection).
- 🧠 **Prompt Toolkit**: Advanced context management, smart token counting (Chinese/English/Mixed), and dynamic conversation compression.
- ⚡ **High Performance & Safety**: Designed for concurrency. Features strict timeout contexts, panic-recovery, and graceful degradation during LLM outages.
- 🌐 **Provider Agnostic**: Bring your own LLM. Natively integrates with `gochat` for OpenAI, Anthropic, Qwen, and Ollama.

## 🚀 Quick Start

### Installation

```bash
go get github.com/ray/goreact
```

### The 1-Minute Example

Build an agent that can calculate dates and perform math, using a real LLM (e.g., Qwen/OpenAI):

```go
package main

import (
	"context"
	"fmt"
	"github.com/ray/goreact/pkg/engine"
	"github.com/ray/goreact/pkg/thinker"
	"github.com/ray/goreact/pkg/actor"
	"github.com/ray/goreact/pkg/tools"
	"github.com/ray/goreact/pkg/tools/builtin"
	// Assume you have an initialized gochat client
)

func main() {
	// 1. Equip Tools
	toolMgr := tools.NewSimpleManager()
	toolMgr.Register(
		builtin.NewCalculator(),
		builtin.NewDateTime()
	)

	// 2. Build the ReAct Engine
	reactor := engine.NewReactor(
		engine.WithThinker(thinker.Default(client, 
			thinker.WithModel("gpt-4o"),
			thinker.WithToolManager(toolMgr),
		)),
		engine.WithActor(actor.Default(actor.WithToolManager(toolMgr))),
	)

	// 3. Run the Agent
	ctx := context.Background()
	result, _ := reactor.Run(ctx, "session-1", "If today is 2026-03-18, what date is 100 days from now? Calculate it.")

	fmt.Println("Final Answer:", result.FinalResult)
}
```

## 🏗️ Architecture Design

The core of GoReAct is an iterative State Machine composed of four primary abstractions:

1. **Thinker**: Consumes the context and current observation, reasoning about the next step via LLM, and selecting a Tool.
2. **Actor**: Safely executes the selected Tool in an isolated sandbox.
3. **Observer**: Validates the tool's execution result, formatting it into an `Observation` to feed back into the prompt.
4. **Terminator**: Evaluates if the `Final Answer` is reached or if the loop should terminate (e.g., max iterations, timeout).

```text
Task Input
    │
 ┌──▼─────────────────────────────┐
 │ Thinker (Reason & Select Tool) │◄──┐
 └─┬──────────────────────────────┘   │
   │ Thought + Action                 │
 ┌─▼──────────────────────────────┐   │
 │ Actor (Execute Tool Safely)    │   │
 └─┬──────────────────────────────┘   │
   │ Raw Result / Error               │
 ┌─▼──────────────────────────────┐   │
 │ Observer (Format Observation)  │   │
 └─┬──────────────────────────────┘   │
   │ State & Context Updates          │
 ┌─▼──────────────────────────────┐   │
 │ Terminator (Check Completion)  ├───┘
 └────────────────────────────────┘
```

## 📖 Documentation & Guides

Dive deeper into GoReAct's capabilities:

- [Architecture Deep Dive](./ARCHITECTURE.md) - Learn the inner workings and design decisions.
- [RAG Integration Guide](./docs/RAG_INTEGRATION_GUIDE.md) - How to build an Agentic RAG system.
- [Tool Development Guide](./pkg/tools/builtin/USAGE_GUIDE.md) - Learn how to write your own custom tools.

## 🛠️ Built-in Tool Library

GoReAct comes with a suite of battle-tested tools ready for use:
- **System**: `Bash`, `Grep`, `Read`, `Write`, `Edit`, `Glob`, `LS`
- **Utilities**: `Calculator`, `DateTime`, `Email`

## 🤝 Contributing

We welcome contributions from the community! Whether it's fixing bugs, improving documentation, or proposing new features. 

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
