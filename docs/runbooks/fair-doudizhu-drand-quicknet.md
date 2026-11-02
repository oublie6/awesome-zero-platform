# Production fairness: independently verified drand Quicknet (Goal 0046)

This service cannot be described as publicly fair while operating with the legacy `HMACBeaconProofVerifier`. The legacy scheme only authenticates a server-controlled value and is kept for isolated tests. A production release must enable the Quicknet BLS12-381 verifier, pinned to a *separately checked* public key.

## Required deployment configuration

Set these application-level values when enabling Doudizhu:

```yaml
Doudizhu:
  Enabled: true
  BeaconProvider: drand-quicknet
  BeaconRound: auto
  BeaconPublicKeyHex: "" # supply via APP_DOUDIZHU_DRAND_QUICKNET_PUBLIC_KEY_HEX
  # Supply contribution encryption key, reveal key manifest, HTTPS and WSS
  # endpoint configuration through existing secret-management controls.
```

`APP_DOUDIZHU_DRAND_QUICKNET_PUBLIC_KEY_HEX` must be the **96-byte public key** (192 hex characters) from the authenticated Quicknet chain-info record, independently verified before initial deployment. The chain identifier is fixed in code to `52db9ba70e0cc0f6eaf7803dd07447a1f5477735fd3f661792ba94600c84e971`. Its scheme is `bls-unchained-g1-rfc9380` (public key on G2, signature on G1). Do **not** retrieve and blindly trust a new signing public key from the same HTTP relay returning the beacon.

The service requests only the *committed* round from `https://api.drand.sh/v2/beacons/quicknet/rounds/{round}`. It locally validates its BLS signature with the pinned key and reconstructs `SHA256(signature)`. A failed signature, mismatched round or digest prevents locking. Pending/future rounds are retried; there is no HMAC fallback.

## Lifecycle and fairness deadlines

1. At hand creation the immutable domain snapshot fixes a future Quicknet round (strictly more than 60 seconds in the future). An untrusted client cannot select or replace it.
2. All three players must finish Commit/HPKE Reveal **before that round is publicly released**. Contributions after the publication deadline are rejected even if their command timestamps or signatures otherwise pass.
3. A bounded SQL scanner aborts incomplete expired Commit/Reveal hands. It will never substitute a different beacon or silently re-deal.
4. For a complete three-player reveal, a bounded background worker fetches and verifies the **exact** Quicknet round, persists the verified beacon through `LockHandBeacon` and then runs `MarkHandDealt`, using durable system-actor commands. A restarted server resumes WAITING_PUBLIC_BEACON and DEALING from MySQL, not a volatile queue.
5. No public client can supply a substitute unsigned beacon through this automated path.

## Release cautions

- Verify the actual Quicknet historical-signature fixture and injected integration test on the **same final SHA**. Use HTTPS egress to the official drand relay; network failure must fail closed.
- The 60-second pre-publication contribution window is a *hard protocol deadline*. Set user-visible countdowns and abort reasons before production launch; confirm that real three-device users can complete contribution in time.
- The signed drand result still requires the existing protected contribution records, secure reveal-key manifest verification and encrypted server seed continuity.
- This change does **not** establish full B-grade fault guarantees: bounded encrypted state checkpoints, actual process-kill tests, cross-node takeover, race tests and verified three-real-client gameplay are separate release gates.
- Performance-test quantiles are reported from the CI MySQL environment only, not a production SLA. External AppID, handset, approved HTTPS/WSS domains and credentials remain operator-owned.
