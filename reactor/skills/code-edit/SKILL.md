---
name: code-edit
description: >
  Edit, modify, and refactor source code files with precision.
  Use when the user needs to change, fix, refactor, or improve existing code.
allowed-tools: read glob grep file_edit write
---

# Code Edit

Make precise modifications to source code files while preserving code structure and intent.

## When to Activate
Use this skill when the user asks to:
- Fix a bug in an existing file
- Refactor or reorganize code
- Add new functionality to existing files
- Update imports, dependencies, or configuration
- Rename variables, functions, or types across files
- Apply coding patterns or best practices

## Workflow

### 1. Understand Before Editing
- **Read first**: Always use `read` to understand the full file content before making changes.
- **Find context**: Use `grep` to locate relevant code patterns and their surroundings.
- **Identify scope**: Use `glob` to find all files that may need coordinated changes.

### 2. Plan the Edit
- Identify exact lines or sections to change.
- Consider side effects (imports, tests, dependent functions).
- For multi-file changes, establish the correct order of edits.

### 3. Execute Edits
- **Targeted changes**: Use `file_edit` for precise edits (find/replace within a file).
- **Small files**: Use `write` to rewrite small files entirely when most content changes.
- **New files**: Use `write` to create new files that don't exist yet.

### 4. Verify
- After editing, use `read` to confirm the result matches intent.
- Check for syntax issues (unbalanced brackets, missing commas).
- If using a language with a compiler/linter, verify the edit doesn't break builds.

## Tool Reference

| Tool | Best For | Example |
|------|----------|---------|
| read | Understanding current file content | Read before editing |
| glob | Finding related files by pattern | `**/*.go`, `src/**/*.ts` |
| grep | Locating specific code patterns | `"func.*Handler"`, `"TODO"` |
| file_edit | Precise targeted edits (find/replace) | Replace a function body |
| write | Full file rewrite or new creation | Rewrite a config file |

## Guidelines
- Never edit without reading the target section first.
- Preserve existing code style and formatting conventions.
- Make minimal, focused changes — avoid reformatting unrelated code.
- When renaming symbols, search for all references with `grep` first.
- After editing, briefly describe what changed and why.
