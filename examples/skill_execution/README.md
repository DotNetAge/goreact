# Skill Execution Example

This example demonstrates how to use the Skills system in GoReAct.

## What are Skills?

Skills are reusable instruction sets that guide the Agent on how to perform specific tasks. Based on the [Agent Skills specification](https://agentskills.io), Skills package:

- **Instructions**: Step-by-step guidance for the Agent
- **Scripts**: Executable code (optional)
- **References**: Documentation (optional)
- **Statistics**: Performance tracking and ranking

## How It Works

1. **Discovery**: SkillManager loads available skills
2. **Matching**: Engine selects the most appropriate skill for the task
3. **Activation**: Skill instructions are injected into the LLM context
4. **Execution**: Agent follows the instructions step-by-step
5. **Recording**: Statistics are tracked for skill evolution

## Running the Example

```bash
cd examples/skill_execution
go run main.go
```

## Expected Output

The example will:
1. Create a calculation skill with step-by-step instructions
2. Execute a math task using the skill
3. Show execution trace
4. Display skill statistics and ranking

## Key Features Demonstrated

- ✅ Skill creation and registration
- ✅ Automatic skill selection based on task
- ✅ Instruction injection into Agent context
- ✅ Execution statistics tracking
- ✅ Skill ranking system

## Next Steps

- Load skills from SKILL.md files
- Use scripts and references
- Implement skill evolution (优胜劣汰)
- Add more complex skills
