# Goal 0033: Server v1 Release Baseline

## Status

- State: in_progress
- Started: 2026-07-29
- Completed: Not yet.
- Blockers: None.

## Goal

Turn the completed maintainable server v1 into an accurate, operator- and client-developer-facing release baseline. Remove stale capability descriptions, provide one reproducible local three-player integration path, document the production enablement boundary, and leave the repository with a clear verified server-v1 release anchor.

## References

- `AGENTS.md`
- `README.md`
- `server/README.md`
- `docs/api/authentication.md`
- `docs/api/fair-doudizhu-protocol-v1.md`
- `docs/architecture/doudizhu-realtime-v1.md`
- `docs/operations/realtime-websocket.md`
- `docs/operations/production-deployment.md`
- `server/apps/app-api/etc/main-api.yaml`
- `server/apps/app-api/etc/production.yaml`

## Deliverables

1. Update the repository README so its capability and deferred-work sections match the completed server v1.
2. Update the server README with a concise server-v1 overview, local startup path, Doudizhu enablement requirements, and documentation map.
3. Add one release-baseline operations guide covering local dependency startup, administrator bootstrap, three-account preparation, authentication, HTTP/WSS integration, verification, shutdown, and known v1 boundaries.
4. Update production deployment documentation with the optional secrets and configuration required to enable Fair Doudizhu safely.
5. Verify Markdown references, repository cleanliness, formatting, tests, builds, Compose configuration, integration, and production runtime acceptance through the existing CI baseline.
6. Record the final verified commit as the server-v1 release anchor; create no speculative v2 implementation.

## Constraints

- Follow `AGENTS.md` and commit directly to `main`.
- Do not change public API or business behavior in this documentation-focused goal.
- Do not claim active-hand crash restoration, cross-instance game migration, public matchmaking, spectator support, or value-bearing features.
- Do not expose or commit real passwords, access-token secrets, contribution keys, beacon proof secrets, reveal private keys, or TLS private keys.
- Prefer links to authoritative protocol documents over duplicating every command schema.
- Do not modify archived goals.

## Acceptance Criteria

- The root README no longer says completed Doudizhu transport, rules, play, settlement, or evidence capabilities are deferred.
- A new developer can identify the exact commands and documents needed to start the server and integrate three authenticated players.
- Doudizhu remains disabled by default, and the documentation lists every sensitive value required before enabling it.
- HTTP and WSS responsibilities, authentication, actor derivation, private-state delivery, reconnect behavior, and terminal evidence are described consistently with the implementation.
- Production documentation distinguishes the single-process in-memory active-hand boundary from durable MySQL/Redis state.
- All existing repository verification remains green.
- The completion report records the final verified release-anchor commit and any intentionally deferred work.

## Agent Strategy

The primary agent owns documentation changes, consistency review, integration, and final verification. No subagent is required unless a specific unavailable verification gap appears.

## Working State

### Completed

- Server Goals 0001–0032, including authenticated HTTP and WSS Doudizhu transport, private synchronization, terminal evidence, and full runtime verification.

### In progress

- Release documentation and repository-entrypoint cleanup.

### Remaining

- Documentation updates, final verification, completion report, archive, and release-anchor record.

### Verification status

- Not started for Goal 0033.

## Completion Report

Pending.
