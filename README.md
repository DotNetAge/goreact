<div align="center">

# ⚛️ GoReAct

**A High-Performance, Programmable ReAct (Reasoning + Acting) Framework for Go.**

[![Go Report Card](https://goreportcard.com/badge/github.com/ray/goreact)](https://goreportcard.com/report/github.com/ray/goreact)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

[**English**](./README.md) | [**中文说明**](./README_zh-CN.md)

</div>

---

## 💡 What is GoReAct?

**GoReAct** is a lightweight, production-ready framework that implements the **ReAct** pattern. GoReAct champions the **Agent-as-a-Tool (AAAT)** philosophy and features a **Turing-complete logic pipeline**, enabling agents to reason, plan, and execute tasks with human-like flexibility and programmable precision.

## ✨ Evolution Highlights (Phase 4)

- 🧠 **Tri-Modal Memory System**: Mimics human cognition with **Working** (decaying), **Semantic** (RAG), and **Muscle** (experience-based SOP) memory modules.
- 🚀 **Universal Logic Pipelines**: Native support for **If/Loop/Break/Return** logic directly within the execution pipeline, upstreamed to `gochat`.
- 🧩 **Thinker as Architect**: Advanced **Codeword-driven modes** (`/plan`, `/specs`) and a task compiler that transforms natural language plans into executable logic flows.
- 🧰 **Prompt Protocol**: Fluent API for protocol-grade prompt construction, featuring smart token counting and context compression (Sliding Window).
- 🛡️ **Sudo HITL Security**: Granular security levels for tools with mandatory Human-In-The-Loop (HITL) authorization for sensitive operations.

## 🚀 Quick Start

Build a programmable agent with /plan capabilities:

```go
// 1. Initialize Reactor with Memory and Tools
reactor := engine.NewReactor(
    engine.WithThinker(thinker.Default(client, 
        thinker.WithMemoryBank(myMemory),
        thinker.WithToolManager(toolMgr),
    )),
    engine.WithActor(actor.Default(actor.WithToolManager(toolMgr))),
)

// 2. Run with /plan to decompose tasks
result, _ := reactor.Run(ctx, "sess-1", "/plan Analyze the CSV and if errors found, fix them using the python script.")
```

## 🏗️ Architecture

```text
       [ User Intent ] 
              │
      ┌───────▼────────┐      ┌─────────────────────────┐
      │    Thinker     │◄─────┤   MemoryBank (3-Modal)  │
      │ (Task Compiler)│      └─────────────────────────┘
      └───────┬────────┘
              │ Logical Plan (If/Loop/Sequence)
      ┌───────▼────────┐      ┌─────────────────────────┐
      │    Pipeline    │◄─────┤   Security Hook (HITL)  │
      │   (Execution)  │      └─────────────────────────┘
      └───────┬────────┘
              │ Action Results
      ┌───────▼────────┐
      │   Crystallizer │──────► [ Muscle Memory / SOP ]
      │  (Evolution)   │
      └────────────────┘
```

## 📖 Deep Dives

- [Detailed Features](./FEATURES.md) - Explore the core innovations.
- [Architecture Guide](./ARCHITECTURE.md) - Deep dive into the programmable pipeline.
- [Skill Specification](./pkg/skill/README.md) - Build self-evolving skills.

## 🛠️ Tool Library
- **System**: `Bash`, `Grep`, `Read`, `Write`, `Edit`, `Glob`, `LS`
- **Utility**: `Calculator`, `DateTime`, `Email`, `Echo`

## 🤝 Contributing
Contributions are welcome! MIT Licensed.
