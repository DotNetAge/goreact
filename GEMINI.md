# GoReAct

## Project Overview

GoReAct is a high-performance, extensible ReAct (Reasoning + Acting) engine framework built with Go. It enables the creation of robust AI systems capable of advanced reasoning and tool execution.

### Key Architecture & Features
- **Clean Architecture:** Separation of concerns using core abstractions: **Thinker**, **Actor**, **Observer**, and **LoopController**.
- **Extensible Tool System:** Easily register custom and built-in tools (e.g., HTTP, DateTime, Calculator).
- **Prompt Toolkit:** A complete toolbox for Prompt construction, featuring fluent APIs, multiple tool formatters (JSON Schema, Markdown), smart compression strategies, and accurate token counters.
- **Middleware System:** A web-inspired Thinker Middleware System for enhancing the "Think" phase of the ReAct loop.
- **LLM Integration:** Built-in support for real LLM providers including Ollama, OpenAI, and Anthropic, alongside a Mock LLM for local testing.
- **Advanced Capabilities:** Intelligent in-memory caching, Task Decomposition, RAG (Retrieval-Augmented Generation) extension points, Multi-agent Coordination, and a complete SKILLS system.

## Directory Structure
- `pkg/`: Core framework packages (`actor`, `agent`, `engine`, `llm`, `loopctrl`, `observer`, `prompt`, `tool`, etc.).
- `examples/`: Example implementations demonstrating usage of the framework, including toolkit demonstrations, LLM integrations, and metric dashboards.
- `docs/`: Extensive project documentation, design records, and integration guides.

## Building and Running

Since this is a standard Go module (`go 1.25.1`), use the standard Go toolchain for building and testing:

```bash
# Download dependencies
go mod download

# Run all tests (Note: ensure files are formatted via go fmt first)
go test ./...

# Run a specific example
go run examples/simple/main.go
```

## Development Conventions

- **Go Standards:** Strictly adhere to standard Go formatting and idioms. Ensure you use `go fmt` before submitting or pushing code, as redundant newlines or formatting issues will cause build failures in this project environment.
- **Testing:** New features or bug fixes in `pkg/` should be accompanied by comprehensive unit tests in corresponding `*_test.go` files.
- **Extensibility:** When adding new LLM providers, place them in `pkg/llm/`. When introducing new built-in tools, define them in `pkg/tool/builtin/`.
- **Interfaces:** Prefer satisfying established interfaces when replacing or extending components like Cache, Prompts, or Observability hooks.
