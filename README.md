<div align="center">

# ⚛️ GoReAct

**The High-Performance, Pattern-Driven ReAct (Reasoning + Acting) Engine for Go.**

[![Go Report Card](https://goreportcard.com/badge/github.com/DotNetAge/goreact)](https://goreportcard.com/report/github.com/DotNetAge/goreact)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Documentation](https://img.shields.io/badge/docs-goreact.rayainfo.cn-6019bd.svg)](https://goreact.rayainfo.cn)

[**Website**](https://goreact.rayainfo.cn) | [**English**](./README.md) | [**中文说明**](./README_zh-CN.md)

</div>

---

## 💡 What is GoReAct?

**GoReAct** is a lightweight, production-ready framework that implements the **ReAct** pattern. Unlike generic LLM wrappers, GoReAct is built for **Programmable Cognition**. It champions the **Pattern-Driven Orchestration** philosophy, enabling developers to build agents that don't just "chat," but execute complex, reliable logic via a **Turing-complete pipeline**.

By integrating with its sibling project [**GoRAG**](https://gorag.rayainfo.cn/), GoReAct achieves industry-leading **Semantic Matching** for tools, memories, and skills, eliminating the brittleness of hardcoded string matching.

## ✨ Evolution Highlights (Phase 4)

- 🧠 **Tri-Modal Memory System**: Mimics human cognition with **Working** (decaying), **Semantic** (GoRAG-powered RAG), and **Muscle** (experience-based SOP) memory modules.
- 🚀 **Universal Logic Pipelines**: Native support for **If/Loop/Break/Return** logic directly within the execution pipeline, enabling reliable long-running tasks.
- 🧬 **Evolutionary Fast-Path**: Automatically compiles successful reasoning traces into **CompiledAction** (Muscle Memory). Future identical tasks execute via a zero-LLM **Fast-Path**, reducing latency by 99%.
- 🧩 **Thinker as Architect**: Advanced **Codeword-driven modes** (`/plan`, `/specs`) transform natural language intentions into structured execution graphs.
- 🛡️ **Sudo HITL Security**: Granular security levels for tools with mandatory Human-In-The-Loop (HITL) authorization for sensitive operations (e.g., file deletion, email).

## 🚀 Quick Start

Build a programmable agent with /plan capabilities in seconds:

```go
package main

import (
    "context"
    "github.com/DotNetAge/goreact/pkg/agent"
    "github.com/DotNetAge/goreact/pkg/engine"
)

func main() {
    // 1. Build an Agent with Pattern-Driven capabilities
    builder := agent.NewBuilder("DevOpsAgent")
    builder.WithModel("gpt-4o")
    builder.WithSystemPrompt("You are a senior DevOps engineer.")
    builder.WithTools(myDeploymentTools...)
    
    myAgent, _ := builder.Build()

    // 2. Execute a complex task with planning
    // The agent will decompose the task and execute it via the logic pipeline.
    ctx := context.Background()
    result, _ := myAgent.Chat(ctx, "session-1", "/plan Deploy the service and if tests fail, rollback to v1.0.0.")
    
    println(result)
}
```

## 🏗️ The 3-State Architecture

GoReAct skills evolve through three distinct states:

1.  **Source State**: Human-readable Markdown (`SKILL.md`) interpreted via the **Master-Sub** ReAct loop.
2.  **Compiled State**: The **Compiler** distills successful traces into a structured **CompiledAction** with execution fingerprints.
3.  **Execution State**: The **Adaptive Runner** executes the "Fast-Path" directly, using the **Observer** as a judge to verify fingerprints.

## 🏛️ System Overview

```text
       [ User Intent ] 
              │
      ┌───────▼────────┐      ┌─────────────────────────┐
      │    Thinker     │◄─────┤   MemoryBank (3-Modal)  │
      │ (Task Compiler)│      │  (Powered by GoRAG)     │
      └───────┬────────┘      └─────────────────────────┘
              │ Logical Plan (If/Loop/Sequence)
      ┌───────▼────────┐      ┌─────────────────────────┐
      │    Pipeline    │◄─────┤   Security Hook (HITL)  │
      │   (Execution)  │      └─────────────────────────┘
      └───────┬────────┘
              │ Action Results
      ┌───────▼────────┐
      │     Runner     │──────► [ CompiledAction Cache ]
      │  (Evolution)   │
      └────────────────┘
```

## 📖 Documentation & Resources

- [**Official Documentation**](https://goreact.rayainfo.cn) - Comprehensive guides and API reference.
- [Detailed Features](./FEATURES.md) - Explore the core innovations.
- [Architecture Design](./ARCHITECTURE.md) - Deep dive into the engine.
- [Skill Specification](./pkg/skill/README.md) - Build self-evolving skills.

## 🛠️ Built-in Capabilities
- **Models**: OpenAI, Anthropic, Ollama, DashScope (Qwen), DeepSeek.
- **Tools**: `Bash`, `FileIO`, `HTTP`, `Calculator`, `DateTime`, `Search`.
- **Patterns**: `Master-Sub`, `Evolution`, `Chain-of-Thought`.

## 🤝 Contributing

We welcome contributions! Please check out our [Contributing Guidelines](CONTRIBUTING.md).

**GoReAct** is released under the [MIT License](LICENSE).
