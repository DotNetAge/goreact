---
name: task-manager
description: >
  Manage tasks, todos, and action items with creation, tracking, and execution.
  Use when the user needs to organize work, track progress, or manage a to-do list.
allowed-tools: todo_write todo_read todo_execute
---

# Task Manager

Create, read, update, and execute task/action items for personal or project productivity.

## When to Activate
Use this skill when the user asks to:
- Create a new task, todo item, or reminder
- Check what tasks are pending or completed
- Mark a task as done or update its status
- Execute a task (run its associated command or action)
- Organize a work plan or checklist

## Workflow

### 1. Create Tasks
- Use `todo_write` to create new tasks with clear titles and descriptions.
- Include relevant metadata: priority, due date, tags, status.
- Break large goals into smaller actionable sub-tasks.

### 2. Review Tasks
- Use `todo_read` to list existing tasks by status (pending, completed, all).
- Filter by tags, priority, or date range to focus on what matters.
- Review completed tasks for progress tracking.

### 3. Execute and Update
- For actionable tasks, use `todo_execute` to run the associated action.
- Update task status after completion or when priorities change.
- Regularly review and clean up outdated tasks.

## Tool Reference

| Tool | Best For | Example |
|------|----------|---------|
| todo_write | Creating or updating tasks | Create "Fix login bug" |
| todo_read | Listing and filtering tasks | Show pending tasks |
| todo_execute | Running a task's action | Execute task #42 |

## Best Practices
- Use descriptive, actionable task titles.
- Set reasonable priorities — not everything is urgent.
- Review regularly; delete completed tasks periodically.
- Link task execution to concrete commands or actions.
