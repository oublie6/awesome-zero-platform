# Fair Doudizhu — administrator-controlled protection roots

**Current policy:** a stable operator-managed 256-bit protection root. The
service **does not auto-rotate** a root on startup, on a timer, or per hand.
This is an operational decision; it does **not** weaken the existing journal
integrity or private recovery-data protection.

## What is protected

| Record | Representation in MySQL | Protection |
| --- | --- | --- |
| Live journal (bid/play/pass, IDs and versions) | Ordinary structured fields | Root-derived, domain-separated HMAC-SHA256 authenticated v2 records |
| Protected server fairness seed | Authenticated ciphertext | Root-derived AES-GCM |
| Live recovery checkpoint / private recovery state | Authenticated ciphertext | A separate root-derived AES-GCM key |

The journal is **not encrypted wholesale**. It must not contain plaintext
private hands or uncommitted fairness secrets. Removing the HMAC would remove
an independent authentication layer for recovered actions; a bare SHA-256 of
a database row cannot provide equivalent tamper resistance when an attacker
can rewrite both the row and its checksum.

This root is not the WeChat AppSecret, token signing secret, reveal-key private
key, database password or the optional local beacon HMAC secret. Handle those
credentials with their own operational procedures.

## Normal operations: stable root, no scheduled replacement

1. An administrator generates a high-entropy 32-byte root, for example using
   a trusted system's cryptographic random-number generator. Do **not** put
   the value into a command argument, committed source file or screenshot.
2. Store the root and its recovery copy in an access-controlled external
   secret manager or offline secure backup, with documented restore authority.
   Back up the **key and database together** using separate protected stores.
3. Supply the same stable `APP_DOUDIZHU_CONTRIBUTION_KEY_ID` and
   `APP_DOUDIZHU_CONTRIBUTION_KEY_HEX` across all API replicas/restarts.
   The value must be exactly 64 lowercase hex characters (32 bytes).
4. Leave `APP_DOUDIZHU_PREVIOUS_KEYS_JSON` **unset** for a new deployment.
   Only an explicitly approved migration may supply retained read-only roots.
5. The application derives independent keys for journal HMAC, checkpoint
   AES-GCM, seed AES-GCM and other existing per-purpose protection. It never
   needs the operator to distribute separate per-hand keys.
6. Test an isolated restore with the **same** root before declaring a backup
   recoverable. Losing the root can make a perfectly intact database
   unrecoverable.

The Compose deployment forwards the root environment settings from a
controlled operator environment. For stronger isolation, use your platform's
secret delivery mechanism; environment variables can be visible to privileged
host/container administrators. Keep secrets out of logs and ordinary CLI
diagnostics.

## Read-only inventory (implemented)

Run from the `server` directory with a separately provisioned, **read-only**
database account whenever possible. Set `APP_MYSQL_DSN` through a protected
environment or secret injection, never as a CLI argument:

```bash
# No credentials or root values are printed.
go run ./apps/doudizhu-maintenance --mode key-audit
go run ./apps/doudizhu-maintenance --mode key-audit --hand <authorized-hand-id>
```

The output includes journal action counts by integrity version, protected seed
row count and checkpoint row count. It always says
`old_root_retirement=UNVERIFIED`, because a count alone **cannot prove**
which key authenticated a row, that all older records were migrated, or that a
cold recovery with only the new root is safe.

The command is read-only; `--execute` is forbidden for `key-audit`.
Do not copy raw root material into support tickets or operator reports.

## Current release decision: rotation and retirement deferred (2026-11-06)

The user has explicitly removed full historical root migration (**C09**) and verified retirement (**C10**) from the current Goal 0046 release scope. Both are future optional enhancements, **not implemented or approved**, and require a new authorized Goal and independent acceptance before use. This does not change the original Round 12/13 audit evidence or reduce the protection of existing data.

For the current release, retain the original administrator-managed root and its secure, separately backed-up recovery copy. Changing root configuration without a validated migration is **not supported**: a new root may be unable to authenticate historical Journal or decrypt seeds/checkpoints. Root loss or suspected compromise must be treated as an operational incident requiring a separate containment/recovery decision, not as a reason to re-enable the old partial rotation CLI.

## Legacy rotation implementation and future requirements (not supported in current release)

**The old `--mode rotate` entry point is disabled completely**, including
dry runs and no-Journal hands. The CLI rejects it before opening MySQL or
reading root keys, with a nonzero exit and a clear R12-N01/C09/C10 reason.
No protected seed or Checkpoint is rewritten, regardless of whether a first
live Bid races the command or the CLI process is killed. The previous
per-hand implementation ran two separate transactions, did not
reauthenticate the journal HMAC, and allowed a first action to arrive
between the inventory check and rewrap. Merely counting journal rows or
having an operator request quiescence cannot fix that race.

The ordinary `key-audit` (read-only) and independently authorized
`prune` modes remain available. `--mode rotate` is **not** an approved
method of changing production protection roots. Existing encrypted records
and HMAC data are not modified. A new complete migration command with
quiesced/fenced writers, integrity-checked Journal reauthentication,
atomicity/crash recovery, new-only restore and independent acceptance must
be implemented separately before C09/C10 can be considered fulfilled.

The module-internal re-encryption methods used by historical tests still
exist as infrastructure primitives; their existence does not grant an
operator-safe whole-root migration capability. They must not be wired into
a new CLI or used against live data without completing the above design.

If a migration is later explicitly authorized:

1. Obtain a written operator approval/change window, backups of MySQL and
   all affected roots, and a tested rollback procedure. Stop/quiesce writers
   and coordinate all replicas; do not mix old and new writer configurations.
2. Audit **every** retained journal, encrypted seed, checkpoint and any other
   root-dependent record; classify missing, legacy and corrupt data.
3. Keep the old root read-only while rewrapping encrypted data and
   **authenticating/re-signing** the durable action log under the new root.
   Migration must be idempotent, bounded, integrity-verified and recoverable
   after interruption. The disabled legacy command does not do this.
4. Run a real isolated **new-root-only** cold recovery, historical exact
   command replay and final-evidence verification across the entire retained
   inventory, including failure injection and corrupt-data fail-closed cases.
5. Only after verified coverage, backup/rollback readiness, independent review
   and explicit administrator approval may the old root be removed from
   retained keys and eventually retired. Do not silently delete the recovery
   copy before the approved retirement point.

If a root is suspected compromised, treat it as an incident: restrict access
and affected writes, preserve evidence/backups, and arrange a controlled
emergency migration. Do not just change the environment variable and restart
the service: this can break durable recovery.

## Acceptance boundary

Round 12 established **109 PASS / 1 FAIL / 4 BLOCKED** on its fixed
candidate: R12-N01/P2 demonstrated the unsafe historical rotation race and
a real C09 new-only recovery failure. The unsafe CLI entry is now disabled
as a **subsequent repair candidate**, not independently accepted yet.
C09/C10 still lack full migration/new-only recovery/retirement support;
do not mark these as PASS or silently change their completion conditions.
Goal 0045/0046 remain `in_progress`.

No gameplay APIs, wire formats, domain logic or database schema are changed
by this operator-policy work. It is intentionally outside the normal
latency-sensitive bid/play/pass path.
