# Fair Doudizhu Card, Shuffle, Deal, and Transcript v1

## 1. Boundary

This document fixes the engine-independent Card/Deck v1 algorithms used to reconstruct a Fair Doudizhu hand after verification material is disclosed. The implementation lives in:

```text
server/business/doudizhu/domain/carddeck
```

The package is pure Go and imports only the standard library. It does not fetch beacon values, decrypt contributions, persist snapshots, expose HTTP/WSS routes, select the landlord, validate plays, score a game, or render a Cocos client.

All algorithms are explicitly versioned. A verifier must reject unknown versions rather than guessing compatible behavior.

## 2. Primitive encodings

Unless a section says otherwise:

- text is UTF-8 without leading or trailing whitespace;
- text fields are limited to 128 bytes;
- `U16BE(n)` is an unsigned 16-bit big-endian integer;
- `U32BE(n)` is an unsigned 32-bit big-endian integer;
- `U64BE(n)` is an unsigned 64-bit big-endian integer;
- `LP(value)` is `U32BE(len(value)) || value`;
- a domain separator is `UTF8(domain) || 0x00`;
- every digest, seed, commitment, and public-key hash is exactly 32 bytes;
- fixed arrays are encoded in their documented order, never by map iteration or implementation-dependent serialization.

## 3. Card v1

Version:

```text
fair-doudizhu-card-v1
```

A card is one unsigned byte in the range `0..53`.

### 3.1 Standard cards

IDs `0..51` are rank-major. Rank order is:

```text
3, 4, 5, 6, 7, 8, 9, 10, J, Q, K, A, 2
```

Suit order inside each rank is:

```text
C = clubs
D = diamonds
H = hearts
S = spades
```

For a standard card:

```text
cardId = rankIndex * 4 + suitIndex
```

The canonical deck therefore starts:

```text
C3, D3, H3, S3, C4, D4, H4, S4, ...
```

and ends its standard cards with:

```text
C2, D2, H2, S2
```

### 3.2 Jokers

```text
52 = XJ = small joker
53 = YJ = big joker
```

`XJ` and `YJ` are used because they cannot collide with a suited card code such as `SJ` for the jack of spades.

Card codes are uppercase and canonical. A parser rejects aliases, surrounding whitespace, lowercase forms, zero-padded ranks, and unknown suits or ranks.

## 4. Deck v1

Version:

```text
fair-doudizhu-deck-v1
```

The canonical unshuffled deck is the 54-byte sequence:

```text
0, 1, 2, ... 53
```

A valid deck contains every valid card exactly once. Order is significant.

The Deck v1 digest is:

```text
SHA-256(
  UTF8("fair-doudizhu/deck-digest/v1") || 0x00 ||
  LP(UTF8("fair-doudizhu-deck-v1")) ||
  U16BE(54) ||
  cardId[0] || ... || cardId[53]
)
```

## 5. Commitments used by the transcript

### 5.1 Server seed commitment v1

A server seed is exactly 32 bytes from a cryptographically secure random source. The versioned commitment is:

```text
SHA-256(
  UTF8("fair-doudizhu/server-commit/v1") || 0x00 ||
  LP(UTF8(handId)) ||
  serverSeed[32]
)
```

Binding the hand ID prevents the same disclosed seed and commitment from being reused as valid evidence for another hand.

The HandSetup provider must eventually use this function when Card/Deck rules are integrated into the application command flow. Goal 0023 defines and verifies the rule but does not change the current setup provider or persistence contract.

### 5.2 Client commitment compatibility

The transcript recomputes the existing domain commitment without changing its bytes:

```text
SHA-256(
  UTF8("fair-doudizhu/client-commit/v1") || 0x00 ||
  LP(UTF8(handId)) ||
  U8(seat) ||
  contributionDigest[32]
)
```

Seats are encoded in fixed order `1, 2, 3`. A cross-package test compares this implementation against `domain.ComputeClientCommitment`.

## 6. Shuffle seed v1

Version:

```text
fair-doudizhu-shuffle-seed-v1
```

Inputs are:

- hand ID;
- disclosed 32-byte server seed;
- accepted contribution digest for seat 1;
- accepted contribution digest for seat 2;
- accepted contribution digest for seat 3;
- locked beacon provider;
- locked beacon round;
- verified 32-byte beacon digest.

The canonical bytes are:

```text
UTF8("fair-doudizhu/shuffle-seed/v1") || 0x00 ||
LP(UTF8("fair-doudizhu-shuffle-seed-v1")) ||
LP(UTF8(handId)) ||
serverSeed[32] ||
U8(1) || seat1ContributionDigest[32] ||
U8(2) || seat2ContributionDigest[32] ||
U8(3) || seat3ContributionDigest[32] ||
LP(UTF8(beaconProvider)) ||
LP(UTF8(beaconRound)) ||
beaconDigest[32]
```

The shuffle seed is:

```text
SHA-256(canonicalShuffleInput)
```

The beacon proof itself is validated by a provider adapter outside this package. The transcript binds its opaque proof reference for audit correlation.

## 7. Deterministic random stream v1

Version:

```text
fair-doudizhu-hmac-counter-v1
```

The shuffle seed is the HMAC-SHA256 key. Counter blocks start at counter `0`:

```text
block(counter) = HMAC-SHA256(
  key = shuffleSeed,
  message =
    UTF8("fair-doudizhu/random-block/v1") || 0x00 ||
    LP(UTF8("fair-doudizhu-hmac-counter-v1")) ||
    U64BE(counter)
)
```

Blocks are concatenated. A `uint64` sample consumes the next eight bytes in stream order and interprets them as unsigned big-endian.

No wall-clock time, process-global state, platform endianness, floating point, or `math/rand` affects the stream.

## 8. Unbiased bounded sampling

To sample uniformly from `[0, bound)` for non-zero `bound`:

```text
threshold = (2^64 mod bound)
repeat:
  candidate = nextUint64()
until candidate >= threshold
return candidate mod bound
```

The Go implementation calculates the threshold with unsigned wraparound:

```text
(uint64(0) - bound) % bound
```

Values below the threshold are discarded. Applying `% bound` without this rejection step is forbidden because it introduces modulo bias whenever `bound` does not divide `2^64`.

## 9. Fisher–Yates shuffle v1

Version:

```text
fair-doudizhu-fisher-yates-v1
```

Start with the canonical Deck v1. For `i` from `53` down to `1`:

```text
j = Uniform(i + 1)
swap(deck[i], deck[j])
```

The implementation performs exactly one accepted bounded choice per position. Rejection sampling may consume more than one raw `uint64` candidate for a choice.

## 10. Deal v1

Version:

```text
fair-doudizhu-deal-v1
```

Deal the first 51 shuffled cards over 17 rounds. Each round assigns cards in seat order:

```text
seat 1 -> seat 2 -> seat 3
```

For zero-based `round` and `seatIndex`:

```text
hands[seatIndex][round] = deck[round * 3 + seatIndex]
```

The final cards `deck[51]`, `deck[52]`, and `deck[53]` are the three landlord cards.

Deal v1:

- preserves dealing order;
- does not sort a hand;
- does not assign a landlord;
- can reconstruct the exact shuffled deck by interleaving the three hands and appending the landlord cards.

The Deal v1 digest is:

```text
SHA-256(
  UTF8("fair-doudizhu/deal-digest/v1") || 0x00 ||
  LP(UTF8("fair-doudizhu-deal-v1")) ||
  U8(1) || U16BE(17) || seat1Cards[17] ||
  U8(2) || U16BE(17) || seat2Cards[17] ||
  U8(3) || U16BE(17) || seat3Cards[17] ||
  U16BE(3) || landlordCards[3]
)
```

## 11. Fairness transcript v1

Version:

```text
fair-doudizhu-fairness-transcript-v1
```

A post-hand transcript contains:

- all seven algorithm-version strings;
- hand ID;
- disclosed server seed and server commitment;
- seat-ordered contribution digests and client commitments;
- beacon provider, round, digest, and proof reference;
- reveal-key ID and X25519 public-key SHA-256 for audit correlation;
- derived shuffle-seed digest;
- complete shuffled deck;
- deck digest;
- the three dealt hands and landlord cards;
- deal digest;
- transcript digest.

The canonical transcript bytes are, in order:

```text
UTF8("fair-doudizhu/transcript-canonical/v1") || 0x00 ||
LP(cardVersion) ||
LP(deckVersion) ||
LP(seedVersion) ||
LP(randomStreamVersion) ||
LP(shuffleVersion) ||
LP(dealVersion) ||
LP(transcriptVersion) ||
LP(handId) ||
serverSeed[32] ||
serverCommitment[32] ||
for seat 1..3:
  U8(seat) || contributionDigest[32] || clientCommitment[32] ||
LP(beaconProvider) ||
LP(beaconRound) ||
beaconDigest[32] ||
LP(beaconProofRef) ||
LP(revealKeyId) ||
revealPublicKeySha256[32] ||
shuffleSeedDigest[32] ||
U16BE(54) || shuffledDeck[54] ||
U8(1) || U16BE(17) || seat1Cards[17] ||
U8(2) || U16BE(17) || seat2Cards[17] ||
U8(3) || U16BE(17) || seat3Cards[17] ||
U16BE(3) || landlordCards[3] ||
deckDigest[32] ||
dealDigest[32]
```

The transcript digest is:

```text
SHA-256(
  UTF8("fair-doudizhu/transcript-digest/v1") || 0x00 ||
  LP(canonicalTranscriptBytes)
)
```

Verification recomputes every commitment, canonical shuffle input, shuffle seed, random stream, deck, deal, deck digest, deal digest, canonical transcript bytes, and transcript digest. Unknown versions or any mismatch fail closed.

## 12. Golden vector

The cross-language fixture is committed at:

```text
server/business/doudizhu/domain/carddeck/testdata/golden-v1.json
```

It fixes:

- all input bytes and commitments;
- the first 64 random-stream bytes;
- the complete shuffled deck as stable card codes;
- all three hands and landlord cards;
- shuffle-seed, deck, deal, and transcript digests.

Future TypeScript, Cocos, Web/H5, Android, iOS, and third-party verifiers must reproduce this vector before claiming v1 compatibility.

## 13. Privacy and publication

This transcript is post-hand verification evidence. Goal 0023 does not publish it through HTTP or WSS and does not add it to active-hand events or logs.

The transcript deliberately omits:

- original phrases;
- clients' original 32-byte random values;
- private encryption keys;
- database-protection keys;
- access tokens or session data.

It includes accepted contribution digests because those are sufficient to verify the commitments and deterministic shuffle input without exposing the raw contribution plaintext.

## 14. Deferred integration

Later goals will decide:

- when and how the Hand aggregate records a deal and advances `DEALING -> BIDDING`;
- persistence and publication of completed or terminal-hand transcripts;
- beacon-provider adapters and proof formats;
- TypeScript/Cocos verifier implementation;
- landlord selection, bidding, play patterns, turns, scoring, settlement, replay, and UI.

Those integrations must use the exact v1 algorithms and golden vector defined here or introduce a new explicit version.
