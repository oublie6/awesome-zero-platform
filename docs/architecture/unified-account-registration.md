# Unified accounts — incremental delivery decision (2026-11-15)

## Authority and ownership
- `server/platform/identity` owns UUID AccountID, password credentials, email/username uniqueness, and verified provider mappings. `authn` issues revocable sessions. `authz` controls permissions. Web/Cocos/WeChat do not own alternate account IDs.
- Games, profiles and historic data remain attached to AccountID regardless of provider binding changes. Never automatically merge accounts or migrate existing game data.
- Administrator account creation remains in place. Self-registration uses the same identity service, never grants admin roles, and never creates a separate web identity.

## V1 development and rollout
- Required: username, unique email, display name and password; phone is optional. Email is **not yet verified**. Never treat the email as proof of ownership for reset, unlink, or other sensitive operations.
- New `POST /auth/register` is **disabled by default** and enabled only with `APP_PUBLIC_REGISTRATION_ENABLED=true` on an isolated IP/HTTPS test deployment. Process-local per-IP quota is a temporary test guard, **not** an internet-grade, distributed abuse protection. Do not enable public internet registration until the TODOs below are complete.
- Registration persists the platform AccountID and password hash through the existing Identity service; the user logs in separately. No new DB/account or game authority.
- Existing `POST /admin/accounts` continues working. Existing `/auth/wechat/code` still auto-provisions: **not compliant with the desired opt-in bind-or-create flow**; do not claim third-party linking complete.
- Rollback: disable environment switch, no account deletion or data migration. Existing accounts remain valid. Existing deployed API image must be rebuilt and separately verified before this route can serve traffic.

## Third-party flow — required before feature completion
1. Verify provider code server-side. If bound, issue normal AccountID session. If unbound, issue a short-lived single-purpose **pending bind** proof, not a user session.
2. User chooses bind existing account (recent password/session verification) or register a new AccountID (username/password and required unverified email during V1).
3. Atomically enforce unique verified provider subject and unique provider/account pair. Prevent concurrent double-bind and reject targets already bound to that provider.
4. For unlink/relink require recent authentication of relevant owners plus verified provider proof, prevent last-login-factor loss, revoke impacted sessions appropriately. No data moves between AccountIDs. Existing automatic WeChat account creation must be removed once this flow is implemented.
5. Never trust provider identity sent by client, never expose raw OpenID, never downgrade root/authz or game ownership guarantees.

## TODO (not implemented)
- [ ] Complete pending-proof, bind-existing/new-account, unlink and relink APIs, database uniqueness and concurrent tests; update WeChat code login to stop silent auto-provisioning.
- [ ] Add email ownership verification with expiring single-use Redis codes, rate limits and secure `MailSender` adapter; require verification before production self-register and recovery.
- [ ] Purchase/configure domain, sender DNS SPF/DKIM/DMARC and real transactional mail provider; switch from test IP/self-managed TLS to trusted production domain HTTPS/WSS.
- [ ] Add production distributed IP/account/email rate limits, abuse controls, admin/audit alerts and privacy notice.
- [ ] Optionally require verified phone later (SMS cost deferred). Do not turn phone into a separate account identity.
- [ ] End-to-end Go/MySQL/Web/WeChat proof tests, real provider device checks, Kubernetes test rollout and independent Codex acceptance.

## Acceptance
Source CI is not proof of deployed functionality. In particular, emailed OTP, public signup hardening, third-party bind/unbind and DNS/provider qualification must remain clearly pending until real evidence exists.
