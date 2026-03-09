# Load Skill from SKILL.md File Example

This example demonstrates how to load Agent Skills from SKILL.md files following the [Agent Skills specification](https://agentskills.io).

## Skill Structure

```
skills/math-wizard/
├── SKILL.md              # Required: Frontmatter + Instructions
├── scripts/              # Optional: Executable scripts
│   └── verify.sh
└── references/           # Optional: Reference documentation
    └── order-of-operations.md
```

## SKILL.md Format

```markdown
---
name: math-wizard
description: Expert mathematical problem solver
license: MIT
compatibility: Works with any calculator tool
metadata:
  version: "1.0.0"
  author: "GoReAct Team"
allowed-tools: calculator echo
---

# Instructions

Your step-by-step instructions here...
```

## Running the Example

```bash
cd examples/skill_from_file
go run main.go
```

## What This Example Does

1. ✅ Loads skill from SKILL.md file
2. ✅ Parses YAML frontmatter (name, description, metadata, etc.)
3. ✅ Loads markdown instructions
4. ✅ Loads optional scripts and references
5. ✅ Registers skill with SkillManager
6. ✅ Executes task using the loaded skill
7. ✅ Tracks and displays statistics

## Expected Output

- Skill metadata display
- Loaded scripts and references list
- Task execution with skill instructions
- Execution trace
- Skill statistics

## Key Features

- **Frontmatter Parsing**: YAML metadata extraction
- **Instruction Loading**: Markdown body parsing
- **Optional Content**: Scripts and references support
- **Statistics Tracking**: Usage, success rate, performance metrics
- **Skill Selection**: Automatic matching based on task description
