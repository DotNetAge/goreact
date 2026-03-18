# GoReAct Testing & Remediation Plan (v1.0 Release Candidate)

## 🎯 Overview & Objectives
The GoReAct project is currently in a transitional state (a "construction site" post-refactoring). The goal of this testing plan is to systematically repair broken dependencies, achieve high test coverage for the core pipeline, and validate the system's readiness for a production `v1.0` release.

### Target Metrics
- **Build Success**: 100% of the codebase compiles (`go build ./...`).
- **Core Coverage**: >80% code coverage on `pkg/engine`, `pkg/core`, `pkg/thinker`, `pkg/actor`, `pkg/observer`, and `pkg/terminator`.
- **E2E Reliability**: 0 panics during long-running ReAct loops under mocked load.

---

## 🛠️ Phase 1: CI/CD & Compilation Sanity

**Objective**: Ensure the project compiles cleanly and does not rely on local environment hacks.

### Tasks
1. **Dependency Cleansing (`go.mod`)**:
   - Remove `replace github.com/DotNetAge/gochat => ../gochat`.
   - Update to a published remote version/tag of `gochat`.
   - Run `go mod tidy` and verify dependencies.
2. **Ghost Code Remediation (`pkg/tools/builtin_test.go`)**:
   - Delete or refactor tests for deleted tools: `TestEcho`, `TestHTTP`, `TestCurl`, `TestPort`.
   - Ensure the test suite for existing tools (`Calculator`, `DateTime`, `Bash`, `Grep`, `Read`, `Write`) passes.
3. **Architectural Cleanup**:
   - Review `pkg/agent`. If `Coordinator` and `TaskDecomposer` are unintegrated legacy code (relying on ghost singletons), move them to `experimental/` or delete them to avoid confusing the compiler and future contributors.

**Exit Criteria**:
- `go build ./...` exits with code 0.
- `go test ./pkg/tools/...` passes completely.

---

## 🔬 Phase 2: Core State Machine Verification (Unit Testing)

**Objective**: Validate the structural integrity of the `Thinker -> Actor -> Observer -> Terminator` pipeline and fix the known state bug.

### Tasks
1. **Fix `reactor_test.go` Bug**:
   - Diagnose the `Expected CurrentStep to be 3, got 5` failure in `TestReactor_Run`.
   - Ensure the internal state transitions strictly follow: `Think (Step 1) -> Act (Step 2) -> Observe (Step 3) -> Check Finish (Step 4)`.
2. **Mock the Core Abstractions**:
   - Create table-driven tests for `pkg/thinker`. Verify that varying mock LLM responses (e.g., JSON vs Markdown Action blocks) are correctly parsed into `Thought` and `Action`.
   - Create table-driven tests for `pkg/actor`. Verify that it correctly maps to the Tool registry, executes the tool, and handles `ToolNotFound` errors gracefully.
   - Create table-driven tests for `pkg/observer`. Verify that raw bytes/errors from the Actor are accurately wrapped into `Observation` structs.
3. **Pipeline Context Tests**:
   - Verify `PipelineContext` passes data safely between steps without race conditions.
   - Test early termination rules (e.g., when `Terminator` hits the max iteration limit).

**Exit Criteria**:
- `go test -v ./pkg/engine ./pkg/core ./pkg/thinker ./pkg/actor ./pkg/observer ./pkg/terminator` passes with >80% coverage.

---

## 🧰 Phase 3: Integration & Tooling Validation

**Objective**: Ensure that built-in tools behave reliably when orchestrated by the engine.

### Tasks
1. **Prompt Toolkit Verification**:
   - Test the Token Counter with edge cases (mixed English/Chinese, emojis).
   - Test the History Sliding Window / Compression logic. Verify that older interactions are pruned correctly when approaching the token limit.
2. **Built-in Tool Isolation Tests**:
   - Verify `Bash` and filesystem tools (`Read`, `Write`, `Grep`) properly jail their execution paths if required, or at least return safe error messages for bad commands.
3. **Mock Reactor End-to-End**:
   - Write a test using `mock.Client` where the mock LLM issues a sequence of tool calls (e.g., `Calculator` -> `DateTime` -> `FinalAnswer`).
   - Validate the sequence completes and `FinalResult` matches the expected outcome.

**Exit Criteria**:
- All tool execution paths, including error handling, are covered.
- Prompt builders generate valid JSON Schema / XML / Markdown as required by different LLMs.

---

## 🤖 Phase 4: Agent & E2E Validation (The AAAT Architecture)

**Objective**: Prove the system works with real LLMs and validate the "Agent-as-a-Tool" (AAAT) vision.

### Tasks
1. **Real LLM Integration Test (Manual/Integration)**:
   - Run `examples/qwen_agent/main.go` using an actual API key.
   - Verify the agent can correctly parse a complex prompt ("If today is 2026-03-18...").
2. **Nested Agent (AAAT) Test**:
   - Create an integration test where a `Supervisor Reactor` registers a `Sub-Agent Reactor` as a Tool.
   - Give the Supervisor a task requiring the Sub-Agent's specific skillset.
   - Verify the Sub-Agent boots its own loop, resolves its sub-task, and returns a string `Observation` to the Supervisor, which then finishes the main task.

**Exit Criteria**:
- The real-world examples in the `examples/` directory execute successfully.
- Nested AAAT pattern is empirically proven to work.

---

## 🌪️ Phase 5: Chaos, Performance, and Edge Cases

**Objective**: Ensure the framework won't collapse in production under network stress or bad LLM behaviors.

### Tasks
1. **Graceful Degradation Testing**:
   - Simulate an LLM timeout (using context cancellation) and ensure the Engine recovers or exits cleanly without panicking.
   - Simulate hallucinated/invalid JSON outputs from the LLM. Verify the `Thinker` generates an error observation and asks the LLM to correct itself in the next loop.
2. **Middleware Evaluation**:
   - Write a test for the `Thinker Middleware`. Ensure order of execution is correct (e.g., Cache -> RateLimit -> RAG -> Think).
3. **Concurrency Stress Test**:
   - Spin up 100 concurrent `Reactor` instances (using Mock LLMs) with localized state.
   - Run Go's race detector (`go test -race ./...`) to ensure no global state contamination exists in the registry or context maps.

**Exit Criteria**:
- 0 data races detected.
- System correctly retries/aborts based on configured timeout contexts.

---

## ✅ v1.0 Release Approval
Once all 5 phases are signed off and the `README.md` examples strictly match the working implementation, the project will be cleared for the `v1.0.0` stable release.
