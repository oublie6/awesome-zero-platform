# Identity Account Foundation Requirements

## Purpose

Establish the platform's first reusable domain capability: identity accounts and password credentials.

This foundation provides account creation, lookup, profile maintenance, account status management, secure password hashing and verification, MySQL persistence, transaction consistency, and focused tests.

It does not provide public registration, login, tokens, sessions, authorization, roles, permissions, tenants, verification codes, third-party login, frontend behavior, or product-specific user semantics.

## Module boundary

The implementation belongs under:

```text
server/platform/identity/
```

The module may contain focused packages for accounts, passwords, persistence, and services. Package boundaries must follow ownership and dependency direction rather than a fixed layer template.

The identity module owns its domain rules and database tables. Other platform or business modules must not query identity tables directly.

Do not create generic `common`, `utils`, `helpers`, generic repository, generic CRUD, active-record, dependency-injection, plugin, or database-driver abstraction packages.

## Account model

An account contains at least:

- A stable internal account ID.
- Optional username.
- Optional email address.
- Optional phone number.
- Display name.
- Account status.
- Creation timestamp.
- Update timestamp.

Supported initial statuses are:

- `active`
- `disabled`

At least one supported identity value must be supplied when creating an account.

When present, username, email, and phone must be normalized consistently and protected by database-level unique constraints. Account records are not physically deleted in this phase.

The account table must not contain roles, permissions, departments, tenants, menus, session state, JWT data, product fields, or audit-event history.

## Password credentials

Password credentials must be stored separately from account profile data and contain at least:

- Account ID.
- Password hash.
- Password-changed timestamp.
- Creation timestamp.
- Update timestamp.

Plaintext passwords must never be stored, logged, returned, seeded, committed, or included in errors.

Use a maintained password-hashing algorithm suitable for application authentication. Argon2id is preferred. Parameters must be explicit, documented, and conservative enough for the repository's 1–2 GB development environment while remaining suitable for real authentication use.

Password input must have explicit minimum and maximum lengths. Oversized inputs must be rejected before expensive hashing work. Verification must use the selected library's safe comparison behavior.

## Supported operations

The identity capability must support focused operations equivalent to:

- Create an account with an initial password.
- Get an account by ID.
- Find an account by username.
- Find an account by email.
- Find an account by phone.
- Update supported profile fields.
- Enable an account.
- Disable an account.
- Verify an account password.
- Change an account password.

Names and signatures may follow idiomatic Go conventions.

No public HTTP account-management or authentication endpoints are required in this phase.

## Persistence

Persistence interfaces must be specific to the identity module. Do not introduce a platform-wide generic repository.

MySQL implementations must use explicit, readable SQL and the existing database foundation.

Domain and persistence failures must be translated into focused safe errors, including equivalents of:

- Account not found.
- Identity conflict.
- Invalid account state.
- Invalid credentials.
- Internal persistence failure.

Raw MySQL errors, SQL statements, DSNs, credentials, passwords, hashes, and internal addresses must not escape through public service errors.

## Transactions

Creating an account and its initial password credential must occur in one transaction using `server/foundation/database`.

Required behavior:

- Commit when all writes succeed.
- Roll back when account insertion fails.
- Roll back when credential insertion fails.
- Never leave a partial account.
- Never leave a credential without an account.
- Preserve useful error meaning without exposing sensitive implementation details.

Changing a password must remain consistent and transactional whenever multiple writes are required.

## Database schema

Update the current complete schema at:

```text
server/database/schema/current.sql
```

Suggested tables:

```text
identity_accounts
identity_password_credentials
```

Use deterministic MySQL syntax, InnoDB, `utf8mb4`, UTC-safe timestamps, explicit foreign keys where appropriate, and explicit unique constraints.

Do not create migration history or commit temporary upgrade SQL. The schema must remain rebuildable from a clean database.

Development seed data must not contain a default administrator, password, fake user, role, permission, or product account.

## Identifier policy

Account IDs must not depend on MySQL auto-increment sequencing. Choose one stable identifier approach for identity entities and document it.

Do not introduce a generic distributed-ID platform unless the selected account implementation genuinely requires it.

## Security requirements

The implementation must:

- Never persist or log plaintext passwords.
- Never log password hashes.
- Never include passwords or hashes in errors.
- Reject empty, invalid, and oversized password inputs safely.
- Use secure library-provided verification behavior.
- Avoid hidden credential mutation or automatic password retries.
- Keep production hashing defaults explicit and testable.
- Use bounded test parameters where necessary without weakening application defaults.

## Testing

Tests must cover:

- Valid account creation.
- Rejection when no username, email, or phone is supplied.
- Duplicate username, email, and phone conflicts.
- Lookup by ID, username, email, and phone.
- Account-not-found behavior.
- Profile updates.
- Enable and disable transitions.
- Password hashes not containing plaintext.
- Correct and incorrect password verification.
- Password change invalidating the old password.
- Empty and oversized password rejection before expensive hashing.
- Transaction rollback when credential creation fails.
- No partial account after transaction failure.
- Concurrent creation of the same unique identity producing only one success.
- Safe errors and logs without passwords, hashes, DSNs, SQL, raw MySQL errors, or internal addresses.
- Clean-schema application and isolated MySQL integration behavior.

All Go tests must follow the low-memory rules in `AGENTS.md`:

```bash
go test -p 1 -parallel 1 ./...
```

Docker-backed integration checks, compilation, code generation, password hashing tests, and subagents must run sequentially.

## Documentation

Active documentation must describe:

- Identity module ownership.
- Account and credential responsibilities.
- Supported account states.
- Password security policy.
- Database schema ownership.
- Identifier choice.
- The absence of public login and account-management APIs in this phase.
- Deferred capabilities.

## Deferred capabilities

The following are explicitly deferred:

- Public registration and login.
- JWT access tokens and refresh tokens.
- Redis-backed sessions and logout.
- Token revocation and password recovery.
- Email or SMS verification and CAPTCHA.
- Login throttling and risk controls.
- WeChat, OAuth, OIDC, SAML, or other third-party login.
- Roles, permissions, Casbin, menus, departments, organizations, and tenants.
- User avatars and file upload.
- Audit-event storage and notifications.
- Management frontend or other client code.
