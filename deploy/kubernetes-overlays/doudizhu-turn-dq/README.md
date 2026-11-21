# Goal0063 isolated turn scheduler infrastructure — NOT deployed

**Do not apply directly.** Beanstalkd image name is deliberately an UNBUILT placeholder.
Build the Dockerfile `deploy/docker/beanstalkd.Dockerfile`, publish to an approved
registry, and replace BOTH StatefulSet image references with the verified immutable
digest. Then verify allocated PVCs, DB backup/DDL, namespace scope and rollback.
Only use context `kubernetes-admin@network-learning`, namespace
`awesome-zero-game-test`; no production rollout is authorized.

`go-queue/dq.NewProducer` requires at least TWO DISTINCT Beanstalkd endpoints.
Two separate StatefulSets and Services are intentionally provisioned, each
with an independent 1Gi PVC, with no public Service exposure.

Test configuration, injected only after safe additive migration and a fully
drained/examined set of active game hands:

```
APP_DOUDIZHU_TURN_DQ_ENABLED=true
APP_DOUDIZHU_TURN_DQ_ENDPOINTS=beanstalkd-a.awesome-zero-game-test.svc:11300,beanstalkd-b.awesome-zero-game-test.svc:11300
APP_DOUDIZHU_TURN_DQ_TUBE=doudizhu-turn
```

Reuse existing configured go-zero Redis on DB 0; no dedicated new password.
The default is DISABLED. With disabled mode the previous shared 1s lifecycle
sweep (which expires whole hands) remains unchanged; this is intentionally
NOT deployed until correctness and UX tests are accepted.

Run the authoritative `server/database/schema/current.sql` additive DDL
safely, preserving all prior data. The new table is append/update-only per
active hand; inspect existing active hands first. A running hand without a
recorded deadline MUST block enabling the feature; never reset its time limit.

Dq consumer deletes a task before invoking user callback and uses Redis
SETNX-based deduplication. Neither queue replication nor its binlog replaces
MySQL authority. App worker's bounded repair scan retries failed delivery
after 2 seconds and completed delivery after deadline+10 seconds until a new
turn version or completed hand retires it.

Operational checks: both queue endpoints and PVCs healthy, queue depth,
oldest outstanding deadline, failed enqueue/callback rate, rejected stale
tickets, MySQL deadline drift and reconnect display. Verify rollback with
the API drained: disable the feature only after all active hands are
completed/aborted under explicit operator review. Do not mix old sweep mode
with enabled new mode during the same active hand.
