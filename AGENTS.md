# AGENTS.md

## Project intent

Awesome Zero Platform is a modular application platform built on go-zero. It provides reusable server capabilities and supports multiple clients without coupling business modules to a specific frontend.

## Architecture rules

- Start as a modular monolith; extract services only when real scaling or ownership needs appear.
- `server/apps` contains runnable processes.
- `server/platform` contains reusable platform capabilities shared by different products.
- `server/business` contains product-specific business modules and should be created only when real business implementation begins.
- `server/foundation` contains reusable technical infrastructure without business semantics.
- Platform and business modules must not access another module's database tables or repository implementation directly.
- Cross-module calls must use explicit public interfaces or events.
- Keep transport, application logic, and persistence concerns separated.
- Do not create generic dumping grounds such as `common`, `utils`, or `helpers`.
- The repository stores the current complete database schema, not incremental migration history during the early development phase.
- Temporary database upgrade SQL must not be committed.

## Goal workflow

Supported goal states are:

- `idle` — no executable goal is active.
- `ready` — the goal is defined and may be started.
- `in_progress` — implementation has started and may be resumed.
- `completed` — all acceptance criteria passed and the goal is ready to archive.
- `blocked` — execution stopped because a genuine blocker is documented.

Only `ready` and `in_progress` goals may be executed by `scripts/run-current-goal.sh`.

Before planning or editing code:

1. Read this file completely.
2. Read `docs/goals/current.md` completely.
3. Read only the architecture, requirement, and decision documents explicitly referenced by the current goal, plus source files needed for implementation.
4. Confirm the working tree is clean before starting. Preserve any pre-existing local work and stop if the repository is not clean.
5. Synchronize the current branch with its configured upstream using `git pull --ff-only`. If synchronization cannot complete cleanly, stop and document the blocker.
6. Inspect the synchronized repository state and relevant Git diff.
7. Treat the current goal and referenced documents as authoritative.

Execution rules:

- Do not load archived goals unless the current goal explicitly requires historical investigation.
- Do not expand, reinterpret, or silently replace the goal, deliverables, constraints, or acceptance criteria.
- The primary agent owns architecture, integration, and final verification.
- Subagents may be used for independent analysis, implementation, testing, or review, but multiple agents must not modify the same files concurrently.
- Codex may update only the status, working-state, and completion-report sections of `docs/goals/current.md` unless the goal explicitly permits other documentation changes.
- Set the goal state to `in_progress` when implementation begins.
- Set the goal state to `completed` only after every acceptance criterion passes.
- Set the goal state to `blocked` only when a genuine blocker is recorded with evidence.
- When resuming after a long pause, context compaction, or substantial scope discussion, reread this file and the current goal before continuing.

Before finishing:

1. Run every acceptance check required by the current goal.
2. Update the permitted status and completion-report sections.
3. Inspect the final Git diff and ensure only goal-related changes are included.
4. Commit all completed goal changes with a concise, descriptive commit message.
5. Push the current branch to its configured upstream with `git push`. Do not use force push. If the push fails, document the exact blocker and keep the verified commit locally.
6. Summarize changed files, verification results, commit and push results, unresolved blockers, and intentionally deferred work.
7. Stop when the goal is completed and pushed, or when a genuine blocker is documented.

## Change rules

- Keep generated go-zero files distinguishable from handwritten code.
- Every public API change must update the relevant API documentation or schema.
- Every database structure change must update `server/database/schema`.
- Add tests for reusable foundation capabilities and module-level business rules.
- Prefer small, reviewable changes over broad speculative abstractions.
