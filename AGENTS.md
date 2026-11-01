# AGENTS.md

## Project intent

Awesome Zero Platform is a general-purpose, highly extensible Go/go-zero application backend platform, not a Doudizhu-only backend or a game-only framework. Its primary product is reusable platform capabilities that can host independently evolving business modules and integrate external systems. Fair Doudizhu is the first real business implementation and an end-to-end validation case, not a reason to place game-specific semantics in shared platform APIs or infrastructure.

Build a maintainable **modular monolith first**. Preserve clear ownership and contracts so a module can be extracted into a separately deployed service when measurable scale, reliability, team ownership, or release independence justifies it. Microservices, dynamic plugins, and distributed infrastructure are evolution options rather than default architecture.

## Mandatory architecture principles — design-time compliance

**Every design, coding task, review, refactor, architecture proposal, Goal definition and Codex handoff MUST comply with the canonical principles in `docs/principles/`.** This is not optional guidance. Read [the index](docs/principles/README.md) and every relevant principle page before choosing a design, changing an interface, or writing implementation code:

- [01 Modularity and evolution](docs/principles/01-modularity-and-evolution.md): ownership, dependency direction, modular monolith, go-zero first, extraction readiness.
- [02 Game runtime and durable recovery](docs/principles/02-game-runtime-and-recovery.md): memory-authoritative execution, MySQL durable log/commit-before-ack B-level recovery target, Redis as auxiliary, snapshots, replay, epoch fencing, performance.
- [03 Client portability](docs/principles/03-client-portability.md): WeChat-first qualification, future cross-platform battles on the same backend, unified AccountID, platform-neutral business core, security and fairness.
- [04 Engineering gates](docs/principles/04-engineering-gates.md): design checklist, performance budgets, fault tests, resource/secret hygiene and honest evidence.

**Before implementing a design**, write down its owner, public contract, data authority, module dependencies, operational and failure semantics, performance cost, security constraints, migration/rollback approach and test evidence. Explicitly distinguish **implemented facts** from **accepted target architecture** and future work. A design violating these principles must be revised; if the active Goal or a published contract conflicts, **stop and document the conflict for an explicit Goal/contract amendment** rather than silently expanding scope or changing production behavior.

**Binding decisions:** Go/go-zero modular monolith first; Domain free of infrastructure/platform SDKs; isolated module-owned data; memory execution with a target of durable MySQL log-before-ack recovery for live hands (not currently implemented); Redis for routing/presence/discardable cache, not per-turn authority; same authoritative Go game and stable internal identities across all clients; WeChat Mini Game qualifies first, future targets share pure TS protocol/game code; strict versions, idempotency, bounded resources, crash/fencing tests and performance measurement.

This directory is the normative source for these decisions. Keep `AGENTS.md` compact by linking detailed requirements there. Existing goal-specific acceptance and historical evidence remain intact; future Goals should link relevant principles and prove their compliance.

## go-zero first infrastructure standard

- **go-zero First:** go-zero is the primary application infrastructure framework. Prefer its maintained, public, version-compatible components over directly invoking parallel native clients or implementing duplicate infrastructure when go-zero already provides an equivalent capability. This includes SQL access (sqlx), Redis/cache, HTTP routing and middleware, configuration, logging, telemetry, and other applicable capabilities.
- **Use go-zero in application call paths, not just during initialization.** When an operation has an equivalent go-zero API, make that API the default in adapters, repositories, and services; do not keep normal business calls on a native client merely because both clients are initialized.
- **Native clients may coexist** for operations with no equivalent go-zero API, where exact semantics matter (e.g. Redis WATCH and long-lived Pub/Sub, MySQL driver or complex cursor/transaction operations). Constrain these exceptions to infrastructure adapters, explain the reason, and use the same configuration, tracing, lifecycle, and error-handling conventions where feasible.
- **Extend rather than replace:** If a requirement exceeds go-zero's supported API, first compose or extend the go-zero component behind a narrowly scoped adapter; avoid building a second competing framework or large generic abstraction.
- **Compatibility means verified behavior, not only matching package names.** Preserve timeouts, connection pools, Redis DB selection, transaction isolation and atomicity, lock ordering, TTL precision, error identity, cancellation, shutdown, observability, security, and existing business contracts. Never silently weaken correctness just to eliminate native APIs.
- **Tracing:** propagate request context to go-zero clients and verify SQL/Redis child spans appear in actual traces. Do not assume that Prometheus metrics or a go-zero wrapper automatically instruments raw driver calls, WebSocket messages, background tasks, or bypass paths.
- **For each new component or upgrade,** check the selected go-zero release's source/API and document where native integration remains necessary. Default future modules to these standard components without requiring business modules to configure their own pools or tracing.

## Architecture rules

- Start as a modular monolith; extract services only when real scaling or ownership needs appear.
- `server/apps` contains runnable processes.
- `server/platform` contains reusable platform capabilities shared by different products.
- `server/business` contains product-specific business modules and should be created only when real business implementation begins.
- `server/foundation` contains reusable technical infrastructure without business semantics.
- Platform and business modules must not access another module's database tables or repository implementation directly.
- Cross-module calls must use explicit public interfaces or events.
- Keep transport, application logic, and persistence concerns separated.
- Prefer the existing Kubernetes base and explicit Kustomize observability overlay for new platform observability deployments. Do not require Docker Compose or a publicly exposed log dashboard for Kubernetes environments; preserve existing application deployment baselines unless a goal explicitly changes them.
- Do not create generic dumping grounds such as `common`, `utils`, or `helpers`.
- The repository stores the current complete database schema, not incremental migration history during the early development phase.
- Temporary database upgrade SQL must not be committed.

## General-purpose platform and extensibility principles

- **Platform first; business modules second.** `foundation` owns technical primitives; `platform` owns product-neutral, reusable application capabilities (identity, authentication, authorization, administration, realtime transport, and other features justified by real consumers); `business/<module>` owns its domain, use cases, persistence adapters, and business-specific integration. Neither `foundation` nor `platform` may depend on a concrete business module. `server/apps` is a composition root, not a home for business rules.
- **No game-centric platform assumptions.** Doudizhu, `gamecore`, and Cocos belong to the game business/client context. `gamecore` may contain reusable *game* semantics, but must not become a mandatory platform foundation for unrelated domains. Adding a non-game business must not require importing game packages or modifying Doudizhu logic.
- **Strong module boundaries.** Modules expose intentionally small, documented public interfaces, commands, queries, and/or versioned events. No module imports another module's internal implementation, persistence adapter, table, or private state. Prefer Go compile-time boundaries, package `internal` restrictions where appropriate, and dependency-boundary tests over folder naming alone.
- **Module-owned data.** Modules can share a MySQL instance in the monolith, but each owns its schema objects, repositories, invariants, and writes. A consumer must not query or join another module's tables directly; obtain data through the owner's public contract or an explicitly designed read model. Avoid assuming a cross-module database transaction is portable to a future service split. Keep the present single-schema deployment mechanism until a real upgrade requirement justifies changing it, but record ownership and plan versioned migrations before production deployments need in-place upgrades.
- **Simple in-process composition now.** Register and wire real modules explicitly at startup, support controlled enable/disable and lifecycle shutdown where required, and keep dependencies visible. Do not introduce reflection-driven discovery, Go binary plugins, dynamic code loading, a universal module SDK, or a framework-scale registry on speculation. If a second business exposes repeated wiring problems, extract the smallest necessary registration/lifecycle contract.
- **Separate two extension paths.** A business module running *inside* this backend is different from an external system integrating *with* it. In-process modules use typed interfaces; external integrations use explicit, versioned HTTP/API, WebSocket, webhook, messaging, or identity-federation adapters as real use cases demand. Do not require external systems to share Go code, database schema, or the platform's internal account model.
- **Design public integration contracts deliberately.** For each external API or event, specify versioning, authenticated identity and authorization scope, tenant/resource isolation when actually required, pagination and errors where applicable, idempotency/retry semantics, timeouts, correlation IDs, security boundaries, and backward-compatibility/deprecation policy. Never expose internal database records or business implementation types as accidental public contracts.
- **Asynchronous integration is demand-driven.** Begin with direct in-process calls for synchronous module coordination. Add domain events, transactional outbox relays, queues, or RPC only when a concrete reliability, fan-out, ownership, or independent-scaling requirement exists. Persisting an outbox row alone is not proof that delivery, retries, consumer idempotency, or dead-letter handling works.
- **Keep clients independent.** The server exposes stable application contracts rather than depending on Vue, Cocos, WeChat, Douyin, Android, or iOS. Different client shells share appropriate protocol and security packages while platform SDK differences stay behind client-side adapters.
- **Do not confuse extensibility with feature breadth.** Do not prebuild multi-tenancy, workflow engines, plugin marketplaces, generic low-code systems, service meshes, or universal data models without a specific supported use case and acceptance test. Prefer the smallest abstraction validated by a second genuinely different business module.

## Monolith-to-microservice evolution criteria

- Prefer modularization, profiling, capacity tuning, bounded resource use, and ordinary process scaling before introducing service boundaries.
- Consider extracting a module when independent scaling, failure isolation, security boundary, operational ownership, deployment cadence, or technology requirements provide a demonstrated benefit. User count alone does not mandate microservices.
- Before extraction, confirm an explicit contract, data ownership, independent startup and health lifecycle, authentication/authorization propagation, timeout/retry/idempotency semantics, observability, compatible versioning, and a safe migration/rollback path.
- Replacing a local interface call with RPC does not automatically solve distributed state, cross-node routing, sagas, or active-instance recovery. Preserve each business module's explicit consistency and state-authority model.
- Never split deployment units merely to claim microservice readiness; keep `apps` as the only composition/deployment boundary until the new service has an actual owner and operational benefit.

## Architecture checks for every goal

- State whether the change is reusable technical infrastructure, reusable platform capability, concrete business behavior, client integration, or external-system adapter; place it in the corresponding module.
- Check that a new business can use platform capabilities without changing an unrelated business module or introducing business-specific symbols into `foundation`/`platform`.
- Check import direction, module-owned data access, exposed contract compatibility, access-control scoping, failure isolation, startup/shutdown behavior, and whether a future service extraction would require hidden cross-module database access.
- Add or update narrow architecture/dependency tests when introducing a new cross-module boundary. Treat `gamecore` import tests as game-specific, not a substitute for platform-wide dependency checks.
- Preserve backwards compatibility and real verification evidence; do not refactor a working subsystem or introduce distributed components solely for theoretical extensibility.

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
3. **Always read `docs/principles/README.md` and the relevant principle pages**, then read architecture, requirement, and decision documents explicitly referenced by the current goal, plus source files needed for implementation.
4. Confirm the working tree is clean and the checked-out branch is `main`. Preserve any pre-existing local work and stop if the repository is not clean.
5. Synchronize `main` with `origin/main` using `git pull --ff-only`. If synchronization cannot complete cleanly, stop and document the blocker.
6. Inspect the synchronized repository state and relevant Git diff.
7. Treat the current goal and referenced documents as authoritative.

Execution rules:

- Do not load archived goals unless the current goal explicitly requires historical investigation.
- Do not expand, reinterpret, or silently replace the goal, deliverables, constraints, or acceptance criteria.
- ChatGPT, as the primary agent, owns architecture, implementation, test design, execution of every test available in its environment, integration, failure fixes, and final verification.
- Subagents may be used for independent analysis or review. **Codex is the mandatory independent acceptance executor** after ChatGPT finishes and pushes a development checkpoint; Codex intentionally reruns relevant checks and tests full Goal acceptance, while ChatGPT retains implementation and defect-fix ownership.
- Codex may update only the status, working-state, verification-status, and completion-report sections of `docs/goals/current.md` unless the goal explicitly permits other changes. It independently reviews the complete Goal and reports PASS/FAIL/BLOCKED.
- Set the goal state to `in_progress` when implementation begins.
- Set the goal state to `completed` only after every acceptance criterion passes, including mandatory independent Codex acceptance at the verified source SHA.
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

## ChatGPT–Codex collaboration workflow — mandatory independent acceptance

This workflow is **mandatory for every executable development Goal** unless the user explicitly changes it. The roles are distinct: **ChatGPT designs, develops, tests and fixes; Codex independently accepts and reports.** Codex verification is required even when ChatGPT's tests and GitHub Actions pass.

1. **ChatGPT owns design and implementation.** Inspect the current code and applicable principles, define a bounded Goal and measurable acceptance criteria, implement code and tests, run all available checks, perform adversarial self-review, fix own findings, and commit/push a coherent development checkpoint to `main`.
2. **GitHub Actions verifies automatically.** Record CI build, lint, unit/integration/contract checks, dependency boundaries and any supported runtime/race gates. Do not treat green CI as Codex acceptance.
3. **Codex independently verifies the complete Goal.** After a development checkpoint is pushed, Codex pulls the exact SHA on clean synchronized `main` and checks requirements against source and behavior, executes the full relevant acceptance suite (including tests ChatGPT ran), and performs real integration/runtime, performance and fault-injection validation when applicable and available. Independent rerunning is **intended**, not prohibited duplication.
4. **Codex is an acceptance reviewer, not the default implementer.** Codex records test environment, commands, artifact/source SHA, observed outputs, defects, blockers, unverified checks and an explicit `PASS / FAIL / BLOCKED` result. It must not silently repair production code, redesign architecture, widen the Goal or declare missing tests passed. Source fixes belong to ChatGPT unless the user explicitly delegates otherwise.
5. **Defect loop:** Codex reports issues → ChatGPT investigates, changes implementation/tests and performs available re-verification → ChatGPT commits/pushes `main` → Codex independently reruns affected tests and the applicable regression gates. Repeat until acceptance passes or a genuine blocker remains.
6. **Completion is evidence-based.** A Goal may become `completed` only after required ChatGPT verification, CI gates and Codex independent acceptance pass at the **same accepted source SHA**, or when a documented criterion is explicitly amended by the user. Record real-device and environment limitations without inventing success.
7. **Main-only sequential coordination.** No feature branch, PR or concurrent edit of the same files; ChatGPT hands off a pushed immutable commit SHA and acceptance checklist. Codex may edit only Goal working state, verification status and completion report in `docs/goals/current.md` unless expressly authorized otherwise.
8. **Handoff must be actionable.** Specify exact commit SHA, the Goal requirements and principles, commands and environments for validation, performance/recovery failure cases when applicable, expected evidence and reporting location. ChatGPT must not promise that Codex ran tests until actual evidence is available.

Default sequence:

`ChatGPT design → ChatGPT implementation and available tests → ChatGPT self-review / fixes → commit and push main → GitHub Actions → Codex independent full acceptance → [FAIL: ChatGPT fixes + commit, Codex rechecks] → PASS and Goal completion`

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

## Mandatory self-review for primary-agent changes

When the same agent owns both design and implementation, it must perform a distinct second-pass review before declaring the goal complete. This is required even when all automated tests pass.

Self-review rules:

- Treat the implementation as untrusted during review. Re-read the active goal, referenced requirements, decisions, architecture rules, and public contracts, then look for ways the change could be wrong rather than arguments that justify the current design.
- Passing tests is necessary but not sufficient. Review requirement coverage, explicit exclusions, invariants, module ownership boundaries, and whether the implementation changed any behavior that the goal did not authorize.
- Review at least these risk classes when relevant: empty and boundary inputs; duplicate, stale, retry, timeout, cancellation, and partial-success paths; concurrency, ordering, idempotency, transaction, and lifecycle races; error propagation, cleanup, rollback, restart, and recovery; backward compatibility of APIs, configuration, database schema, generated contracts, persisted data, and frontend behavior; security, authorization, credentials, sensitive payloads, and path validation; observability accuracy; resource usage and performance regressions.
- Trace at least one successful flow and representative failure/recovery flows end to end across transport, application, module, persistence, and frontend layers affected by the change. For stateful or concurrent code, explicitly inspect state transitions and operation ordering rather than relying only on happy-path tests.
- Inspect the final diff line by line, challenge every new assumption, and check that tests would fail for the defect the change is intended to prevent. Tests written by the implementing agent must also be reviewed for false positives, missing assertions, and overfitting to the implementation.
- If self-review finds a defect or material gap, fix it, add or strengthen deterministic regression coverage, rerun focused and full verification, and repeat the self-review on the corrected diff.
- An independent subagent or Codex review may strengthen confidence but does not replace this mandatory self-review. When the same agent performed design, implementation, and review, the completion report must call it self-review and must not describe it as independent review.
- The completion report must summarize the self-review scope, important risks checked, findings and fixes, remaining uncertainties, and any frontend, dependency, integration, runtime, or environment checks that were not actually run.
