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

## ChatGPT–Codex collaboration workflow

Use the following as the default ownership model unless the active goal explicitly assigns different responsibilities.

1. **ChatGPT is the primary design and implementation agent for substantial work.** ChatGPT clarifies requirements, defines the goal and acceptance criteria, designs architecture and contracts, changes production code, writes the first complete set of tests and documentation, runs focused verification, reviews the integrated diff, and commits and pushes the implementation checkpoint.
2. **GitHub Actions performs automatic baseline verification after commits.** CI should cover compilation, deterministic unit and package tests, repository regression, dependency boundaries, frontend builds, and appropriate baseline race checks.
3. **Codex performs independent follow-up verification.** Starting from a clean synchronized repository, Codex reruns every verification command required by the active goal and may add stronger execution variants such as shuffle, repeated race runs, stress counts, or low-resource builds when appropriate.
4. **Codex fixes failures that verification actually exposes.** Codex must identify the root cause, make the narrowest correct fix, add or correct deterministic regression coverage, rerun the narrow failing command, and then rerun the complete verification set until it passes or a genuine blocker is documented.
5. **Verification is not an excuse for speculative refactoring.** When all checks pass, Codex should record the results without manufacturing code changes. It must not redesign architecture, broaden scope, weaken tests, hide failures, or duplicate implementation work already completed by ChatGPT.
6. **ChatGPT reviews substantive Codex changes.** If Codex changes production behavior, public contracts, configuration, architecture, or significant test semantics, ChatGPT reviews the resulting diff before the next feature goal. Verification-only documentation changes or deterministic test synchronization fixes need only lightweight review.
7. **Small failure-driven tasks may go directly to Codex.** Known compiler errors, vet findings, race reports with useful stacks, deterministic test failures, build-script failures, formatting issues, and other tightly bounded defects may use Codex as the primary fixer without a separate ChatGPT implementation phase.
8. **Use committed handoff checkpoints.** Do not let ChatGPT and Codex modify the same files concurrently. The implementing agent commits and pushes before the verifying agent begins; the verifying agent also commits and pushes any fixes before handing control back.
9. **Preserve evidence.** Completion reports must distinguish tests that actually ran from integrations that were unavailable, and must record failures encountered, fixes made, final verification results, commit SHA, and push result.

The intended default flow is:

`ChatGPT goal/design/implementation/tests -> GitHub Actions baseline CI -> Codex independent verification and failure fixes -> ChatGPT review only when Codex made substantive changes`

## Resource constraints

- The primary development machine has approximately 1–2 GB of available memory.
- Run memory-intensive work sequentially by default, including code generation, builds, unit tests, integration tests, local dependency startup, runtime verification, and teardown.
- Do not run multiple memory-intensive subagents or verification commands concurrently.
- Prefer a single primary implementation agent. Use additional subagents only for lightweight independent review or analysis, and run them sequentially when memory pressure is possible.
- Use low-concurrency Go commands such as `go test -p 1 -parallel 1` unless a goal explicitly demonstrates that higher concurrency is safe on the active machine.
- Do not overlap Docker Compose startup or dependency verification with Go compilation, test execution, goctl generation, or multiple agent tasks.
- Stop unused local dependency containers after verification.
- If swap usage rises materially, the machine becomes unresponsive, or an out-of-memory condition is suspected, stop parallel work, preserve the current verified checkpoint, reduce concurrency, and resume sequentially.

## Change rules

- Keep generated go-zero files distinguishable from handwritten code.
- Every public API change must update the relevant API documentation or schema.
- Every database structure change must update `server/database/schema`.
- Add tests for reusable foundation capabilities and module-level business rules.
- Prefer small, reviewable changes over broad speculative abstractions.
