# Goal 0012: Account Row Action Buttons

## Status

- State: completed
- Started: 2026-07-26
- Completed: 2026-07-26
- Blockers: None.

## Goal

Make every account-row action in the Vue 3 Admin Web visually distinct by replacing text-link actions with four independent compact buttons arranged in a clear two-column layout.

## References

- `AGENTS.md`
- `clients/admin-web/src/views/accounts/AccountsView.vue`
- `clients/admin-web/src/styles.css`

## Deliverables

1. Replace the account table row's link-style action texts with four real compact buttons.
2. Present `详情`, `编辑`, `启用/禁用`, and `重置密码` as independent controls with visible boundaries.
3. Use a stable two-column action layout so labels neither touch nor resemble one combined button.
4. Preserve all existing click handlers, confirmation behavior, permissions, labels, and API calls.
5. Pass Vue type checking, production build, and the repository `ci/full` gate.

## Constraints

- Work directly on `main`; do not create branches or pull requests.
- Limit production changes to the account-row action presentation.
- Do not change backend APIs or account-management behavior.
- Do not use link-style buttons for this row action group.

## Required Verification

```bash
cd clients/admin-web
npm install --no-audit --no-fund
npm run build
cd ../..

git diff --check
```

GitHub Actions must report `ci/full: success` for the final commit.

## Acceptance Criteria

- Every account row shows four visibly independent buttons.
- Buttons are arranged as a 2×2 grid with consistent spacing.
- Button labels do not touch, overlap, or appear as one continuous string.
- Enable/disable and reset-password actions remain visually differentiated.
- Existing actions still call the original handlers.
- Admin Web build and final `ci/full` pass.

## Working State

### Completed

- Archived Goal 0011.
- Confirmed the account action column used four `link` buttons, which visually resembled adjacent text rather than distinct controls.
- Replaced all four row actions with compact bordered Element Plus buttons.
- Arranged `详情`, `编辑`, `启用/禁用`, and `重置密码` in a stable 2×2 grid.
- Applied distinct semantic styles: info for details, primary for edit, warning/success for disable/enable, and danger for password reset.
- Kept the original handlers, confirmation behavior, labels, API calls, and fixed-right action column.
- Set a stable 236px operation-column width and full-width centered buttons with 8px grid gaps.

### In progress

- None.

### Remaining

- None.

### Verification status

- Implementation commit: `35f93179777347f72426ceb6836105972034c925`.
- GitHub Actions run `30226117605` reported `ci/full: success`.
- Admin Web dependency installation, Vue type checking, and production build passed.
- Clean client source verification passed.
- Repository unit, race, build, Compose validation, MySQL 5.7 integration, clustered authorization, and production runtime jobs passed.
- No backend API, database schema, authorization behavior, or runtime secret changed.

## Completion Report

Completed on 2026-07-26.

The account-management operation column presents four visually independent controls instead of a continuous row of link text:

- `详情` — neutral information button;
- `编辑` — primary button;
- `禁用` or `启用` — warning/success button according to account state;
- `重置密码` — danger button.

The controls are displayed as a 2×2 grid with visible borders, equal widths, centered labels, and 8px spacing. Existing account-management behavior remains unchanged. The implementation passed the full repository gate on commit `35f93179777347f72426ceb6836105972034c925` in GitHub Actions run `30226117605`.
