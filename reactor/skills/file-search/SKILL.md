---
name: file-search
description: >
  Search, locate, and read file contents from the codebase.
  Use when the user needs to find files, search for code patterns, or read and understand file contents.
---

# File Search

Search, locate, and read file contents from the codebase to understand project structure and navigate code.

## When to Activate
Use this skill when the user asks to:
- Find a file or set of files matching a pattern (e.g., "*.go", "config.*")
- Search for text, code, or symbols within files
- Read and understand the contents of specific files
- Explore project structure or navigate the codebase
- Locate where a function, class, or variable is defined or used
- Understand how different parts of a codebase relate to each other

## Required Capabilities

This skill requires the following capabilities from the agent platform:

1. **File Discovery by Pattern** — Ability to locate files matching name patterns, wildcards (e.g., `**/*.go`, `src/**/*.ts`), or file type filters across directory trees. This enables fast discovery of candidate files when you know naming conventions but not exact paths.

2. **File Content Retrieval** — Ability to retrieve the full text content of a file given its path, including support for reading specific line ranges when only a portion is relevant. Essential for understanding implementation details after locating a target file.

3. **Text Pattern Search** — Ability to search for text, regular expressions, or symbolic patterns within file contents across multiple files. Returns matched lines with surrounding context for understanding usage patterns. Critical for finding where functions are called, variables are referenced, or patterns appear.

## Workflow

### 1. Locate Files
- **By pattern**: When you know the naming convention but not the exact path, use file pattern matching to discover candidate files across the codebase. Start broad then narrow down with more specific patterns if results are too numerous.

- **By content**: When you know what you're looking for (a function name, an error message, a TODO comment) but don't know which file contains it, use text pattern searching across files to locate matches.

### 2. Read Contents
Once target files are identified through step 1:

- **Single file**: Retrieve the full contents to understand the complete implementation. If the file is large, focus on reading the relevant sections using line-range retrieval when available.

- **Multiple files**: After discovering candidate files via pattern matching, retrieve each matched file's contents systematically. Prioritize files most likely to contain relevant information based on names and paths.

- **Specific lines**: After retrieving content, examine the specific portions that match your search criteria. Cross-reference findings across multiple files when tracing dependencies or call chains.

### 3. Refine and Iterate
If initial search results are too broad or miss the target:

- Narrow the file pattern by adding directory constraints, more specific extensions, or exclusion filters.
- Make the text search pattern more specific: add word boundaries, require case sensitivity, include surrounding context terms, or use more precise regular expressions.
- Chain capabilities strategically: first discover candidate files via pattern matching, then filter those candidates via text search, then retrieve the most promising matches for detailed analysis.
- Use findings from one file to inform searches in related files (e.g., finding imports to discover dependent modules).

## Guidelines
- Always use file pattern discovery before attempting blind content retrieval when you don't know the exact file path.
- Prefer text pattern search over retrieving files and manually scanning when searching for specific strings, symbols, or patterns — it's faster and more thorough.
- Chain capabilities in this order: discover → filter → retrieve → analyze. Each step informs the next.
- When exploring unfamiliar codebases, start with broad pattern discovery to understand the project layout before diving into specific files.
- Keep track of which files you've already examined to avoid redundant work on large searches.
- Consider both the definition site and usage sites when searching for symbols — a function may be defined in one file but used across many others.
