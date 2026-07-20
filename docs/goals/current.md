# Goal 0005: Establish identity account foundation

## Status

- State: ready
- Started:
- Completed:
- Blockers:

Supported states: `idle`, `ready`, `in_progress`, `completed`, `blocked`. New executable goals start as `ready`.

## Goal

Establish a reusable identity account module with account profiles, password credentials, account status, secure password hashing and verification, MySQL persistence, transactional consistency, and focused tests, without introducing public login, token, session, authorization, or product-specific user behavior.

## References

- `AGENTS.md`
- `docs/architecture/overview.md`
- `docs/requirements/infrastructure-foundation.md`
- `docs/requirements/identity-account-foundation.md`
- `server/README.md`
- `server/foundation/database`
- `server/database/schema/current.sql`
- `server/database/seed/development.sql`
- `server/apps/app-api/internal/svc`
- `Makefile`

## Deliverables

1. Create a focused identity platform module under `server/platform/identity`.
2. Define account, identity value, account status, and password-credential domain types with explicit validation.
3. Add deterministic MySQL definitions for identity accounts and password credentials to the current complete schema.
4. Enforce database-level uniqueness for non-empty usernames, email addresses, and phone numbers.
5. Implement identity-specific persistence interfaces and concrete MySQL persistence without creating a generic repository framework.
6. Implement secure password hashing and verification using a maintained algorithm and explicit bounded parameters suitable for the low-memory development environment.
7. Implement transactional account creation with an initial password credential.
8. Implement account lookup by ID, username, email, and phone.
9. Implement profile update, account enable, account disable, password verification, and password change operations.
10. Translate persistence and domain failures into focused safe identity errors without leaking SQL, driver, DSN, password, hash, or internal-address details.
11. Add focused unit and integration tests for validation, password handling, persistence, uniqueness, transactions, concurrency, and sensitive-data safety.
12. Update active documentation with identity ownership, account-state behavior, password policy, schema ownership, identifier policy, and deferred capabilities.
13. Do not expose public registration, login, account-management, token, session, or authorization HTTP endpoints in this goal.

## Constraints

- Follow `AGENTS.md` and every referenced document.
- Respect the 1–2 GB development-machine constraint and run generation, compilation, tests, Docker verification, password hashing work, and agent work sequentially.
- Use `go test -p 1 -parallel 1`.
- Do not run Docker-backed integration verification concurrently with compilation, unit tests, code generation, password-hashing tests, or subagents.
- Keep identity inside `server/platform/identity`; do not create `server/business`.
- Do not create a separate user module unless a concrete distinction is required by this goal.
- Do not create generic repositories, CRUD frameworks, active-record models, base models, dependency-injection frameworks, plugin systems, generic services, or database-driver abstractions.
- Use the existing MySQL database foundation and transaction runner.
- Do not introduce PostgreSQL support or a multi-database abstraction.
- Do not store, log, return, seed, or commit plaintext passwords or password hashes.
- Do not include a default administrator or fake account in development seed data.
- Do not add public registration, login, JWT, refresh tokens, Redis sessions, logout, password recovery, verification codes, CAPTCHA, OAuth, WeChat login, roles, permissions, Casbin, departments, organizations, tenants, menus, file upload, audit storage, notifications, frontend code, CI, Kubernetes, or production deployment resources.
- Do not create migration history or commit temporary upgrade SQL.
- Do not modify archived goals.
- Do not expand or reinterpret this goal without explicit user instruction.
- Codex may update only the Status, Working State, and Completion Report sections of this file unless a deliverable explicitly requires another documentation update.

## Acceptance Criteria

1. `make generate` succeeds and remains repeatable without unintended diffs.
2. `make fmt` succeeds.
3. `make test` succeeds serially.
4. The identity module exists only under the intended platform boundary and introduces no generic repository or CRUD infrastructure.
5. The current complete MySQL schema applies successfully to a clean database.
6. The schema contains deterministic identity account and password-credential tables using InnoDB and `utf8mb4`.
7. No public login, registration, account-management, token, session, or authorization endpoint is added.
8. Creating a valid account with an initial password commits both account and credential.
9. Failure while storing the credential rolls back the account insertion.
10. No credential can exist without its account.
11. At least one supported identity value is required.
12. Duplicate non-empty username, email, or phone values return a focused conflict error.
13. Concurrent attempts to create the same unique identity produce exactly one successful account.
14. Account lookup by ID, username, email, and phone behaves correctly, including not-found results.
15. Profile updates persist correctly.
16. Account enable and disable transitions behave correctly.
17. Password hashes never contain or expose the plaintext password.
18. Correct passwords verify successfully and incorrect passwords fail.
19. Changing a password makes the old password invalid and the new password valid.
20. Empty and oversized password inputs are rejected safely before expensive hashing.
21. Password hashing uses explicit documented parameters and remains suitable for the low-memory environment.
22. Identity errors and logs do not expose passwords, hashes, DSNs, SQL text, raw MySQL errors, or internal addresses.
23. Unit and integration tests remain isolated from pre-existing developer data.
24. Active documentation accurately describes module ownership, password policy, account status, schema behavior, identifier policy, and deferred capabilities.
25. Final Git inspection contains only Goal 0005 implementation, schema, tests, and necessary documentation changes.
26. All verified changes are committed and pushed to the configured upstream without force pushing.

## Agent Strategy

The primary agent owns identity boundaries, schema design, identifier choice, password policy, persistence, transaction consistency, integration, and final verification. Use at most one implementation subagent or one lightweight review subagent at a time because the development machine has approximately 1–2 GB of available memory. Do not run subagents concurrently with code generation, compilation, password hashing tests, or Docker-backed verification. Do not allow multiple agents to modify the same files concurrently.

## Execution Process

1. Synchronize according to `AGENTS.md`, read every reference completely, and inspect the current MySQL foundation, schema, service context, package conventions, and test setup.
2. Produce a concise implementation plan before editing, including module boundaries, identifier choice, schema constraints, normalization, password algorithm and parameters, transaction flow, error mapping, test strategy, and serial verification sequence.
3. Set the Goal state to `in_progress`.
4. Implement the identity domain and password capability without exposing public HTTP endpoints or introducing speculative abstractions.
5. Implement explicit MySQL persistence and transaction-backed account creation using the existing database foundation.
6. Update the complete schema and active documentation without creating migration history or seed accounts.
7. Add unit tests first, then run formatting and serial unit tests.
8. Start MySQL and Redis only after unit verification, apply the clean schema, and run integration tests serially.
9. Inspect errors and logs for sensitive-data leakage and inspect the repository for scope expansion or generic abstractions.
10. Update only the permitted Goal sections with concrete evidence, commit, and push without force pushing.
11. Stop only when every acceptance criterion passes or a genuine blocker is documented with evidence while preserving safe work.

## Working State

### Completed

- Goal 0004 archived and current Goal reset.
- Identity account foundation requirements defined.
- Goal 0005 execution contract prepared.

### In progress

- None.

### Remaining

- All implementation deliverables.

### Verification status

- Not started.

## Completion Report

Not started.
