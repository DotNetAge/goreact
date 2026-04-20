---
name: remember
description: >
  Manage project conventions, instructions, and shared memory.
  Use when the user mentions remember, convention, instruction, memory, or documentation.
allowed-tools: grep bash read
---

# Remember: Memory & Convention Review

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
   - No action needed
