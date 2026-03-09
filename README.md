# GoReAct

A high-performance, extensible ReAct (Reasoning + Acting) engine framework built with Go.

## Overview

GoReAct is a lightweight framework that implements the ReAct pattern - a powerful approach for building AI systems that can reason about tasks and take actions using tools.

**Current Status**: Phase 3 - Advanced Features (Completed)

## Phase 3 Features (NEW)
- **Thinker Middleware System**: Web-inspired middleware pattern for enhancing the Think phase
- **Prompt Toolkit**: Complete toolbox for Prompt construction and context management
  - FluentPromptBuilder with fluent API
  - Multiple tool formatters (JSON Schema, Markdown, Compact)
  - Accurate token counters (supports Chinese/English/Mixed)
  - Smart compression strategies (Priority, Sliding Window, Hybrid)
  - Debugging and tracking tools
- **Multi-agent Coordination**: Support for multiple agents with different skills
- **SKILLS System**: Skill evaluation and management
- **Task Decomposition**: Break down complex tasks into sub-tasks
- **RAG Extension Point**: Extension point for integrating Retrieval-Augmented Generation
- **Monitoring & Metrics**: Comprehensive metrics collection and monitoring
- **Additional LLM Providers**: Support for OpenAI and Anthropic

## Phase 2.1 Features
- **Error Handling & Retry Mechanism**: Automatic retries for LLM and tool execution failures
- **Graceful Degradation**: Fallback to simplified mode when LLM is unavailable
- **Enhanced Error Messages**: More detailed and user-friendly error information
- **Cache Error Recovery**: Leverage caching to handle LLM outages

## Features

### Core Features
- **Clean Architecture**: Separation of concerns with Thinker, Actor, Observer, and LoopController
- **Extensible Tool System**: Easy to register and use custom tools
- **Flexible Design**: All core components can be customized via interfaces
- **Type-Safe**: Leverages Go's strong type system
- **Simple API**: Minimal learning curve with intuitive API design

### Phase 2 Features (NEW)
- **Real LLM Integration**: Support for external LLM providers (Ollama, and easy to add more)
- **Intelligent Caching**: In-memory caching with TTL support for massive performance gains
- **Enhanced Tools**: HTTP requests, DateTime operations, Calculator, and Echo
- **External LLM Design**: Framework provides interfaces, users bring their own LLM clients

## Quick Start

### Prerequisites

For real LLM integration, install Ollama:

```bash
# Install Ollama (macOS/Linux)
curl -fsSL https://ollama.com/install.sh | sh

# Start Ollama
ollama serve

# Pull a model
ollama pull qwen3:0.6b
```

### Installation

```bash
go get github.com/ray/goreact
```

### Basic Usage with Mock LLM

```go
package main

import (
    "fmt"
    "github.com/ray/goreact/pkg/engine"
    "github.com/ray/goreact/pkg/tool/builtin"
)

func main() {
    // Create engine (uses mock LLM by default)
    eng := engine.New(
        engine.WithMaxIterations(5),
    )

    // Register tools
    eng.RegisterTools(
        builtin.NewCalculator(),
        builtin.NewEcho(),
    )

    // Execute task
    result := eng.Execute("Calculate 15 * 23 + 7", nil)

    fmt.Printf("Success: %v\n", result.Success)
    fmt.Printf("Output: %s\n", result.Output)
}
```

### Usage with Real LLM (Ollama)

```go
package main

import (
    "fmt"
    "time"

    "github.com/ray/goreact/pkg/engine"
    "github.com/ray/goreact/pkg/llm/ollama"
    "github.com/ray/goreact/pkg/cache"
    "github.com/ray/goreact/pkg/tool/builtin"
)

func main() {
    // Create Ollama client
    llmClient := ollama.NewOllamaClient(
        ollama.WithModel("qwen3:0.6b"),
        ollama.WithTemperature(0.7),
    )

    // Create cache for performance
    memCache := cache.NewMemoryCache(
        cache.WithMaxSize(100),
        cache.WithDefaultTTL(1 * time.Hour),
    )

    // Create engine with real LLM and cache
    eng := engine.New(
        engine.WithLLMClient(llmClient),
        engine.WithCache(memCache),
        engine.WithMaxIterations(10),
    )

    // Register tools
    eng.RegisterTools(
        builtin.NewCalculator(),
        builtin.NewHTTP(),
        builtin.NewDateTime(),
    )

    // Execute task
    result := eng.Execute("What's 25 + 17?", nil)

    fmt.Printf("Success: %v\n", result.Success)
    fmt.Printf("Output: %s\n", result.Output)
}
```

## Architecture

### Core Components

- **Engine**: Orchestrates the ReAct loop
- **Thinker**: Analyzes tasks and generates reasoning (uses LLM)
- **Actor**: Executes actions by calling tools
- **Observer**: Analyzes execution results and provides feedback
- **LoopController**: Controls the iteration loop
- **Context**: Manages execution context across the loop

### Data Flow

```
Task Input
    ↓
Thinker (Think) → Thought
    ↓
Actor (Act) → ExecutionResult
    ↓
Observer (Observe) → Feedback
    ↓
LoopController (Control) → Continue/Stop
    ↓
Result Output
```

## Built-in Tools

### Calculator

Performs basic arithmetic operations.

```go
eng.RegisterTool(builtin.NewCalculator())
```

Supported operations: `add`, `subtract`, `multiply`, `divide`

Parameters: `{operation: "add", a: 10, b: 5}`

### Echo

Echoes back the input message (useful for testing).

```go
eng.RegisterTool(builtin.NewEcho())
```

Parameters: `{message: "Hello, World!"}`

### HTTP (NEW)

Makes HTTP requests.

```go
eng.RegisterTool(builtin.NewHTTP())
```

Parameters: `{method: "GET", url: "https://api.example.com", body: "...", headers: {...}}`

### DateTime (NEW)

Date and time operations.

```go
eng.RegisterTool(builtin.NewDateTime())
```

Operations:
- `now`: Get current time
- `format`: Format a time string
- `parse`: Parse a time string

Parameters: `{operation: "now", format: "2006-01-02 15:04:05"}`

## Creating Custom Tools

Implement the `Tool` interface:

```go
type MyTool struct{}

func (t *MyTool) Name() string {
    return "my_tool"
}

func (t *MyTool) Description() string {
    return "Description of what my tool does"
}

func (t *MyTool) Execute(params map[string]interface{}) (interface{}, error) {
    // Your tool logic here
    return result, nil
}

// Register it
eng.RegisterTool(&MyTool{})
```

## Examples

### Simple Echo Example (Mock LLM)

```bash
go run examples/simple/main.go
```

### Calculator Example (Mock LLM)

```bash
go run examples/calculator/main.go
```

### Ollama Integration Example (Real LLM)

```bash
# Make sure Ollama is running
ollama serve

# Run the example
go run examples/ollama/main.go
```

### Caching Example

```bash
go run examples/with_cache/main.go
```

This example demonstrates the performance improvement from caching - the second execution is ~900,000x faster!

### Thinker Middleware Example (NEW)

```bash
go run examples/thinker_middleware/main.go
```

Demonstrates the powerful middleware system for enhancing the Think phase with logging, retry, caching, RAG, rate limiting, and more. See [MIDDLEWARE_GUIDE.md](./docs/MIDDLEWARE_GUIDE.md) for details.

### Prompt Toolkit Example (NEW)

```bash
go run examples/prompt_toolkit/main.go
```

Demonstrates the complete Prompt construction and context management toolbox. Learn how to:
- Format tools for different LLMs (JSON Schema, Markdown, Compact)
- Count tokens accurately (supports Chinese/English/Mixed)
- Compress conversation history intelligently
- Debug and optimize Prompt construction

See [PROMPT_TOOLKIT_USAGE.md](./docs/PROMPT_TOOLKIT_USAGE.md) for practical usage guide.

## Configuration Options

```go
engine.New(
    engine.WithMaxIterations(10),           // Set max iterations
    engine.WithLLMClient(myLLMClient),      // Use custom LLM client (Ollama, OpenAI, etc.)
    engine.WithCache(myCache),              // Enable caching for performance
    engine.WithThinker(myThinker),          // Use custom thinker
    engine.WithActor(myActor),              // Use custom actor
    engine.WithObserver(myObserver),        // Use custom observer
    engine.WithLoopController(myController), // Use custom loop controller
)
```

### LLM Client Options

#### Ollama Client

```go
ollama.NewOllamaClient(
    ollama.WithModel("qwen3:0.6b"),              // Model name
    ollama.WithTemperature(0.7),                 // Temperature (0.0-1.0)
    ollama.WithBaseURL("http://localhost:11434"), // Ollama server URL
    ollama.WithTimeout(60 * time.Second),        // Request timeout
)
```

#### Mock Client (for testing)

```go
mock.NewMockClient([]string{
    "Thought: ...\nAction: ...",
    "Thought: ...\nFinal Answer: ...",
})
```

### Cache Options

```go
cache.NewMemoryCache(
    cache.WithMaxSize(100),                    // Maximum cache entries
    cache.WithDefaultTTL(1 * time.Hour),       // Default time-to-live
)
```

## Current Limitations

- No error retry mechanism (coming in Phase 2.1)
- No task decomposition
- Basic loop control (iteration count only)
- No distributed caching (Redis support coming later)

## Roadmap

### ✅ Phase 1: MVP (Completed)
- Core ReAct loop
- Tool system
- Mock LLM for testing

### ✅ Phase 2: Real LLM & Performance (Completed)
- Ollama integration
- Intelligent caching system
- Enhanced built-in tools (HTTP, DateTime)
- External LLM design pattern

### ✅ Phase 2.1: Reliability (Completed)
- Error handling and retry mechanism
- Better error messages
- Graceful degradation

### ✅ Phase 3: Advanced Features (Completed)
- **Thinker Middleware System**: Composable middleware for Think phase enhancement
- Multi-agent coordination
- SKILLS system with evaluation
- Task decomposition
- RAG extension point
- Monitoring and metrics
- Support for more LLM providers (OpenAI, Anthropic, etc.)

## Project Structure

```
goreact/
├── pkg/
│   ├── engine/      # Core engine implementation
│   ├── core/        # Core modules (Thinker, Actor, Observer, etc.)
│   ├── tool/        # Tool system
│   │   └── builtin/ # Built-in tools (Calculator, HTTP, DateTime, Echo)
│   ├── llm/         # LLM client interface
│   │   ├── ollama/  # Ollama client implementation
│   │   ├── openai/  # OpenAI client implementation
│   │   ├── anthropic/ # Anthropic client implementation
│   │   └── mock/    # Mock client for testing
│   ├── cache/       # Caching system
│   ├── agent/       # Multi-agent coordination system
│   ├── skill/       # SKILLS system
│   ├── task/        # Task decomposition system
│   ├── rag/         # RAG extension point
│   ├── metrics/     # Monitoring and metrics system
│   └── types/       # Shared type definitions
└── examples/        # Example programs
    ├── simple/      # Simple example with mock LLM
    ├── calculator/  # Calculator example with mock LLM
    ├── ollama/      # Ollama integration example
    ├── with_cache/  # Caching demonstration
    ├── thinker_middleware/ # Middleware system demonstration
    └── multi_agent/ # Multi-agent coordination example
```

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

MIT License

## Performance

With caching enabled, repeated tasks show dramatic performance improvements:

- **First execution**: ~7-14 seconds (with real LLM)
- **Cached execution**: ~7 microseconds
- **Speedup**: ~900,000x faster

Cache is automatically invalidated based on TTL or can be manually cleared.

## Implementing Your Own LLM Client

GoReAct is designed to work with any LLM provider. Simply implement the `llm.Client` interface:

```go
package llm

type Client interface {
    Generate(prompt string) (string, error)
}
```

Example for OpenAI:

```go
type OpenAIClient struct {
    apiKey string
    model  string
}

func (c *OpenAIClient) Generate(prompt string) (string, error) {
    // Call OpenAI API
    // Parse response
    // Return generated text
}
```

Then use it with the engine:

```go
openaiClient := NewOpenAIClient("your-api-key", "gpt-4")
eng := engine.New(
    engine.WithLLMClient(openaiClient),
)
```

## References

- [ReAct Paper](https://arxiv.org/abs/2210.03629)
- [Architecture Documentation](./ARCHITECTURE.md)
- [Ollama](https://ollama.com/)
