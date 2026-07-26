# Goal 0012: Account Row Action Buttons

## Status

- State: in_progress
- Started: 2026-07-26
- Completed:
- Blockers:

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
- Confirmed the account action column currently uses four `link` buttons, which visually resemble adjacent text rather than distinct controls.

### In progress

- Replacing the link actions with compact bordered buttons.

### Remaining

- Update the account action template and scoped layout.
- Run the full verification gate.
- Record completion evidence.

## Completion Report

Not completed.
