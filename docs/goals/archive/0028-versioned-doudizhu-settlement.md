# Goal 0028: Versioned Doudizhu Multipliers, Settlement, and Normal Completion

## Status

- State: completed
- Started: 2026-07-29
- Completed: 2026-07-29
- Blockers: None.

## Goal

Complete a normal Doudizhu hand after Goal 0027 detects the winning seat. Define deterministic non-monetary integer scoring, bomb/rocket and spring multipliers, produce a zero-sum settlement, atomically return a completed final payload from the winning play, and use the existing `gamecore.LiveDirectory` retry-safe archive path without replaying the winning command.

## Delivered

1. Added `doudizhu-settlement-v1` as a non-monetary integer-points ruleset.
2. Used the winning bid `1`, `2`, or `3` as the base score.
3. Added independent doubling for every accepted bomb and for the accepted rocket.
4. Added spring when the landlord wins before either farmer makes an accepted play.
5. Added anti-spring when a farmer wins after the landlord made exactly one accepted play.
6. Kept voluntary player doubling outside v1.
7. Added deterministic zero-sum settlement:
   - landlord win: landlord `+2 × finalStake`, each farmer `-finalStake`;
   - farmer win: landlord `-2 × finalStake`, each farmer `+finalStake`.
8. Added strict input verification for seats, bid score, completed playing snapshot, ordered action history, recomputed patterns, reused physical cards, bomb/rocket bounds, and arithmetic overflow.
9. Added `playing.State.Clone` and changed winning-play handling to evaluate cards, gameplay completion, settlement, result JSON, and final JSON on cloned state before committing authoritative memory.
10. Added `doudizhu-live-completed-v1` containing:
    - setup artifact and digest;
    - canonical fairness transcript and digest;
    - complete bidding snapshot;
    - complete accepted play/pass history;
    - final physical hands and landlord cards;
    - landlord, winning bid, winner, multiplier breakdown, and seat points.
11. Changed a winning play to return a terminal `gamecore.CommandOutcome`; `LiveDirectory.Apply` archives the completed record before removing the live hand.
12. Preserved retry safety:
    - archive failure retains one immutable pending final record;
    - repeated gameplay commands are rejected while finalization is pending;
    - `RetryArchive` resubmits the identical digest;
    - the winning play and settlement are not replayed.
13. Kept all non-winning play and pass commands memory-only.
14. Added production coordinator verification for terminal/non-terminal result contracts.
15. Added `docs/architecture/doudizhu-settlement-v1.md` and linked it from the architecture overview.

## Constraints Preserved

- All edits and commits went directly to `main`; no branch or pull request was created.
- Points are not money, balances, prizes, recharge value, withdrawable value, or transferable assets.
- Active gameplay remains process-memory authority until final archive success.
- `gamecore` remained game-agnostic and required no semantic change.
- Goal 0023 deterministic shuffle/deal/transcript bytes remain unchanged.
- Goal 0026 bidding and Goal 0027 play command semantics remain compatible.

## Acceptance Results

- Landlord and farmer wins produce exact zero-sum points.
- Bomb, rocket, spring, and anti-spring multipliers are deterministic and independently tested.
- Invalid or incomplete snapshots, forged patterns, reused physical cards, and overflow are rejected.
- A winning play returns `Terminal=true`, a completed command result, and one immutable final payload.
- Successful archive removes the live hand exactly once.
- Failed archive leaves the exact pending record; retry preserves the digest and does not replay the winning command.
- Non-winning play and pass make zero archive calls.
- Production runtime rejects inconsistent terminal/non-terminal adapter results.
- Full repository, real MySQL/Redis, build, Compose, and production protocol verification passed twice.

## Completion Report

### Implementation commits

- Goal boundary: `e9f19c259267ce4494a9e3a1aa2832706ef774f8`.
- Settlement calculator and validation: `4b182d73a9e08edc506aa5603446a7089c84d5f1`, `ea763b0b7fe639470ce8f99a73555065002f25f1`.
- Settlement tests: `22526d2706f91d5c21c0d4ef69cbb56567e6443e`.
- Atomic winning-play completion and completed archive: `5c651229b3f094fc431ec68e6110e746f558be08`.
- Completed archive retry tests: `f1e8672103997896188d21e7d7eb45ef5009b324`.
- Production result verification: `0e98c9493ebafb2015b5bd97a2edf25e843e99ed`, `3dee94ee8bf66de7ceb0062d14ba17b9ac055746`.
- Settlement architecture: `01886e7c33b5ccdf007ef225efbec79ce215a31b`, `549c743b705a26cd259c201955f7e937f4c12abe`.

### Focused verification

- `30381913170` — settlement formatting, ordinary tests, race, and vet succeeded.
- `30382339505` — atomic completion, completed archive, retry safety, ordinary tests, race, and vet succeeded.
- `30382765499` — all Doudizhu domain/application/runtime tests, integration-tag MySQL compilation, race, and vet succeeded.

### Final verification

Two complete final-main runs succeeded:

- `30382926374`;
- `30383098801`.

They verified:

- Go 1.25.8 module cleanliness, generated-code repeatability, and formatting;
- all Goal 0028 focused tests, race tests, and vet checks;
- all Go tests and Security/Admin race tests;
- server build and local/production Compose validation;
- Secure Envelope and signed-manifest TypeScript/Go interoperability;
- Cocos deterministic-randomness policy and Admin Web build;
- full MySQL 5.7 and Redis integration;
- production container startup, HTTP, authenticated WebSocket, HTTPS, WSS, administrator bootstrap/login, and cleanup.

The final `goal0028/final-full` commit status is `success` on commit `750b4b6cc18785c51adad399bfcbd1877481a0cf`.

### Cleanup

All temporary Goal 0028 scripts and focused workflows were removed after successful verification. The temporary final verifier was removed by `1d4cba3351967da3a3b15cb243ccd078b0051968`, preventing further triggers.

### Deferred to Goal 0029 and later

- player-facing retrieval and independent verification of final evidence;
- all-pass replacement-hand orchestration;
- explicit cancellation and expiration policies;
- public HTTP/WSS gameplay transport and durable network-command replay cache;
- Cocos gameplay and settlement screens;
- active-hand persistence, crash restoration, or cross-instance migration.
