# Goal 0023: Deterministic Card, Shuffle, Deal, and Fairness Transcript

## Status

- State: ready
- Started: Not yet.
- Completed: Not yet.
- Blockers: None.

## Goal

Define and implement the versioned, engine-independent Fair Doudizhu card/deck foundation as pure Go modules: canonical 54-card encoding, a deterministic cryptographic random stream, unbiased Fisher–Yates shuffle, immutable three-player dealing results, and a reconstruction-oriented fairness transcript that can independently verify the server commitment, accepted client contributions, beacon context, final deck, and deal.

## Scope

ChatGPT owns architecture, implementation, tests, failure diagnosis and fixes, documentation, repository verification, commits, and pushes directly to `main`.

Deliver:

1. A stable Card/Deck v1 representation for the 54-card Doudizhu deck, including 52 suited cards, small joker, big joker, canonical IDs, rank/suit accessors, stable text codes, parsing, validation, and the canonical unshuffled deck order.
2. Explicit algorithm-version constants for card encoding, seed derivation, deterministic random stream, shuffle, deal, and fairness transcript.
3. A canonical binary encoding for shuffle inputs that binds the hand ID, server seed, three accepted client contribution digests in seat order, and the locked public-beacon provider, round, and digest.
4. A versioned server-seed commitment function that binds the server seed to the hand ID and can be independently checked after disclosure.
5. A deterministic HMAC-SHA256 counter random stream derived from the canonical shuffle input.
6. Unbiased bounded-integer sampling using rejection sampling; modulo-only sampling is forbidden.
7. An in-place and copy-returning Fisher–Yates shuffle over the canonical deck, using the deterministic stream and exactly one unbiased bounded choice for each position from `53` down to `1`.
8. A deterministic deal contract: 17 rounds in seat order `1 -> 2 -> 3`, followed by the final three cards as landlord cards. The deal preserves dealing order and does not assign a landlord or sort a player's cards.
9. Immutable-by-copy deal accessors, deck/deal validation, reconstruction helpers, and deterministic SHA-256 digests for the final deck and deal.
10. A versioned fairness transcript containing only post-hand verification material: hand and algorithm versions, server seed and commitment, three contribution digests and commitments, beacon context and proof reference, reveal-key audit metadata, derived shuffle-seed digest, final-deck digest, deal digest, and transcript digest.
11. A transcript verifier that recomputes commitments, seed derivation, shuffled deck, deal, and all digests; any modified signed input, version, commitment, card, order, or digest must fail verification.
12. Golden test vectors suitable for future TypeScript/Cocos and third-party verifiers.
13. Architecture and verification documentation describing exact byte encodings, deck order, shuffle/deal algorithms, privacy boundaries, and deferred gameplay rules.
14. Focused unit, property-style deterministic, race, vet, and available full-repository CI verification.

The following remain outside this goal:

- fetching or cryptographically validating a public-beacon provider proof; this goal consumes a beacon value already verified by an adapter;
- changing the existing Commit-Reveal command flow, encrypted contribution storage, reveal-key lifecycle, HTTP/WSS transport, MySQL schema, or application command persistence;
- landlord selection, bidding, doubling, legal play-pattern recognition, turn comparison, gameplay state, scoring, spring rules, settlement, replay UI, and Cocos gameplay screens;
- publishing an active hand's deck, private cards, server seed, raw phrase, or client random bytes;
- TypeScript/Cocos implementation of the verifier, which will use the committed golden vectors in a later client-focused goal.

## References

- `AGENTS.md`
- `docs/requirements/fair-doudizhu-v1.md`
- `docs/architecture/fair-doudizhu-domain.md`
- `docs/architecture/fair-doudizhu-application-persistence.md`
- `docs/architecture/reveal-key-lifecycle-v1.md`
- `docs/api/fair-doudizhu-protocol-v1.md`
- `server/business/doudizhu/domain`
- `server/business/doudizhu/application`
- `.github/workflows/doudizhu.yml`
- `docs/architecture/fair-doudizhu-card-shuffle-v1.md` (to be created)

## Constraints

- Follow `AGENTS.md` and keep all changes directly on `main`.
- The card/shuffle/deal/transcript implementation must be pure Go and use only the standard library.
- Keep the new rules under the Fair Doudizhu domain boundary; do not create a generic randomness, card-game, or transcript framework.
- Use explicit domain-separation strings, big-endian fixed-width integers, and length-prefixed UTF-8 fields in canonical encodings.
- All IDs, versions, provider names, rounds, proof references, card values, and fixed-size digests must be validated before computation.
- Use cryptographic hashes and HMAC only for determinism and integrity; no call to `math/rand`, global mutable entropy, wall-clock time, map iteration order, floating-point arithmetic, platform endianness, or implementation-dependent serialization may affect a result.
- Rejection sampling must be testable independently and prove that out-of-range samples are discarded without bias.
- The transcript must not contain raw phrases or the clients' original 32-byte random values. It may contain only their already-accepted contribution digests and commitments.
- The full deck, server seed, and transcript are post-hand verification evidence and must not be added to active-hand events, logs, or public transport in this goal.
- Do not modify the current MySQL schema or add migration files.
- Do not add HTTP, WSS, Cocos, or TypeScript production code.
- Run memory-intensive verification sequentially and use low-concurrency Go commands such as `go test -p 1 -parallel 1`.

## Acceptance Criteria

- The canonical deck contains exactly 54 unique valid cards and has a documented stable order.
- Every valid card round-trips through ID, rank/suit or joker classification, stable code, and parser; malformed or non-canonical values are rejected.
- The server commitment changes when the hand ID or server seed changes and is independently verifiable.
- Canonical shuffle-input bytes are deterministic and change when any hand ID, server seed, seat contribution, beacon provider, beacon round, or beacon digest changes.
- The deterministic random stream has committed golden vectors and yields identical bytes and integers across repeated runs.
- `Uniform(bound)` rejects zero bounds, returns only values below the bound, uses rejection sampling, and has a deterministic test proving that at least one scripted out-of-range sample is discarded.
- Fisher–Yates always produces a 54-card permutation with no missing or duplicate cards and has a committed golden final deck and digest for at least one complete transcript input.
- Repeating shuffle and deal with identical versioned inputs produces byte-for-byte identical results; modifying any input changes the verified transcript or causes verification failure.
- Deal v1 gives each of seats 1, 2, and 3 exactly 17 cards, leaves exactly three landlord cards, follows the documented round-robin order, and reconstructs the exact shuffled deck.
- Returned deck, hands, landlord cards, and transcript data cannot be mutated through shared backing storage.
- Deck and deal digests bind card order, not only card membership.
- Transcript construction rejects inconsistent contribution commitments, malformed cards, mismatched digests, unsupported versions, and missing required audit context.
- Transcript verification recomputes server commitment, client commitments, shuffle seed, shuffled deck, deal, deck digest, deal digest, canonical transcript bytes, and transcript digest.
- Tampering tests cover the server seed, hand ID, every seat contribution, client commitment, beacon context, algorithm version, reveal-key metadata, deck order, deal order, and stored digests.
- Cross-package tests prove the transcript's client-commitment calculation remains byte-for-byte compatible with the existing Doudizhu domain commitment contract.
- The module imports only the Go standard library and remains independent of HTTP, WSS, Cocos, database, Redis, go-zero, and beacon-provider adapters.
- Architecture documentation fixes every algorithm and canonical byte encoding sufficiently for an independent implementation.
- `go test -count=1 -p 1 -parallel 1 ./business/doudizhu/domain/...` succeeds.
- `go test -race -count=1 -p 1 -parallel 1 ./business/doudizhu/domain/...` succeeds.
- `go vet ./business/doudizhu/domain/...` succeeds.
- Existing generated-code, formatting, full Go unit, Secure Envelope, Admin Web, real MySQL 5.7/Redis integration, Compose, HTTP/WS, HTTPS/WSS, and production runtime verification remain green.
- Final evidence records exact commits, actual failures and fixes, CI run IDs, unavailable checks, and intentionally deferred work.

## Working State

### Completed

- Goal 0022 reveal-key lifecycle and signed manifests was completed, fully verified, archived, and cleaned up.
- Repository inspection found no existing Card/Deck v1 package, deterministic shuffle implementation, deal result, or fairness transcript implementation.
- The existing hand lifecycle, contribution commitment contract, beacon plan, application/persistence boundaries, and focused Fair Doudizhu CI were reviewed.

### In progress

- None.

### Remaining

- Implement the card model, canonical deck, deterministic stream, rejection sampling, shuffle, deal, transcript, golden vectors, tests, documentation, and full verification.

### Verification status

- Baseline main before Goal 0023 definition: `34ca67357b56db6d15d04214f1430534a73f895d` plus Goal 0022 cleanup commits through `8f53b446480ec910b3fda3a2bf2b8a3d418817d7`.
- Goal 0022 final main CI `30335660784`, Fair Doudizhu `30335660790`, and runtime acceptance `30335660783` succeeded.
- Goal 0023 implementation verification pending.

## Completion Report

Pending.
