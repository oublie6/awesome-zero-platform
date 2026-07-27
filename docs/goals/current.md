# Current Goal: Idle

## Status

- State: idle
- Started:
- Completed:
- Blockers: None.

## Current State

Goal 0014 completed the reusable realtime WebSocket transport and HTTPS/WSS edge. Its implementation checkpoint and completed-goal checkpoint already passed the repository `ci/full` gate.

The previously prepared Goal 0015 for broad independent Codex verification was withdrawn before execution because it conflicted with the repository's corrected ownership model.

No executable goal is currently active.

## Collaboration Baseline

- ChatGPT owns requirement clarification, architecture, implementation, production code, test code, documentation, all testing available in its environment, failure fixes, final verification, commits, and pushes.
- GitHub Actions performs automatic repository and runtime verification after commits.
- Codex is not a routine second implementation or full-verification agent.
- Codex may be used only for a concrete supplementary test that ChatGPT cannot execute because of an environment, platform, device, browser, resource, credential, or tooling limitation.
- Before any Codex handoff, the active goal must document the exact unavailable test, why ChatGPT could not run it, the command or procedure, and the evidence expected.
- Codex should report the supplementary result only. ChatGPT owns diagnosis, fixes, regression tests, complete re-verification, commit, and push.
- Codex must not edit production code or test semantics unless the user explicitly authorizes that expanded role.

## Verification Status

- Goal 0014 implementation checkpoint: `207db9eba2abae7ae46eb249a7e7775ca1db95db`.
- Goal 0014 completed checkpoint: `40b6a6bea332fa58cbbc318ef10ffdaf35603e8f`.
- GitHub Actions run `30233535644` reported `ci/full: success` for the completed checkpoint.
- No Codex supplementary test is currently required or scheduled.

## Next Goal

The next feature or platform goal must be designed, implemented, tested, and verified by ChatGPT first. A Codex testing step may be added only if ChatGPT records a specific remaining test that its environment cannot perform.

## Completion Report

The incorrectly scoped independent Codex verification goal was withdrawn without execution or source changes. Repository collaboration rules now make ChatGPT responsible for development and testing, with Codex limited to explicitly documented supplementary tests unavailable to ChatGPT.
