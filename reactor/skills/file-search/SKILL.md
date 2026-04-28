---
name: file-search
description: >
  Search and read file contents using glob patterns, grep, and direct file reading.
  Use when the user needs to find files, search for code patterns, or read file contents.
allowed-tools: glob read grep
---

# File Search

Search, locate, and read file contents from the codebase.

## When to Activate
Use this skill when the user asks to:
- Find a file or set of files matching a pattern (e.g., "*.go", "config.*")
- Search for text, code, or symbols within files
- Read and understand the contents of specific files
- Explore project structure or navigate the codebase

## Workflow

### 1. Locate Files
- **By pattern**: Use `glob` to find files matching a glob pattern. This is fast and works across directories.
- **By content**: Use `grep` to search for text/regex patterns inside files when you know what you're looking for but not which file contains it.

### 2. Read Contents
Once target files are identified:
- **Single file**: Use `read` to get full file contents.
- **Multiple files**: Combine `read` with `glob` to read each matched file.
- **Specific lines**: Use `read` then examine the relevant portion of output.

### 3. Refine Search
If initial results are too broad:
- Narrow the `glob` pattern or add directory constraints.
- Make the `grep` regex more specific (add word boundaries, context).
- Use `glob` first to identify candidate files, then `grep` on those results.

## Tool Reference

| Tool | Best For | Example |
|------|----------|---------|
| glob | Finding files by name/pattern | `**/*.go`, `src/**/*.ts` |
| grep | Searching file contents | `"func.*Handler"`, `"TODO"` |
| read | Reading complete file content | `/path/to/file.go` |

## Tips
- Always use `glob` before `read` when you don't know the exact file path.
- Prefer `grep` over `read`+manual scan when searching for specific strings.
- Chain tools: `glob` -> filter -> `read` -> analyze.
