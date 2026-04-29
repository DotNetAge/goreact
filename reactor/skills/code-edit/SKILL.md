---
name: code-edit
description: >
  Edit, modify, and refactor source code files with precision.
  Use when the user needs to change, fix, refactor, or improve existing code.
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

## Required Capabilities

This skill requires the following capabilities from the agent platform:

1. **File Content Retrieval** — Ability to retrieve the full text content of a file given its path or identifier, including the ability to read specific line ranges when only a portion of the file is needed.

2. **Text Pattern Search** — Ability to search for text patterns, function names, symbols, or identifiers within files across the codebase using pattern matching or regular expressions, returning matched lines with surrounding context for understanding usage context.

3. **File Discovery by Pattern** — Ability to locate files matching name patterns, wildcards, or file type filters across directory trees, enabling identification of all files that may be affected by coordinated changes.

4. **Precise Localized Editing** — Ability to make surgical modifications to specific sections of an existing file (e.g., replace a function body, update an import statement, fix a parameter list) at specific line ranges or anchored by surrounding text context (find/replace semantics).

5. **Full File Writing** — Ability to write complete content to a given file path, creating the file if it does not exist yet, used when most of a file's content needs to change or when creating brand new files.

## Workflow

### 1. Understand Before Editing
- **Examine first**: You MUST retrieve and review the target's complete contents before making any modifications. Never edit blindly without understanding the current state of the file.
  
- **Find context**: Search for relevant code patterns, function names, symbols, or identifiers within the codebase to understand how the code you're about to change is used elsewhere. Look for callers, dependents, and related implementations.

- **Identify scope**: Discover all files that may be affected by your changes using name pattern matching. Consider imports, tests, type definitions, and configuration files that might need coordinated updates.

### 2. Plan the Edit
- Identify exact lines or sections to change based on your understanding from step 1.
- Consider side effects: imports that need adding/removing, tests that need updating, dependent functions whose signatures may change.
- For multi-file changes, establish the correct order of edits to avoid breaking references mid-way.
- Estimate the scope: targeted changes vs. full rewrite.

### 3. Execute Edits
- **Targeted changes**: When only specific sections of a file need modification (e.g., replacing a function body, updating an import, fixing parameters), use precise localized editing at the identified location. This preserves the rest of the file unchanged.
  
- **Full rewrite or new creation**: When most of a file's content needs to change, or when creating a brand new file that doesn't exist yet, use full file writing to output the complete desired content.

### 4. Verify
- After editing, re-examine the file to confirm the result matches the intended changes.
- Check for syntax issues: unbalanced brackets, missing commas, mismatched indentation.
- If the language has a compiler or linter available, verify the edit doesn't introduce build errors or warnings.
- Ensure no unintended modifications were made to unrelated sections.

## Guidelines
- Never edit without reading and understanding the target section first.
- Preserve existing code style, formatting conventions, and idiomatic patterns already in use in the file.
- Make minimal, focused changes — avoid reformatting or modifying unrelated code alongside your targeted edits.
- When renaming symbols, perform a thorough search for all references across the codebase before renaming to avoid broken references.
- When changing function signatures or interfaces, identify and update all call sites.
- After editing, briefly describe what changed and why, focusing on the semantic intent rather than mechanical details.
- Prefer small, incremental edits over large rewrites when possible — each edit should be verifiable independently.
