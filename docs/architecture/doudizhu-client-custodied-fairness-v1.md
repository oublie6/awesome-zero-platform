# Player-contributed, server-custodied fairness v1

## Scope and authority

The default new three-player Doudizhu hand uses
`client-contributed-server-custodied-v1`, explicitly selected on
`room.hand.start`. The player enters a **personal phrase** and the client
generates **32 CSPRNG bytes**, which it encrypts together using the pinned
signed HPKE reveal key. The Go server decrypts the authenticated request,
computes a versioned contribution digest, and **atomically** stores the
encrypted original phrase/random record together with the commitment,
acknowledged request, aggregate update, actor/seat binding and Outbox events.

Only transient request processing state is held on the client. No gameplay
secrets, randomness, backup kits, current hand authority, command journal or
checkpoint is persisted on the client. The platform session's ordinary
authentication token lifecycle is separate from per-hand game persistence.
The phrase is the **player's verification input**, not a durable restore key.
The player may exit or switch devices, then authenticate and recover the same
server-owned hand state. An accepted-but-unanswered request must be
reconciled by original command identity; after client process death, an
authoritative read of the player's committed/revealed status determines
whether another submission is allowed.

Legacy hands with no `fairnessMode` retain their two-step Commit/Reveal
and historical manual-kit compatibility. No implicit migration of their
commitments, protected records, snapshots or final payloads is allowed.

## Command semantics

`room.hand.start` accepts
`{"handId":"...","fairnessMode":"client-contributed-server-custodied-v1"}`.
Room state, hand seed and **future drand Quicknet round** are durably
bound at creation. Server-only HMAC beacons are refused in this mode.

`hand.contribution` is a distinct authenticated durable command carrying
an HPKE envelope. The user phrase and 32-byte random are inside its
encrypted plaintext; the request must carry an independently server-MACed
`doudizhu-custodied-context-v1` ticket bound to actor, HandID, seat,
requestId, clientSeq, expectedVersion, HPKE key and issue/expiry times.
The different `fair-doudizhu/custodied-contribution-aad/v1` prevents
cross-mode replay. Each actor/seat is accepted at most once per hand,
until the committed future beacon deadline. There is no dependency on a
locally held secret for a second Reveal operation: the server knows the
material as soon as it accepts the one-step request. Three accepted
contributions move to `WAITING_PUBLIC_BEACON`, followed by the existing
verified drand auto-deal flow.

The plaintext format is the existing strictly decoded
`fair-doudizhu-reveal-v1` containing `secureRandom`, `phrase`,
`handId`, `seat`, `normalization:NFKC-v1`. The existing digest and
commitment functions remain unchanged; no second, unreviewed PRNG or shuffle
algorithm is introduced.

## Storage, proof and recovery

MySQL stores the full original contribution **only within a
record-bound AES-GCM ciphertext**. The associated data binds record ID,
HandID, seat, actor, source command and digest. Snapshot and Outbox contain
only commitments/digests and opaque protected record references. The game
runtime, durable journal, state snapshots, checkpoint and final archive are
server-owned and continue to recover independently of the client.

After a **completed and verified final archive**, a participating
authenticated account may read **only its own** originally submitted random
bytes, seat, commitment and contribution digest from a separate restricted
endpoint. No original phrase or other seats' contribution is returned.
The user re-enters their own remembered phrase and the client recomputes
the canonical commitment with the retrieved random and compares the
result with the archived contribution commitment. It may separately
inspect the verified final shuffle/game transcript and drand proof.
Nonparticipants, other players, incomplete hands, unknown modes,
missing historical keys and tampered protected records fail closed.

## Fairness promise (important limitation)

The server receives both the player-generated random and the user phrase
at submission. The server **can** know these values before the future
public beacon, unlike the old two-stage client-secret protocol. The new
claim is: deterministic shuffle inputs are bound before a future,
independently verifiable drand round; a user can recompute their own
contribution from the original phrase and returned random; and an archived
game can be replay-verified against stored inputs. This does not prove that
the server is unable to modify unseen commitments or selectively abort a
game. Independent historical commitment witnessing or external publication
would be needed for stronger non-equivocation claims. A short user phrase
alone is not cryptographic high entropy and is not treated as an auth key.

## Acceptance

Independent acceptance must check three distinct real authenticated
players, encrypted MySQL data, atomic commit/rollback, exact-ID retry and
unknown outcomes, restart/checkpoint recovery, fixed drand proof,
pre-terminal non-disclosure, participant vs outsider authorization,
local phrase match/mismatch, final archive mode/commitment consistency,
and legacy two-stage compatibility. Passing source CI alone is not
production acceptance.
