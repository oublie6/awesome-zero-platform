# Goal 0062 — unified AccountID, IP-test registration and third-party links

## State
`in_progress` — ChatGPT-owned implementation. **Not independently accepted.**

## Scope and authority
The source of authority is the single platform UUID AccountID. Neither WeChat OpenID nor a Web user ID owns business data. Web self-registration, administrator creation and verified third-party bindings use the same Identity service. Preserve all existing game hands, archives, MySQL protection roots, historical AccountIDs and administrator routes. Do not expand into account merging, payments, phone SMS or public domain migration.

## Implemented candidate (subject to CI and Codex)
- Gated `POST /auth/register` creates a platform AccountID with mandatory unique username/email, display name, Argon2id password. Email is **not verified**. Login remains separate. `APP_PUBLIC_REGISTRATION_ENABLED` defaults to off.
- Verified WeChat `/auth/wechat/code` returns a normal session **only** for a verified, already-bound active platform account. Unbound identities receive a five-minute opaque pending binding proof rather than an automatic account or token.
- `/auth/wechat/bind` binds the verified proof to an existing password-authenticated AccountID; `/auth/wechat/register` atomically creates the new account, credential and binding under the gated test-registration switch.
- Unlink requires an authenticated source session and password; relink additionally requires a newly verified WeChat code, an authenticated source session plus both source and target account passwords, and an unoccupied destination account. Revoking existing sessions is part of security handling.
- DB invariant: unique provider/subject and unique provider/account. A schema preflight must fail closed if either required unique constraint or pending table is missing. No game or identity account data moves on unlink/relink.
- Cocos WeChat mini-game presents bind-existing/create-new, unlink and relink UI. Vue3 Web presents open-account registration. No email verification or SMS is implied.
- Existing test deployment at SHA `1dd54db...` must remain unchanged until separately authorized schema/application rollout.

## Acceptance gates
1. Full affected Go unit and MySQL integration tests, concurrent binding and pending-proof replay tests, and unchanged admin controls; exact new candidate SHA.
2. Vue3 Web typecheck/test/build plus Cocos static/native-source typechecks and WeChat binding flow tests.
3. Provider adapter must reject client-supplied OpenIDs, unverified codes, malformed proofs and token replay.
4. Existing AccountID retains all data through unlink/relink; source and target credentials, provider proof, session checks and unique target check are all enforced.
5. Admin account creation still works; self-registration has no ability to grant elevated roles.
6. Only after safe review of **existing test MySQL schema** for duplicates, additive non-destructive migration and rollback plan may an updated candidate be deployed to the isolated K8s namespace. Preserve existing game workloads/data. Any absent platform credentials cause real-WeChat qualification BLOCKED, not a fake pass.
7. Codex independently reproduces affected flows at the frozen source SHA. If the environment prevents real provider verification, record that boundary honestly; mocks do not qualify real mobile device login.
8. Do not mark 0045/0046 or any device qualification complete based on this Goal.

## Explicit TODO — postponed by user
- Real validated email ownership with single-use short-TTL codes, delivery provider and DNS SPF/DKIM/DMARC.
- Public domain, publicly trusted HTTPS/WSS and internet-grade distributed rate limiting.
- Verified phone as possible future mandatory field; no SMS implementation now.
- Production data migration and a recovery path for historic auto-provisioned WeChat-only accounts with unknown random password; do not bypass password proof.
- Real WeChat account-binding qualification on supported device after existing Goal0056 tool/credential blockers are lifted.

## Implementation/acceptance separation
ChatGPT modifies and tests; Codex **only** independently audits the final candidate and appends its report. Do not merge, rewrite or conceal historical Goal0061 evidence. Source CI, local mocked provider exchange and a healthy K8s Deployment cannot substitute for real provider-device proof.
