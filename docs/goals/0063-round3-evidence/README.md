# Goal0063 Round3 evidence

Fixed source: `3502b6acedc128d37b63a34b4f5286a9a9e642e2`.

`scoped-summary.json` maps executed checks to conclusions; raw labels are preserved. `native-targeted.json` ends at an optional domain-ACK assertion (zero ACKs actually dropped). `cancel-retry.json` ends at a stats observer framing error after actual archive convergence. Neither raw BLOCKED label is silently relabelled; `queue-idle.json` and `final-sql.json` independently close the remaining observations. See the report and `fixture-notes.md`.

External private fixtures, configs and original logs: `/root/goal0063-round3-3502b6a`. Candidate source is clean at `/tmp/goal0063-round3-3502b6a`. Real terminal-guard overlay source is included as text; apply it at `server/business/doudizhu/infrastructure/mysqlstore/codex_round3_guard_test.go` using Go overlay, with separate MySQL schema. The included final log runs no-landlord/completed/cancel-wins; expiry has its own log. Synthetic keys in that fixture are fixed test values, not deployment credentials.

Images are actual official Cocos Web screenshots, not mockups. The missing-control image is 1280×757, accurately renamed after a CDP reconnect changed the viewport. Other image dimensions match filenames. `sha256.json` covers every evidence file except itself.

No account passwords, JWTs, private keys, runtime config, browser profile, complete private socket captures or binaries are published. Resource stop state is in `cleanup.json`; test data and volumes are retained.

Published native-build.log trims trailing CLI whitespace only; the original log remains in E.
