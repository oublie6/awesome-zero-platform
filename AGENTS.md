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

## Main branch workflow

- All project changes by ChatGPT, Codex, automation, and other repository agents must be made directly on `main` unless the user explicitly overrides this rule for a specific task.
- Do not create, switch to, or push feature branches, `agent/*` branches, verification branches, or routine pull requests.
- Before planning or editing, confirm the checked-out branch is `main`, the working tree is clean, and `main` is synchronized with `origin/main` using `git pull --ff-only`.
- If local work prevents a clean switch or synchronization, preserve it and stop with a documented blocker rather than creating another branch.
- Commit goal-related checkpoints directly to `main` and push with `git push origin main`. Never force push.
- ChatGPT–Codex handoffs use sequential commits on `main`; agents must not modify the same files concurrently.

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
4. Confirm the working tree is clean and the checked-out branch is `main`. Preserve any pre-existing local work and stop if the repository is not clean.
5. Synchronize `main` with `origin/main` using `git pull --ff-only`. If synchronization cannot complete cleanly, stop and document the blocker.
6. Inspect the synchronized repository state and relevant Git diff.
7. Treat the current goal and referenced documents as authoritative.

Execution rules:

- Do not load archived goals unless the current goal explicitly requires historical investigation.
- Do not expand, reinterpret, or silently replace the goal, deliverables, constraints, or acceptance criteria.
- ChatGPT, as the primary agent, owns architecture, implementation, test design, execution of every test available in its environment, integration, failure fixes, and final verification.
- Subagents may be used for independent analysis or review. Codex may be used only for specifically identified supplementary tests that ChatGPT cannot execute in its environment; it must not duplicate normal implementation or testing work.
- Codex may update only the status, working-state, verification-status, and completion-report sections of `docs/goals/current.md` unless the goal explicitly permits other documentation changes.
- Set the goal state to `in_progress` when implementation begins.
- Set the goal state to `completed` only after every acceptance criterion passes.
- Set the goal state to `blocked` only when a genuine blocker is recorded with evidence.
- When resuming after a long pause, context compaction, or substantial scope discussion, reread this file and the current goal before continuing.

Before finishing:

1. Run every acceptance check required by the current goal.
2. Update the permitted status and completion-report sections.
3. Inspect the final Git diff and ensure only goal-related changes are included.
4. Commit all completed goal changes directly on `main` with a concise, descriptive commit message.
5. Push with `git push origin main`. Do not use force push. If the push fails, document the exact blocker and keep the verified commit locally.
6. Summarize changed files, verification results, commit and push results, unresolved blockers, and intentionally deferred work.
7. Stop when the goal is completed and pushed, or when a genuine blocker is documented.

## ChatGPT–Codex collaboration workflow

Use the following as the default ownership model unless the user explicitly assigns different responsibilities for a specific task.

1. **ChatGPT owns development and testing.** ChatGPT clarifies requirements, defines the goal and acceptance criteria, designs architecture and contracts, changes production code, writes and maintains tests and documentation, executes every verification available in its environment, fixes failures, reviews the integrated diff, and commits and pushes completed work directly to `main`.
2. **GitHub Actions provides automatic repository verification.** CI should cover compilation, deterministic unit and package tests, repository regression, dependency boundaries, frontend builds, integration checks, runtime acceptance, and appropriate race checks where the CI environment supports them.
3. **Codex is only a supplementary test executor.** Codex may be invoked only after ChatGPT documents a concrete test that ChatGPT cannot perform because of an environment, platform, device, browser, resource, credential, or tooling limitation. The handoff must state the exact missing test, why ChatGPT could not run it, the command or procedure, and the evidence expected.
4. **Codex must stay within the documented gap.** Codex must not independently rerun the entire goal, duplicate tests ChatGPT already completed, add speculative stress variants, redesign architecture, implement features, edit production code, alter test semantics, or fix discovered defects unless the user explicitly authorizes that expanded role.
5. **Findings return to ChatGPT.** Codex records the supplementary test result and evidence. When Codex exposes a defect, ChatGPT owns root-cause analysis, the production or test fix, regression coverage, complete re-verification, commit, and push.
6. **No work is manufactured for handoff.** When ChatGPT and CI can execute all required acceptance checks, no Codex phase is required.
7. **Use committed handoff checkpoints on `main` only when a real supplementary gap exists.** ChatGPT commits and pushes the implementation and available-test checkpoint before Codex starts. Codex should normally produce only a verification report; any source change requires explicit user authorization and subsequent ChatGPT review.
8. **Preserve evidence.** Completion reports must distinguish tests run by ChatGPT, tests run by GitHub Actions, supplementary tests run by Codex, unavailable integrations, failures encountered, fixes made by ChatGPT, final verification results, commit SHA, and push result.

The intended default flow is:

`ChatGPT goal/design/implementation/tests/fixes on main -> GitHub Actions verification -> only when a specific unavailable test remains, Codex runs that exact supplementary test and reports evidence -> ChatGPT handles any resulting code or test changes`

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
