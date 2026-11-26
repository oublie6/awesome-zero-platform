# Round11 evidence index

Fixed product: `33c05a74a98315b2d90a51e817b3cf1fcde122a6`.

- `ci.json`: exact-source Actions jobs, including skipped native export.
- `checks.py` / `checks.json` / test logs: actual affected-module commands/results (303 tests).
- `native-build.cjs` / `native-build.log`: single official Linux Web export and independent generated-type/AST gate.
- `source-identity.json`, `api-reused.json`, `artifact-hashes.json`: immutable source, reused identical-server API, exported/staged file identities.
- `operation-independent.*`: public lifecycle primitive, positive cases and cleanup-exception counterexamples.
- `session-lifecycle.*`, `controller-cancel-restore.*`, `session-extra.*`, `web-auth-extra.*`: original R10 N01 plus session/storage/owner boundaries.
- `platform-independent.*`, `native-adapters.*`: mini login/random/share, controlled SDK; latter executes the actual exported Creator bundle.
- `provider-matrix.*`, `bootstrap-lifecycle.*`, `bootstrap-adjacent.*`: real compiled classes, actual menus/navigation, normal provider flows, cancellation/dispatch/unknown-result boundary, stale recovery.
- `bootstrap-initial-contract-observer.json`: obsolete observer expecting final submitted bind/register to be cancellable; adjudicated against amended dispatch contract, not new defects.
- `callsite-review.json`, `async-callsite-index.txt`: complete new-operation consumer inventory and adjacent audit scope.
- `vue-lifecycle.*`, `vue-foreground-raw.*`, `vue-unmount.*`, `startup-failure.*`: actual browser lifecycle and matching API, with explicitly controlled failure points.
- `login-chain.*`, `authority-matrix.*`: R8 original login/authority regressions.

Drivers ran in `/root/goal0065-round11-33c05a7` against its `input` candidate snapshot. Node v22.23.4; examples: `node --experimental-strip-types session-lifecycle.mjs`, `node operation-independent.mjs`, `node provider-matrix.mjs`; browser drivers attach only to owned CDP19181. `authority-matrix.mjs` was run with `AUTHORITY_CASE='R8 final membership'`, not its entire historical suite.

Private QA credentials, runtime secrets, raw private hands, phrases and screenshots are deliberately excluded. Reproduction needs the private owned QA fixture and same built candidate, not user credentials. Historical backend/cache evidence is linked from the report and was not rerun as fresh evidence.

- `binding-chain.*`: full native Bootstrap first bind/register-bind → binding-required form → fresh login/identity/socket/account path, controlled SDK/HTTP.
- `bootstrap-immediate.*`: starts a new operation via the actual cancel menu before awaiting the old Promise.
- `provider-timeouts.*`: cancellation/timeout/unknown-result controller boundaries; records requested15s timers accelerated to5ms in this fixture.
- `game-summary.json`, `game-exit.json`, `game.log`, `r11-live.json`, `automatic-deal.json`, `startup-gates.json`: one real three-account full game and applicable regressions; raw private payloads excluded.
- `deployment-baseline.json`, `final-audit.json`, `audit-extra.json`, `owned-cleanup.json`: unchanged cluster, isolated terminal counts, stopped owned resources, removed temporary test CA, preserved29457 HTTPS health.
- `environment.json`, `publication-audit.json`, `manifest-sha256.json`: environment, private-data exclusion scan, published-file identities.
