# Round12 evidence

Fixed product `62acd921ce767e13d419dd8021e4d065a84e52f0`. Drivers ran from `/root/goal0065-round12-62acd92`, using its exact `input` archive, current compiled production modules and one official native Web export. Private QA account configuration/runtime secrets are excluded.

- `cleanup-core.*`, `reentrant-count.*`: original P3, error isolation/counts and manager reentry.
- `transport-faults.*`: all affected adapter cleanup boundaries; partial native registration/removal and logical versus physical cleanup.
- `game-cleanup.*`, `shell-reentrant.*`: actual Shell/Controller dependencies, waiters/UNKNOWN and connection replacement.
- `native-cleanup.*`: actual official Creator output, independent receivers. 5PASS/4FAIL; failures correspond to three findings.
- `native-cleanup-initial-observer.json`: superseded aggregation bug, documented in report; do not use its overwritten pass flags.
- `session-lifecycle.*`, `controller-cancel-restore.*`, `bootstrap-immediate.*`: R10 original defect paths.
- `smoke-original-observer.json`, `smoke.*`: creation and continuation of the same owned room; actual socket close/recovery, logout/switch and room closure. Initial manager-level injection failure was not a fresh product defect.
- `vue-cleanup.*`, `startup-failure.*`, `cache-smoke.*`: real candidate browser with controlled cleanup/transport faults and ordinary HTTPS/Go controls.
- `checks.py`, `checks.json`, logs: 310 affected tests, type checks/build commands.
- `native-build.cjs`, `native-build.log`: single official export with corrected internal tsc PATH, generated-declaration strict gate/AST audit.
- `ci.json`: exact candidate runs plus explicitly reused product-equivalent Vue run.
- `source-identity.json`, `api-reused.json`, `artifact-hashes.json`, `environment.json`: source/runtime/artifact identities.
- `run-cleanup-callers.txt`: full cleanup reference search, mapped by report table.
- `final-audit.json`, `audit-extra.json`, `deployment-baseline.json`, `owned-cleanup.json`: no rollout; protected cluster/config/data unchanged, owned resources stopped, temporary CA removed and user29457 healthy.

Typical commands: `node cleanup-core.mjs`, `node transport-faults.mjs`, `node game-cleanup.mjs`, `node --experimental-strip-types session-lifecycle.mjs`; browser drivers attach to owned CDP19181 and require the private QA fixture. Raw log line-end whitespace may be normalized for publication; originals remain private. Published file hashes are recorded in `manifest-sha256.json`.
