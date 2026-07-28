# Goal 0023: Deterministic Card, Shuffle, Deal, and Fairness Transcript

## Status

- State: completed
- Started: 2026-07-28
- Completed: 2026-07-28
- Blockers: None.

## Goal

Define and implement the versioned, engine-independent Fair Doudizhu card/deck foundation as pure Go modules: canonical 54-card encoding, a deterministic cryptographic random stream, unbiased Fisher–Yates shuffle, immutable three-player dealing results, and a reconstruction-oriented fairness transcript that can independently verify the server commitment, accepted client contributions, beacon context, final deck, and deal.

## Scope

ChatGPT owned architecture, implementation, tests, failure diagnosis and fixes, documentation, repository verification, commits, and pushes directly to `main`.

Delivered:

1. A stable Card/Deck v1 representation for the 54-card Doudizhu deck, including 52 suited cards, small joker, big joker, canonical IDs, rank/suit accessors, stable text codes, parsing, validation, and the canonical unshuffled deck order.
2. Explicit algorithm-version constants for card encoding, seed derivation, deterministic random stream, shuffle, deal, and fairness transcript.
3. A canonical binary encoding for shuffle inputs that binds the hand ID, server seed, three accepted client contribution digests in seat order, and the locked public-beacon provider, round, and digest.
4. A versioned server-seed commitment function that binds the server seed to the hand ID and can be independently checked after disclosure.
5. A deterministic HMAC-SHA256 counter random stream derived from the canonical shuffle input.
6. Unbiased bounded-integer sampling using rejection sampling rather than modulo-only sampling.
7. Fisher–Yates shuffle over the canonical deck using one accepted unbiased bounded choice for every position from `53` down to `1`.
8. A deterministic deal contract with 17 rounds in seat order `1 -> 2 -> 3`, followed by the final three landlord cards.
9. Immutable-by-copy accessors, deck/deal validation, exact deck reconstruction, and order-binding SHA-256 deck and deal digests.
10. A versioned post-hand fairness transcript containing algorithm versions, server evidence, seat-ordered contribution evidence, beacon audit context, reveal-key audit metadata, derived shuffle seed, complete shuffled deck, deal, and all integrity digests.
11. A fail-closed transcript verifier that recomputes commitments, canonical seed input, shuffle seed, random stream, shuffled deck, deal, and all digests.
12. A committed golden vector suitable for future TypeScript/Cocos and third-party verifier implementations.
13. Exact architecture documentation for card IDs, byte encodings, random-stream construction, rejection sampling, Fisher–Yates, deal order, transcript encoding, privacy boundaries, and deferred integration.
14. Focused deterministic, tamper, property-style, compatibility, race, vet, and full-repository verification.

The following remain intentionally deferred:

- fetching or cryptographically validating a public-beacon provider proof; Card/Deck v1 consumes a value already verified by an adapter;
- changing the existing Commit-Reveal command flow, contribution storage, reveal-key lifecycle, application persistence, or MySQL schema;
- integrating server-seed commitment generation into the current `HandSetupProvider` and recording a completed deal in the Hand aggregate;
- persistence or HTTP/WSS publication of completed transcripts;
- landlord selection, bidding, doubling, play-pattern recognition, turn comparison, gameplay state, scoring, spring rules, settlement, replay UI, and Cocos gameplay screens;
- TypeScript/Cocos production verifier code, which must reproduce the committed golden vector in a later client-focused goal.

## References

- `AGENTS.md`
- `docs/requirements/fair-doudizhu-v1.md`
- `docs/architecture/fair-doudizhu-domain.md`
- `docs/architecture/fair-doudizhu-application-persistence.md`
- `docs/architecture/reveal-key-lifecycle-v1.md`
- `docs/architecture/fair-doudizhu-card-shuffle-v1.md`
- `docs/api/fair-doudizhu-protocol-v1.md`
- `server/business/doudizhu/domain/carddeck`
- `server/business/doudizhu/domain/carddeck/testdata/golden-v1.json`
- `.github/workflows/doudizhu.yml`

## Acceptance Results

- The canonical deck contains exactly 54 unique cards. IDs `0..51` are rank-major over ranks `3..2` and suits `C,D,H,S`; IDs `52` and `53` are `XJ` and `YJ`.
- Every card round-trips through canonical ID and code; standard cards expose rank and suit, jokers expose their joker rank and reject suit access, and malformed aliases are rejected.
- Server commitments bind both hand ID and the 32-byte server seed.
- Shuffle-input bytes use explicit domain separators, length-prefixed UTF-8, fixed seat order, fixed-size digests, and big-endian integers. Tests prove that changing any bound field changes the canonical input.
- The HMAC-SHA256 counter stream has a fixed first-64-byte golden vector and does not depend on time, process-global state, floating point, map order, platform endianness, or `math/rand`.
- `Uniform(bound)` rejects zero bounds, returns only values below the bound, and has a scripted test proving that an out-of-range sample below the rejection threshold is discarded.
- Fisher–Yates always yields a valid 54-card permutation. Property-style tests cover 128 deterministic inputs and the golden vector fixes the complete final deck and deck digest.
- Deal v1 gives each seat exactly 17 cards in documented round-robin order, retains exactly three landlord cards, preserves dealing order, and reconstructs the exact shuffled deck.
- Public deck slice access returns a copy; deal accessors return arrays by value. No mutable slice is shared with stored results.
- Deck and deal digests change when card order changes.
- Transcript construction validates every required identifier, fixed-size value, server commitment, client commitment, beacon context, reveal-key audit context, deck, deal, and version.
- Transcript verification reconstructs all derived evidence and rejects tampering of versions, hand ID, server seed, server commitment, every contribution or client commitment, beacon provider/round/digest/proof reference, reveal-key ID/hash, shuffle seed, deck/order, deal/order, and stored digests.
- A cross-package test proves Card/Deck v1 client commitments are byte-for-byte compatible with `domain.ComputeClientCommitment`.
- The production `carddeck` package imports only the Go standard library and remains independent of HTTP, WSS, Cocos, TypeScript, MySQL, Redis, go-zero, and beacon-provider adapters.
- The golden fixture fixes all inputs, the first 64 random-stream bytes, complete shuffled deck, three hands, landlord cards, and shuffle/deck/deal/transcript digests.

## Verification Evidence

### Commits

- Goal definition: `bf3cbd8f74bbb93f4d4aafab58c39e058b7df72f`.
- Goal start: `74caf0b55df3a0d15b0ec86bd7e1742c2ca5b7af`.
- Card/Deck error and encoding foundation: `b25210a2c16d39395b17df184b30cdd9be523e08`, `69e526578a55d8b120992a4751a85dc24a9b482d`.
- Card and canonical deck: `7b6719068b3bf99e0781a444580f2a381539a10a`.
- Shuffle input, deterministic stream, and rejection sampling: `adec6bdf559581c54e01c06198cb09905ebcf177`.
- Fisher–Yates shuffle: `35d3fe19242e018ac57d26f8a3dc5cebd54a1697`.
- Deal result and digests: `2dfbf3f03487080dce40b7734c9bde9ea27ef896`.
- Fairness transcript and verifier: `595c22ae399db268785cd39f967becc26dded192`.
- Commitment compatibility and comprehensive tests: `9e6d985404d7fcd8076f1b8c85f0d065e1ba134e`, `7d7f7ae41a5221b30b22fc43143d70aa9d9645da`.
- Golden vector: `9a6a62eac4e84ad5f0bd85776dba30905ec319e1`.
- Card/Deck v1 specification: `0ea820859e15f1d271575a123d8f160cc073033f`.
- Exact Go 1.25.8 formatter output: `8b70b212efb7ab8475c73f77cdfcb15cc9480cae`.

### Actual failures and fixes

- Initial focused Fair Doudizhu verification run `30341723878` passed application unit/race/vet, signed-manifest client/interoperability, and real MySQL 5.7 integration.
- Initial full CI run `30341723751` failed only at repository formatting. Module cleanliness and generated-code repeatability had already passed. The CI-produced gofmt artifact was inspected rather than guessing.
- Temporary formatter run `30341957539` used repository Go `1.25.8`, applied exact `gofmt` output, reran the focused Card/Deck package test, and pushed formatter commit `8b70b212efb7ab8475c73f77cdfcb15cc9480cae`. The temporary formatter workflow was then deleted.

### Final verification

- Final clean main CI run `30342057452`: success.
  - `go mod tidy` cleanliness;
  - goctl generated-code repeatability;
  - repository formatting;
  - all Go unit tests;
  - Security/Admin race tests;
  - Go builds;
  - local and production Compose validation;
  - production HTTPS switch validation;
  - Secure Envelope TypeScript tests and TypeScript-to-Go HPKE interoperability;
  - Admin Web typecheck and build;
  - real MySQL 5.7 and Redis integration.
- Final clean Fair Doudizhu run `30342060323`: success.
  - domain, application, reveal-key, Card/Deck and transcript unit tests;
  - `-race`;
  - `go vet`;
  - signed reveal-key manifest TypeScript tests and Go-to-TypeScript interoperability;
  - real MySQL 5.7 Doudizhu integration.
- Production Compose runtime acceptance run `30342370234`: success.
  - production containers and dependency startup;
  - HTTP and authenticated WebSocket behavior;
  - HTTPS and WSS behavior;
  - administrator bootstrap and login;
  - graceful runtime acceptance and cleanup.
- The runtime-verification workflow was deleted after success. A repeated Fair Doudizhu run `30342370239` also succeeded while runtime acceptance was executing.

## Working State

### Completed

- Goal 0022 was archived and fully cleaned up before Goal 0023 began.
- Repository inspection confirmed no duplicate Card/Deck, shuffle, deal, or transcript implementation existed.
- Card/Deck v1, deterministic random stream, unbiased shuffle, deal result, transcript, verifier, compatibility tests, golden vectors, and architecture specification are implemented.
- Focused and full verification succeeded after applying the one real formatter correction reported by CI.

### In progress

- None.

### Remaining

- None within Goal 0023.

## Completion Report

Goal 0023 is complete. The repository now has a deterministic, independently specified and test-vector-backed Fair Doudizhu Card/Deck v1 foundation. An independent implementation can reconstruct the same shuffle seed, random stream, 54-card permutation, three dealt hands, landlord cards, deck/deal digests, and transcript digest from the disclosed post-hand evidence.

The next stage should integrate these pure rules into the Hand/application boundary: generate and retain a real server seed behind the existing server commitment, consume an adapter-verified beacon value after all reveals, create and persist an immutable deal result, advance `DEALING -> BIDDING`, and store a publishable transcript only for terminal hands. Bidding, gameplay patterns, scoring, HTTP/WSS delivery, and Cocos UI should remain separate goals.
