# Goal0061 — test TLS renewal and deployment continuation

2026-11-15 UTC. **TLS renewal, trusted HTTPS/WSS, deployment and login/room gates PASS. Remaining gameplay acceptance BLOCKED by a separate expired test reveal-key configuration.** This is not full Goal0061 completion.

## Certificate and trust

The existing CA expired2026-11-11 and could not support the requested lifetime. Created exactly one dedicated private test CA, ECDSA P-256/SHA256, with critical CA/pathlen0/keyCertSign constraints. Signed a non-CA P-256 leaf with digitalSignature and serverAuth; no production certificate or unrelated trust configuration changed.

| Certificate | Valid from UTC | Valid until UTC | Lifetime |
| --- | --- | --- | --- |
| Awesome Zero Goal0061 Private Test CA | 2026-11-15 01:30:22 | 2036-11-13 01:30:22 | 3651 days |
| game-web test server | 2026-11-15 01:30:22 | 2036-11-12 01:30:22 | **3650 days** |

SAN DNS: `localhost`, `game-web`, `game-web.awesome-zero-game-test`, `game-web.awesome-zero-game-test.svc`, `game-web.awesome-zero-game-test.svc.cluster.local`. SAN IP: `127.0.0.1`, `::1`, Service IP `10.105.112.89`. Actual access remains loopback `https://127.0.0.1:29457` / `wss://127.0.0.1:29457/ws` through localhost-only forwarding and the existing Nginx same-origin proxy; no ingress/public domain was introduced.

CA SHA256: `A6:22:F7:68:A2:B8:92:3A:DE:FC:2F:5C:C2:B3:EA:21:67:C4:54:2F:EA:7D:BD:1C:65:3F:92:00:BF:F6:2A:96`.
Leaf SHA256: `80:08:69:4F:8A:E3:6E:69:0A:C0:55:87:60:FD:CB:0D:50:3E:0C:9F:BB:4D:B8:67:29:45:E9:67:BB:4C:45:D9`.

Updated only `awesome-zero-game-test/g57-tls` in context `kubernetes-admin@network-learning`; reread data matches generated leaf chain/key, and public-key matching passes. OpenSSL strict chain/self-signature/purpose/IP/Service-DNS verification passes; live TLS hostname/IP checks return0, HTTPS health200. Chromium reports the new issuer, TLS1.3 and secure WebCrypto context; real authenticated room commands traverse WSS. No certificate-error bypass. Default untrusted client correctly rejects this private CA; public CA/browser trust is not claimed.

Private evidence directory E: `/root/.config/awesome-zero-platform/acceptance-0061-tls-20261115` (0700). Separate private-key and original-Secret backup files are0600, never committed/logged. Public trust anchor: `E/pki/ca.crt`; test clients explicitly use this CA. Both NSS stores are in isolated `E/browser-home`, passed as Chromium HOME, leaving other browser/system trust unchanged. `E/rollback.py` supports explicit manual restoration of the expired Secret/previous Web image without deleting data or changing replica counts; it was not executed. Restoring the expired baseline would restore the original blocker.

## Source/artifact reuse and deployed state

Product source remains **`1dd54dbf64c3b8375ffa3fe08dfe2a614d34b9f3`**. Reused the prior accepted native25-file export (all hashes rechecked; manifest SHA256 `5f88b2c5c35d5ea381ecee708c4b16f1e3b504780ef6260a1ddeebaa2a640c12`) and unchanged Vue output; only the existing public same-origin runtime configuration overlays default game-config.js. Git comparisons prove server/Vue/packages unchanged since the reused Goal0058 fixture. No native rebuild, input/rotation/smoke repetition, Go full suite, crypto/full-game/crash/stress campaign or unrelated code change. Prior input/CI and source-bound historical checks remain scoped to the [original report](0061-independent-acceptance.md).

New Web image: `docker.io/awesome-zero-platform/goal0061-web@sha256:d73a2284f993d4dce7a308f2c59aa8b25e84bb5ab14700cb636f1ffbb3747125`. Reused API image: `docker.io/awesome-zero-platform/goal0057-api@sha256:a00c238eb7463929289f16b4675b26d8c13228e2fd27aafcf6e5963c63fe9306`. Actual running Pod hashes match all30 packaged Vue/Cocos files. A missing local digest alias initially caused ErrImageNeverPull; registering the imported digest resolved it without rebuilding or changing source.

Existing MySQL data directory, application configuration and other Secrets retained. Four Deployments were started in dependency order and reached1/1; existing node selectors/resource limits preserved. No production/shared deployment changed; no scale-to-zero cleanup.

| Deployment | Final Pod | Ready | Restarts |
| --- | --- | --- | --- |
| app-api | app-api-985c8f9f5-88ht8 | 1/1 | 0 |
| game-web | game-web-8446c966d5-phpwr | 1/1 | 0 |
| mysql | mysql-5cc48fd9b4-5hz6j | 1/1 | 0 |
| redis | redis-67bdd89687-krvb4 | 1/1 | 0 |

All four remain running on their existing control-plane selectors. Final available memory: control-plane2.82GiB, HK4.02GiB; test Pods' combined working set about230MiB. No redistribution was needed. Browser probes ran sequentially with1GiB cap/zero swap/150% CPU; sampled browser memory reached about754MiB (not a measured peak). Both nodes remain Ready. Final authoritative state: `E/final-state.json`. No sustained-load/resource qualification is implied.

## Business evidence and remaining blocker

Actual Chromium/Go checks pass: three distinct real test account logins; Vue loads the current Cocos export; normal exits from the completed historical room; new room creation; native room-code entry and two joins; three authoritative ready ACKs. Nonmember public room preview contains exactly the six allowed public fields, member=false/sequence0 and no own-discovery membership. Old room and COMPLETED hand rows remain present after normal exits; no database reset/deletion. Evidence: `E/business.json`, retained fixture-error reports, `E/pod-file-verification.json`, `E/retained-data.txt`.

Two private fixture assumptions were corrected against actual contracts: RoomCoordinator exposes roomId/version, not status; nonmembers may read a minimal public200 room preview. Failures are retained, not classified as product defects. The existing created room was reused for continuation; no repeat native build or repeated create/leave campaign.

**StartHand then returned INTERNAL/UNKNOWN.** Read-only diagnosis finds room still READY/v6, active_hand_id=NULL and no hand row. `g57-secrets`' existing `g57-reveal` manifest expires **2026-11-10T12:43:02.613469Z**; the normally verified current-key endpoint returns503/REVEAL_KEY_LOOKUP_FAILED. Registry.Current rejects out-of-validity keys and PrepareHand requires Current, identifying an independent expired test fixture. No business key/manifest/signing root was altered and no second start was issued. Updating this separate business-key configuration was requested explicitly before proceeding beyond TLS authorization.

Personal phrase submission/clearing, fresh three-private-hand isolation, legal bid/play and current hand/page recovery therefore remain **unverified in this deployment**. Earlier Goal0058 evidence remains historical; it does not replace these required current interaction gates. Four healthy Pods do not establish gameplay readiness. Final result remains BLOCKED pending the separate test-key configuration decision.

Temporary browser/forward processes exited; imported image archive removed, native artifact/PKI/rollback evidence retained. To access this private fixture, recreate loopback forwarding with `kubectl --context kubernetes-admin@network-learning -n awesome-zero-game-test port-forward --address=127.0.0.1 service/game-web 29457:443`, then use the CA above for client verification (for example `curl --cacert E/pki/ca.crt https://127.0.0.1:29457/health`, expanding E). No TLS bypass or public trust claim. Goals0045/0046 stay `in_progress`; Goal0056 Honor/ADB pause and C09/C10 DEFERRED unchanged.
