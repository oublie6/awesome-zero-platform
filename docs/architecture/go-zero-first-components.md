# go-zero First: foundation component standard (Goal 0038)

The root [AGENTS.md](../../AGENTS.md) is normative. Follow go-zero v1.10.2 public APIs by default and preserve the existing modular-monolith boundaries. Business modules should use their own repository ports; adapters use go-zero components for ordinary supported operations. Keeping native drivers is allowed only for explicit compatibility gaps, not as a competing default.

## Current default paths

| Capability | Default component | Implementation |
| --- | --- | --- |
| MySQL connection and standard SQL read/write | go-zero `sqlx.SqlConn` backed by the configured `*sql.DB` pool | `server/foundation/database/database.go`, identity, admin, Casbin reads, Doudizhu ordinary read, final archive |
| Redis common commands | go-zero `core/stores/redis.Redis` | `server/foundation/cache/cache.go`, `platform/authn/adapter/redissession`, Casbin publish |
| Entity cache and cache-aside | go-zero `core/stores/cache` | `server/foundation/cache/model.go` |
| HTTP/router/config/logs/traces | go-zero REST/conf/logx/OpenTelemetry | Existing `app-api` composition root |
| WebSocket protocol, custom monitoring, Casbin policy semantics | Gorilla / Prometheus SDK / Casbin as specialized capabilities | Do not replace merely to eliminate an independent library; no equivalent complete go-zero component exists |

**MySQL:** `database.WrapSqlConn(db)` reuses the existing configured pool and retains SQL tracing, with expected MySQL constraint conflicts marked acceptable for sqlx circuit breaking. The existing pool, application error semantics, schema and transaction isolation remain unchanged. SQL statement logging is disabled at startup to avoid exposing credentials, hashes, or business payloads. Span availability still depends on propagated context and OTel sampling.

**Redis:** The go-zero client supports DB 0; `GetCtx`, `DelCtx`, `ScanCtx`, `PipelinedCtx`, and `PublishCtx` are used for matching business operations. Session create uses `PipelinedCtx` with native `SetNX` on the *go-zero-owned* pipeliner to preserve subsecond expiration semantics. ModelCache explicitly rejects configuration of a nonzero Redis logical DB rather than silently reading DB 0. If model caching is disabled and DB is nonzero, the native client remains the correct route.

## Justified native client operations

- **MySQL driver and pool startup/readiness:** `database/sql` is the underlying SQL driver/pool reused by sqlx; `PingContext` and close belong to its lifecycle.
- **Identity password/account transactions:** `foundation/database.WithinTransaction` and identity transactional methods retain `*sql.Tx` to preserve rollback/panic/error composition and `FOR UPDATE` semantics. Replacing this contract requires dedicated transaction migration coverage.
- **Doudizhu command transactions:** `business/doudizhu/infrastructure/mysqlstore/store.go` and `transaction.go` explicitly require `sql.LevelReadCommitted` and precise deduplication/locking/rollback behavior. Standard queries outside these transactions use sqlx.
- **Casbin policy writes/synchronization and super-admin critical sections:** existing native multi-statement transactions and locks preserve serialized policy updates and Casbin's callback contract. Policy reads and straightforward writes use sqlx when equivalent.
- **Redis WATCH/TxPipelined:** `platform/authn/adapter/redissession/store.go` uses native `go-redis/v9` only for atomic refresh-token rotation; go-zero v1.10.2 exposes no equivalent WATCH transaction.
- **Redis Pub/Sub Receive/Subscribe:** `platform/authz/adapter/casbinmysql/cluster.go` retains the native long-lived subscription for resubscription, shutdown, and reconnection behavior; publish prefers go-zero. The background subscription does not automatically become a per-request child span.
- **Native Redis DB != 0:** `foundation/cache/cache.go` exposes a native-only fallback because go-zero v1.10.2 uses DB 0. The application startup probe uses the native client.

## Review and upgrade cautions

The go-zero Redis implementation uses its own shared connection management: do not assume the native client pool-size/timeouts apply to its internal connections. This is an explicit compatibility limit; validate under representative load and monitor connection counts. The original native pool remains necessary for WATCH/Subscribe and health checks. Do not silently add a second model cache on another logical DB.

Trace context must reach `sqlx.*Ctx` and `redis.*Ctx`; raw `*sql.Tx`, `WATCH`, background tasks, and protocol-specific WebSocket messages need separate instrumentation for full coverage. Prometheus metrics are not a substitute for Jaeger spans. SQL logs are disabled and go-zero Redis's slow-command logging threshold raised to protect sensitive values; keep structured redacted application errors and traces.

Verified behavior is defined by CI and focused real-dependency integration tests; see Goal 0038 archive after completion. Cluster-level Jaeger child-span verification is a separate environment-only acceptance step, not something repository-only CI can prove.
