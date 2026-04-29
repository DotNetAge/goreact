---
name: task-manager
description: >
  Manage tasks, todos, and action items with creation, tracking, and execution.
  Use when the user needs to organize work, track progress, or manage a to-do list.
---

# Task Manager

Create, track, update, and execute task/action items for organizing work and managing productivity.

## When to Activate
Use this skill when the user asks to:
- Create a new task, todo item, or reminder
- Check what tasks are pending, in progress, or completed
- Mark a task as done or update its status
- Organize a work plan, checklist, or breakdown of complex work
- Track progress across multiple parallel work streams
- Prioritize and reorder tasks based on changing requirements
- Review completed work and clean up outdated items

## Required Capabilities

This skill requires the following capabilities from the agent platform:

1. **Task Creation and Update** — Ability to create new tasks with descriptive titles, detailed descriptions, and metadata such as priority level, status state, tags/categories, and due dates. Also supports updating existing tasks to modify any of these fields as work progresses or requirements change.

2. **Task Listing and Query** — Ability to list existing tasks with filtering and sorting options: by status (pending, in-progress, completed), by priority level, by tag or category, by date range, or by free-text search. Enables focused views on what matters most at any given time.

3. **Task Execution** — Ability to execute a task's associated action or command. A task may carry an executable payload (a command, script, or action reference) that should be run when the task transitions to an active state. The execution result should feed back into task status updates.

## Workflow

### 1. Create Tasks
- Create new tasks with clear, actionable titles that describe what needs to be done, not just the topic.
- Include rich descriptions that provide enough context for anyone (including another agent session) to understand the task's purpose and requirements.
- Attach relevant metadata: priority level to indicate urgency, status to track lifecycle state, tags for categorization and cross-cutting concerns.
- Break large goals into smaller, independently verifiable sub-tasks. Each sub-task should be completable in a single focused effort.
- Establish dependencies between tasks when order matters — some tasks should only start after their predecessors complete.

### 2. Review and Monitor
- List and filter tasks by status to focus attention on pending and in-progress items.
- Review completed tasks periodically to assess overall progress and identify patterns (what types of tasks take longer, what gets blocked).
- Use priority filtering to ensure high-priority items aren't lost among lower-priority noise.
- Check for stalled tasks that have been in-progress too long without completion — they may be blocked or need re-scoping.

### 3. Execute and Update
- For actionable tasks that carry executable payloads, trigger execution at the appropriate time.
- Update task status promptly upon completion or when priorities/requirements change.
- When a task completes, check whether any dependent tasks are now unblocked and ready to start.
- Record outcomes and learnings from completed tasks when useful for future similar work.
- Regularly review and clean up outdated or obsolete tasks to keep the list maintainable.

## Best Practices
- Use descriptive, verb-oriented task titles (e.g., "Fix login timeout bug" rather than just "Login").
- Set reasonable priorities — not everything is urgent. Reserve high priority for truly blocking issues.
- Keep task granularity balanced: too coarse and progress is invisible; too fine and overhead dominates.
- Review regularly; archive or delete completed tasks periodically to prevent list bloat.
- Link task execution to concrete, reproducible actions so tasks are verifiable.
- When breaking down complex work, make sure each sub-task has clear acceptance criteria.
