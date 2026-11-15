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

## 2026-11-15 Reveal rotation and final continuation — PASS

The subsequent explicit test-key rotation authorization resolves the preceding BLOCKED result. **Goal0061 test-deployment acceptance PASS**, product `1dd54dbf64c3b8375ffa3fe08dfe2a614d34b9f3`; this is not Goal0045/0046 completion or production/device qualification. No product source, images, TLS, persistent protection root or contribution-storage key changed in this continuation; native build/CI/input/smoke and unchanged fault/security suites were reused.

New X25519 Key **`g61-reveal-v2-20261115`**, manifestVersion **2**, active; validity **2026-11-15T02:58:43.438Z → 2026-12-15T02:59:43.438Z** (30 days plus one-minute clock tolerance). Existing Ed25519 signer `g57-sign` retained and derived public key matches both served client trust configurations. Old/new X25519 pairs match. Only `g57-secrets.data.APP_REVEAL_KEYS_STATIC_JSON` changed; all other Secret bytes are identical. Required API replacement loaded the same existing image; one authoritative API replica remains Ready.

Old `g57-reveal` retains its ID, version1, public/private material and original validity. Its status is now retiring with both retirement boundaries at the original expiry: no expired-key revival. Lookup returns410/REVEAL_KEY_RETIRED as required; old archive/own-contribution validation remains usable. There are no other active Reveal Keys in this registry. The signing root has no configured expiration; existing long-lived TLS is unchanged. **Rotate the new business Key before December15**, using the next unique ID/version and retained signing root; no automatic rotation scheduler was introduced.

Current-key HTTP200 and unchanged publication after the API replacement pass. The existing TS verifier using the actually served client root passes signature/hash/time checks and version1→2, rejects bad signatures and rollback, and observes correct ETag304 caching. Actual native pinned-key lookup also verifies signature/hash/time/hand binding for version2. Private manifest artifact SHA256: `44bbf978821e94fca61ced313101a85978a44b37e60576e6a0015356ff52905b`.

**Real business:** reused the original three-account READY/v6 room; no new room/create/join/ready campaign. StartHand persisted the new Key ID. Three real native HPKE contributions committed and cleared their inputs, including page switches. The future beacon produced three distinct private seats/51 distinct private cards. Legal bidding and **58 legal play/pass actions reached COMPLETED/v60**; the durable writer watermark is60 and three contribution records exist. All three clients verified the final archive and each own phrase; each wrong phrase was rejected and input cleared. WSS reconnect plus fresh Vue iframe/account-only recovery preserved Hand/version/seat/cards/public-state digest. All15 actual business groups PASS, no page errors; one new-key full hand only, no repeated game or crash campaign.

Two observation boundaries are explicit. The full-hand harness retained overall FAIL solely because its final observer watched `/current`, while native contributions correctly use the Hand-pinned `/reveal-keys/:keyId`. Its15 business results were not rewritten; one targeted read-only native check of the actual pinned URI PASSes, without replaying gameplay. Also, the old writer row hash changes through existing `LiveJournal.Acquire` epoch/time fencing on the required API restart (now epoch4, committed60); this is expected ownership metadata, not a rewritten action/checkpoint. Old room, Hand, protected seed, contribution, action-log and checkpoint hashes are unchanged; old archive response remains identical, and native old-phrase positive/negative verification passes. No new product defect was confirmed.

Evidence R=`E/reveal-rotation`: `rotation-summary.json`, `manifest-verification.json`, `update-result.json`, `final-compatibility.json`, `final-state.json`, `observer-correction.txt`. Raw `business.json` SHA256 `e5f23129cfe21c555e3a639b61390093d9f957f528211385b51d7d91764a3daa`; targeted `pinned-check.json` PASS SHA256 `20cccf76c687732cc53777bf861b7cffca3194984f6a5e5adeee89f6bf49b01d`. Private before/after Secret backups remain0600 in0700 directory; sensitive candidate removed. Manual rollback is guarded against discarding a Key already bound to persisted hands. Future configuration must retain this new Key, not reapply the expired historical fixture Secret.

Final four Deployments **1/1**, all Pods Ready/restarts0. API=`app-api-5f84ccfc89-t87s6`; the Web/MySQL/Redis Pod UIDs are unchanged from the preceding deployment. Both nodes Ready; available memory sampled1.96/4.02GiB during testing, final2.66/4.02GiB. Sequential browser cap1GiB (targeted check768MiB), zero swap,150%CPU; sampled full-hand browser memory about879MiB, not a peak measurement. No service redistribution needed. Temporary browsers/forward exited; **Web/API/MySQL/Redis remain running**, no scale-to-zero/data deletion.

Operator self-review covered exact-field scope, stable trust, old-key grace/expiry, immutable key material, backup permissions, safe rollback after binding, real pinned-manifest behavior, historical data/fencing distinctions, and honest raw probe results. Earlier broader recovery/device/deferred scopes remain as stated; no performance/soak or extra multi-API qualification is implied.
