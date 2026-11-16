# Goal0062 security repair — ChatGPT-owned development checkpoint

Date: 2026-11-16. Source changes after Codex FAIL at frozen product `2067ca63441cb1264e57a5014fcb4014d310bb32`.

## Confirmed original defect

G62-N01/P2: A verified WeChat code login can resolve its old AccountID after a pre-unlink Redis scan, then create a new Redis session while the InnoDB unlink DELETE is waiting on a row lock. The login's bearer remains valid after unlink HTTP200 in the original implementation. The independent FAIL report is preserved in `0062-independent-acceptance.md`.

## Same-class concerns addressed in the repair candidate

1. **External session provenance:** A WeChat-issued Session now persists the HMAC-digested verified provider subject and provider type, never the raw OpenID. AuthN validates that provider mapping against *current MySQL* on both access and refresh. It rechecks after issuing and revokes on failure. Unbound sessions and DB errors fail closed; platform password sessions do not depend on a WeChat binding.
2. **Session revocation scan race:** Redis account-wide revoke now first increments a monotonically increasing per-account epoch. Session issuance records the epoch; Redis Get and Rotate check it. Scan misses / old keys cannot restore access. This also fences prior sessions across account disable and later reenable.
3. **Pre/post mutation fences:** WeChat unlink/relink revokes before the mapping mutation and again after successful DB cutover, closing the narrow window of sessions issued while the old mapping remains committed. Redundant revocation is intentional; a post-cutover Redis failure is an error requiring recovery/recheck rather than a false success.
4. **Password reset/change race:** A password session records a SHA-256 fingerprint of the *exact Argon2id credential hash verified*, never a plaintext password. The live credential fingerprint is checked on access and refresh; the issuer rechecks before returning a login result. An earlier password verification cannot authorize a session after a concurrent password reset. Administrative reset fences sessions on both sides of the DB update.
5. **Disable/enable safety:** Failure to fence sessions is no longer swallowed on disable; re-enable refuses to restore an account while Redis revocation is unavailable.
6. **Provider uniqueness:** The binding service requires the exact unique provider/account index and exact provider/subject primary key before enabling third-party login.

## Non-goals and rollout boundary

- No email OTP, domain, SMS, account merge, new user-visible login form or real WeChat-device qualification in this repair.
- Redis epoch keys intentionally remain without TTL while prior sessions can exist. Account-wide revoke may intentionally terminate password and third-party sessions of the affected AccountID.
- The live Kubernetes `awesome-zero-game-test` namespace, its database and persistent protection keys are **not** changed by this source repair; an additive schema preflight and separate safe migration remain a prerequisite to candidate deployment.
- A rolling deployment must not mix legacy API instances issuing untagged WeChat sessions with the repaired API. Explicitly drain/replace old API pods and invalidate legacy sessions before relying on the new provenance invariant.
- Existing independent FAIL stays historical. Codex must rerun the **original MySQL/InnoDB + Redis concurrent unlink test**, independently inspect relink and password reset/disable cases, and only then continue Goal0062 acceptance. CI SUCCESS alone is not a PASS.
- Account registration remains test-gated and email ownership remains unverified, as the user explicitly requested.
