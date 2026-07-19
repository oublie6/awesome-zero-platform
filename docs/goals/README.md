# Codex Goal Workflow

This directory controls short-lived Codex execution goals without turning conversation history into the source of truth.

## Files

- `current.md` — the only active execution goal.
- `template.md` — template for the next goal.
- `archive/` — completed goals, retained for history but not loaded by default.
- `summaries/` — optional phase summaries created when archived goals become numerous.

## Goal states

- `idle` — no executable goal is active.
- `ready` — the goal is defined and may be started.
- `in_progress` — implementation has started and may be resumed.
- `completed` — all acceptance criteria passed and the goal is ready to archive.
- `blocked` — execution stopped because a genuine blocker is documented.

Only `ready` and `in_progress` goals may be started by `scripts/run-current-goal.sh`. The archive helper accepts only `completed` goals.

## Lifecycle

1. Discuss and refine a feature or engineering task.
2. Update the long-lived architecture, requirement, and decision documents.
3. Replace `current.md` with a focused `ready` goal that references only the documents required for this execution.
4. Start a fresh Codex session with `scripts/run-current-goal.sh`.
5. Codex changes the state to `in_progress`, implements the goal, runs acceptance checks, and updates only the permitted status and completion sections.
6. After every acceptance criterion passes, Codex changes the state to `completed`, commits the verified work, and pushes the current branch.
7. Archive the goal with `scripts/archive-goal.sh <goal-name>`.
8. Create the next `current.md` from `template.md`.

## Context policy

Codex should read:

1. `AGENTS.md`
2. `docs/goals/current.md`
3. documents explicitly referenced by the current goal
4. source files required to implement and verify the goal

Codex must not load archived goals unless the current goal explicitly requests historical investigation.

Archived goals are kept until repository size becomes a practical concern. Cleanup should be deliberate and may be preceded by a phase summary. Git history remains the final historical record.
