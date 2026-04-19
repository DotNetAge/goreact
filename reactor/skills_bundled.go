package reactor

import (
	"github.com/DotNetAge/goreact/core"
)

const debugPrompt = `# Debug: Session & Bug Analysis

Help the user debug an issue they're encountering in the project or session.

## Instructions
1. **Gather Context**: Use 'grep' and 'read' to locate [ERROR], [WARN], stack traces, and failure patterns in recent logs or code.
2. **Analyze**: Understand the root cause. If the issue is complex, consider launching a subagent ('task_create') to deeply analyze the specific module.
3. **Reproduce & Trace**: Identify the exact steps or code paths that lead to the error.
4. **Explain & Suggest**: Explain what you found in plain language, and suggest concrete fixes or next steps. Provide actionable solutions rather than just listing errors.`

const architectPrompt = `# Architect: System Design & Refactoring

High-level orchestration for system design and major migrations.

## Phase 1: Research & Plan
1. **Analyze**: Use 'glob' and 'grep' to understand the project structure and patterns.
2. **Plan**: Define a multi-phase plan. Break it down into independent work units. Output this plan clearly.

## Phase 2: Delegate
1. **Task Decomposition**: Break the work into self-contained units.
2. **Spawn Sub-agents**: Use 'task_create' to spawn sub-agents for each unit, passing precise instructions and context.

## Phase 3: Govern & Integrate
1. **Monitor**: Check the progress and output of sub-agents.
2. **Integrate**: Perform the final assembly and cross-module verification to ensure architectural consistency.`

const batchPrompt = `# Batch: Parallel Work Orchestration

You are orchestrating a large, parallelizable change across this codebase.

## Phase 1: Research and Plan
1. **Understand the scope**: deeply research what this instruction touches. Find all the files, patterns, and call sites that need to change.
2. **Decompose into independent units**: Break the work into multiple self-contained units. Each unit must be independently implementable.
3. **Determine the test recipe**: Figure out how a worker can verify its change actually works (e.g. unit tests, e2e recipe).
4. **Write the plan**: Output the numbered list of work units.

## Phase 2: Spawn Workers
Spawn one background agent per work unit using the 'task_create' tool. Launch them all in a single message block so they run in parallel.
For each agent, the prompt must be fully self-contained including the specific task and codebase conventions.

## Phase 3: Track Progress & Review
Render an initial status table. After workers finish, parse their results, verify using 'bash', and render a final summary of completed vs failed units.`

const verifyPrompt = `# Verify: Code Change QA

Verify a code change does what it should by running the app and tests.

## Phase 1: Locate tests
1. Check for 'package.json', 'Makefile', 'go.mod', or other build files to find test commands.
2. Find related test files for the recently modified code.

## Phase 2: Execution
1. **Unit Tests**: Run relevant tests using 'bash'.
2. **Lint & Static Analysis**: Run the project's linter.
3. **E2E/Integration**: If applicable, start the dev server and hit endpoints using 'web_fetch' or curl in 'bash'.

## Phase 3: Report & Cleanup
1. Document which tests passed and which failed.
2. If tests failed, automatically attempt to fix the code or the tests if it's a simple discrepancy.
3. Remove any side effects or temporary files.`

const rememberPrompt = `# Remember: Memory & Convention Review

Review the user's memory landscape and produce a clear report of proposed changes, grouped by action type. Do NOT apply changes — present proposals for user approval.

## Steps
1. **Gather**: Read memory files like CLAUDE.md, CLAUDE.local.md, or project specific instruction files.
2. **Classify**: For each new substantive entry, determine the best destination:
   - Project conventions (e.g. "use bun not npm")
   - Personal instructions (e.g. "I prefer concise responses")
3. **Identify Cleanup**: Scan across layers for duplicates, outdated instructions, or conflicts.
4. **Present Report**: Output a structured report grouped by:
   - Promotions (entries to move)
   - Cleanup (duplicates/outdated)
   - Ambiguous (needs user input)
   - No action needed`

const stuckPrompt = `# Stuck: Diagnose Frozen/Slow State

Strategy to break free when the agent is repeating actions, failing, or stuck in a loop.

## Phase 1: Diagnosis
1. **Analyze**: Identify why the previous attempts failed by reading recent conversation history.
2. **Identify Loops**: Are you repeatedly calling the same tool with the same arguments and getting the same error?

## Phase 2: Pivot
1. Change your approach. If Grep failed, try Glob. If Bash failed, try to write a small script and execute it.
2. Simplify the problem. Isolate the failing component and write a minimal reproduction test.

## Phase 3: Action
1. If the environment is hung (e.g., high CPU, stuck processes), use 'bash' to run 'ps' or 'top' and kill hung processes if necessary.
2. If still stuck after pivoting, formulate a clear question and ask the user for guidance rather than continuing to loop.`

const simplifyPrompt = `# Simplify: Code Review and Cleanup

Review all changed files for reuse, quality, and efficiency. Fix any issues found.

## Phase 1: Identify Changes
Run 'git diff' to see what changed, or review the recently modified files.

## Phase 2: Review (Simulated Parallel)
Review the changes across three dimensions:
1. **Code Reuse**: Search for existing utilities and helpers that could replace newly written code. Flag logic that duplicates existing utilities.
2. **Code Quality**: Look for hacky patterns: redundant state, parameter sprawl, copy-paste variations, stringly-typed code, unnecessary comments.
3. **Efficiency**: Look for unnecessary work (redundant file reads, N+1 patterns), missed concurrency, memory leaks, and overly broad operations.

## Phase 3: Fix Issues
Aggregate the findings and fix each issue directly using 'replace_in_file' or 'file_edit'.
Briefly summarize what was fixed.`

// RegisterBundledSkills registers common high-level skills based on CludeCode patterns.
func RegisterBundledSkills(registry core.SkillRegistry) {
	_ = registry.RegisterSkill(&core.Skill{
		Name:         "Bug Hunter",
		Description:  "Expert SOP for locating, isolating and fixing complex bugs.",
		Instructions: debugPrompt,
		Tools:        []string{"grep", "glob", "bash", "task_create", "read_file"},
		TriggerRules: []string{"bug", "fix", "error", "crash", "failed", "debug"},
	})

	_ = registry.RegisterSkill(&core.Skill{
		Name:         "Architect",
		Description:  "High-level orchestration for system design and major migrations.",
		Instructions: architectPrompt,
		Tools:        []string{"glob", "grep", "task_create", "todo_write"},
		TriggerRules: []string{"architecture", "design", "refactor", "migrate"},
	})

	_ = registry.RegisterSkill(&core.Skill{
		Name:         "Batch",
		Description:  "Parallel orchestration of large-scale mechanical changes.",
		Instructions: batchPrompt,
		Tools:        []string{"grep", "task_create", "bash"},
		TriggerRules: []string{"batch", "bulk", "replace all", "migrate all"},
	})

	_ = registry.RegisterSkill(&core.Skill{
		Name:         "Verify",
		Description:  "Rigorous verification of changes through testing and execution.",
		Instructions: verifyPrompt,
		Tools:        []string{"bash", "todo_write"},
		TriggerRules: []string{"verify", "test", "check", "qa"},
	})

	_ = registry.RegisterSkill(&core.Skill{
		Name:         "Remember",
		Description:  "Manage project conventions, instructions, and shared memory.",
		Instructions: rememberPrompt,
		Tools:        []string{"grep", "bash", "read_file"},
		TriggerRules: []string{"remember", "convention", "instruction", "memory", "documentation"},
	})

	_ = registry.RegisterSkill(&core.Skill{
		Name:         "Stuck",
		Description:  "Strategy to break free when the agent is repeating actions or failing.",
		Instructions: stuckPrompt,
		Tools:        []string{"grep", "glob", "bash"},
		TriggerRules: []string{"stuck", "loop", "repeat", "failing"},
	})

	_ = registry.RegisterSkill(&core.Skill{
		Name:         "Simplify",
		Description:  "Post-implementation cleanup to ensure code quality and simplicity.",
		Instructions: simplifyPrompt,
		Tools:        []string{"replace_in_file", "bash", "read_file"},
		TriggerRules: []string{"simplify", "cleanup", "refactor", "polish"},
	})
}

