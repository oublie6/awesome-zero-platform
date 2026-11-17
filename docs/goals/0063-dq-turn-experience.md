# Goal 0063 — DQ turn deadlines, automatic actions and Cocos table acceptance

Status: **in_progress** — implementation branch; NOT deployed, NOT accepted.

Product baseline: `e9cfd77e409b85aad1f37e351cc7e0e3d63a5211`, prior Goal0062 Round5 Kubernetes PASS. Protect that environment until a separate authorized test rollout.

## Confirmed source findings

The existing `lifecycle.Supervisor` already uses one shared second-resolution ticker with in-memory per-hand entries, 45-second bidding and 60-second playing timeouts; its due path currently **expires the entire hand**, not auto-bid/pass/play. `ReconcileRecovered` currently rearms after restart; it does not preserve original deadlines. `go-queue/dq` producer requires **at least two distinct Beanstalkd endpoints**. The consumer deletes the reserved job **before** callback and applies a Redis one-hour duplicate filter; its wrapper also drops tasks sufficiently past their nominal deadline. It cannot alone guarantee retry or exactly-once execution.

## Required delivery scope

1. Introduce non-destructive authoritative MySQL per-turn deadline and delivery recovery, linked to the exact committed live version and current player. Never extend it on restart, duplicate callbacks, or reconnect.
2. Use `go-queue/dq` Beanstalkd backed scheduling (two distinct nodes), Redis dedup only, and a bounded durable reconciliation/repair scan; avoid one timer/goroutine per game. Repeated enqueue must carry a new delivery ID because dq locks duplicate bodies for one hour.
3. Worker compares actual MySQL deadline, version, phase, player and current committed game before acting; handles partial producer failures, dropped callback, duplicate jobs, epoch takeovers and delayed messages.
4. Replace expiry-on-timeout with an actual **server-committed** no-bid (bidding), pass (when legal), or deterministic minimal legal single-card lead. Use existing go-zero-based application authority, live command journal, persistent idempotent ID and player membership; no invented client authority.
5. Expose server deadline and server clock to authorized clients. Cocos countdown is presentation-only; it survives reconnect, resizes, turn changes and stale frames. Test together with previous three-seat, card ordering and click hitboxes.
6. Provide Beanstalkd x2 with bounded resources and durable storage in isolated manifests, operator controls/metrics and a reversible non-destructive migration plan.
7. Independently validate fail/restore, last-millisecond human-vs-timeout races, duplicate delivery, MySQL and queue outage, restart, no false hand termination, and actual Cocos4 Web screenshot/hit testing on desktop and mobile viewport using the same fixed SHA.

No deployment or Goal closure until ALL gates pass. Never mutate production or expose credentials.
