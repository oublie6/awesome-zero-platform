# Web precompression evidence — 2026-11-27

See [performance/deployment report](../../web-precompression-20261127.md) for environment, scope, methods, limits, image identities and conclusions.

`benchmark-before/after.json` contain all three cold/hot pairs per deployed version and per-metric medians. `cpu-before/after.json` contain three sequential downloads and container-cgroup CPU deltas. `http-30474.json` is the isolated candidate; `http-29457.json` is actual deployment. `precompression-manifest.json` contains every original and retained gzip hash/size. `running-image.json`, `integrity.json`, `rollout-result.json` and `cleanup.json` record deployment identity, protected resources and owned-resource cleanup. `server-auth-timing.json` contains sanitized access-log samples selected by browser time windows, not request-ID proof.

Reference harnesses are the actual executed scripts, preserved for method review. They depend on the private working directory, verified artifacts, owned QA `accounts.json`, fixture probes, prior test CA and privileged local Kubernetes/container access. Those inputs and credentials are deliberately not published. Do not run the fixture-mutating isolated HTTP branch against production. Formal reusable packaging code and automated tests are under `scripts/game-web/`; reproduction commands are in `deploy/README.md`.

The initial exact-button-name observer error was corrected before the six valid baseline samples were collected. No incomplete observer run is counted as success. This is implementation/self-review evidence, not a new independent Goal acceptance.
