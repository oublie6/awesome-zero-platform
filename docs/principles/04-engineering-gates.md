# 04 — Mandatory Design, Implementation and Verification Gates

## Before designing

1. Read `AGENTS.md`, `docs/goals/current.md`, this principles index and all relevant principle pages. Inspect the **current implementation and contracts** before proposing a change; documentation intent is not proof of existing behavior.
2. State the module owner, invariants, constraints, interfaces, compatibility/migration plan, security/privacy implications, failure recovery, resource bounds and measurable acceptance criteria.
3. Explicitly distinguish (a) already implemented/verified, (b) planned under the active Goal, and (c) future design requiring a new Goal. A principle must not silently override a Goal's authorized scope.
4. For contentious or expensive decisions record alternatives and rejected options. Prefer the smallest solution that preserves invariant boundaries and has demonstrable consumers.

## Implementation gates

- No domain dependency on transport, infrastructure or client SDK; no cross-module repository/table access. Composition roots wire infrastructure dependencies.
- Use go-zero maintained public facilities by default where behavior is equivalent; document native exceptions.
- Every state-changing operation specifies identity, version, exact idempotency and replay semantics, timeout/deadline and error classification. Commit-before-ack where B-level recovery is promised.
- No unbounded goroutines, queues, caches, timeouts or asset growth. Define ownership, lifecycle shutdown and resource release. Backpressure must isolate slow socket consumers from unrelated game hands.
- Public and private state must be separated. Redact access tokens, HPKE/reveal material, other players' cards and recovery secrets from routine logs and telemetry.
- Pass correlation `requestId / commandId / roomId / handId / stateVersion` through tracing where applicable without including secrets or high-cardinality IDs as unbounded metric labels.
- Review deployment and rollback implications. No production upload, data migration, destructive operation or account use without explicit task authorization.

## Mandatory ChatGPT → Codex independent acceptance workflow

- **ChatGPT owns development:** design against these principles, implement production changes and regression tests, run all available verification, self-review, fix defects, then commit and push an identifiable checkpoint to `main`.
- **GitHub Actions is the automatic verification gate**, not a substitute for independent review. Record the exact commit and actual CI outcomes.
- **Codex owns independent acceptance:** pull the exact checkpoint on clean `main`, independently inspect requirements/contracts/diff, rerun the relevant acceptance suite even when ChatGPT already ran it, and verify actual builds, integration, runtime, performance, security and failure/recovery cases required by the Goal. Report `PASS`, `FAIL` or `BLOCKED` with commands, environment, artifacts and reproducible evidence.
- **ChatGPT owns fixes:** Codex does not silently modify production code or expand requirements; hand defects back. ChatGPT fixes, commits and pushes; Codex reruns affected and regression checks. Repeat as needed.
- **Acceptance cannot be self-declared:** a Goal is complete only after all applicable required checks, CI and Codex independent acceptance are green for the same source revision. If hardware/credentials/platforms prevent verification, document the specific blocker; do not call it a pass.
- **Controlled handoff:** one writer at a time on `main`, no feature branches or concurrent competing changes; include commit SHA, acceptance checklist, applicable principle pages, reproducible commands, and expected evidence. See `AGENTS.md` for the normative full process.

## Proof and acceptance

- CI: narrow unit/contract/typecheck/import-boundary tests; schema/API compatibility; builds per qualified target; race/benchmark/fault tests where supported. Tests should fail for the relevant regression, not simply reassert a mock.
- Performance: measure P50/P95/P99, allocations, heap, lock/block contention, GC, queue depth and database fsync/transaction latency under realistic concurrency. Budgets are hypotheses until measured on stated hardware.
- Recovery: inject pre/post COMMIT, response loss, process kill, snapshot truncation, ownership race, reconnect replay and wrong-epoch commits; assert acknowledged state preserved exactly and unrecoverable conflicts quarantined.
- Client: record Source SHA + built artifact hash + Creator version + emulator/device + supported platform SDK and result. Distinguish code checks, simulated execution, developer-tool simulation, physical-device reports and server-side E2E.
- Evidence must reflect actual executed tests. Do not call a build a runtime pass, claim all platforms when only WeChat worked, claim durable recovery when only in-memory reconnect passed, or mark a Goal completed with required gates unverified.
- Second-pass adversarial self-review, actual diff inspection and regression re-runs are mandatory per `AGENTS.md`. Record unverified gaps openly.

## Design review checklist (copy into goals when relevant)

- [ ] Owner/data boundaries and dependency direction verified
- [ ] Current-versus-target implementation distinguished
- [ ] Protocol versions, authentication, idempotency and authority specified
- [ ] Crash, network partition, timeout, duplicate and rollback handled
- [ ] Performance budget and measurement approach documented
- [ ] Lifecycle/cleanup/backpressure specified
- [ ] Privacy, encryption, secret storage and leakage risks addressed
- [ ] Cross-platform compatibility maintained without requiring simultaneous target qualification
- [ ] Tests, release artifact identity, rollback and operational evidence defined
