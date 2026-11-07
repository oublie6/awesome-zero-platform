# Doudizhu — server-managed fairness v1

## Decision and trust model

Normal new Cocos hands use `fairnessMode = "server-managed-v1"`. The server
generates three separate 32-byte random contributions with a CSPRNG, associates
them with the three fixed account/seat identities and encrypts them with the
existing authenticated contribution-record protector. The players **do not**
generate, back up or submit fairness randomness in this mode.

This explicitly changes the promise of the old client commit/reveal protocol:
**the server knows the three managed contributions before the public beacon is
released.** They must never be presented as independently generated player
entropy. The mode protects against after-the-fact silent deal changes via
versioned commitments, a pinned future independently verifiable drand Quicknet
beacon, deterministic replay and archived final evidence. It does not prove
that the server is unable to abort games selectively or that a player's account
credential alone provides a cryptographic randomness proof.

The player's ordinary authenticated account credential is used for **access
control**, not as a shuffle seed or an encryption key. Seated participants can
retrieve their authorized hand views and final evidence via existing API
methods; the existing final-record verifier checks the deterministic transcript.
A credential grants an authorized view of proof data, not independent proof by
itself.

## Exact new-hand wire contract

```json
{"type":"room.hand.start","payload":{"handId":"hand-...","fairnessMode":"server-managed-v1"}}
```

Omitting `fairnessMode` intentionally continues the historical client-supplied
commit/reveal path; its hand snapshot JSON and command-payload hash remain
backward compatible. Unknown modes fail closed. Client-side requests bind
the mode in the same persistent command identity, preventing changing the
custody mode on an exact-ID retry. Mode is immutable per hand, persisted in
the hand's authoritative snapshot and genesis, surfaced in authenticated
fairness views and final evidence, and checked during durable live recovery.

## Durable lifecycle

1. Owner sends a valid authenticated and version-checked room-start command
   when all three seats are ready. Existing client command serialization,
   dedupe and unknown-outcome reconciliation remain authoritative.
2. The server prepares a committed seed and a fixed **future drand Quicknet**
   round. Managed mode refuses any HMAC/test-beacon fallback.
3. Within the same `room.hand.start` SQL transaction, the server generates
   each seat's CSPRNG material, computes the existing versioned contribution
   digest and hand/seat-bound commitment and applies three commits.
4. The same transaction applies three verified reveals, encrypts each
   contribution in a distinct AES-GCM record with bound hand, seat, actor,
   source command and digest AAD, and persists the resulting
   `WAITING_PUBLIC_BEACON` hand, room update, outbox and original receipt.
   Neither randomness nor phrase plaintext appears in the hand snapshot,
   outbox, HTTP view or ordinary logs.
5. The existing automatic verified-beacon dealer advances
   `WAITING_PUBLIC_BEACON → DEALING → BIDDING` when the locked public
   round becomes available. Existing gameplay/journal/checkpoint/terminal
   archive and credential-authorized final evidence are unchanged.

A failed seal, record insert or SQL update rolls back the entire new-hand
transaction. The existing seed preparation compensation releases an
unacknowledged seed; accepted command replay does not generate new random
contributions. The client may reconnect without any locally held randomness
and must never fabricate a replacement for an already committed hand.

## Client migration

- New `GameOnlineController.startHand` requests the managed mode.
- The room screen shows automatic server-custody/beacon progress and omits
  manual kit generation/import/commit/reveal controls for this mode.
- Pre-upgrade client-contributed hands still show the legacy controls and
  support their existing recovery protocol. Do not reinterpret their material
  or clear unknown requests.
- A completed hand's evidence is retrieved by an authenticated original
  participant; another account cannot access the same private evidence merely
  by knowing the hand ID.

## Remaining qualification

This mode requires independent acceptance at a fixed source SHA with real
MySQL, HTTPS/WSS, three authenticated identities, actual drand proof, service
restart, lost/duplicated start ACK, replay/recovery and terminal verification.
Green source CI alone does not imply a qualified production release.
