# Goal 0041: modular extension guardrails and future microservice readiness

## Status

- State: completed
- Started: 2026-10-30
- Completed: 2026-10-30
- Blockers: None for the scoped module boundaries and extension conventions.

## Goal

Keep the general-purpose backend as a modular monolith while preparing safe module onboarding and later extraction without speculative empty RPC services.

## Deliverables

1. Strengthened `server/architecture/boundaries_test.go` to forbid any platform package SQL reads of Doudizhu/gamecore-owned tables.
2. Limited the `gamecore` shared-kernel exception to actual Doudizhu business imports, rather than granting every future business a game-specific dependency.
3. Documented module ownership and explicit `bootstrap.moduleSpec` registration, go-zero HTTP adapter/route contract updates, lifecycle/rollback, reusable persistence adapters and cross-module public port discipline.
4. Kept existing module composition, schema, runtime processes, K8s replica behavior and business state unchanged.

## Verification

- Code checkpoint: `f29a53c8331420a63d9ec2f712fdb58bb4068d21`.
- Full main CI (unit, race, MySQL/Redis integration, code generation, formatting, production Compose runtime) **success**: https://github.com/oublie6/awesome-zero-platform/actions/runs/41021397036
- Fair Doudizhu regression **success**: https://github.com/oublie6/awesome-zero-platform/actions/runs/41021396898
- Changes were implemented and self-reviewed by ChatGPT. GitHub Actions independently executed repository checks. No Kubernetes server work was performed by ChatGPT.

## Mandatory Self-Review

Checked compile-time import direction, SQL owner naming, module boundaries, startup/teardown and gamecore exception, and verified the new architecture checks in the main test suite. Did not add unused module registries, generic RPC services, new databases or distributed locks.

## Remaining / Important Scope Boundary

A new genuinely different business module is still needed to demonstrate onboarding ease; future microservice extraction requires explicit deployment contracts, independent state ownership, idempotency, message delivery and migration/rollback. This goal does not claim distributed state recovery or multi-Pod realtime games are implemented.

## Completion Report

Concrete boundary tests and onboarding documentation completed and passed main CI. No additional deployed service was created.
