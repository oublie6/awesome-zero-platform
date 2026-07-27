# Goal 0013: Adaptive Account Row Actions

## Status

- State: in_progress
- Started: 2026-07-26
- Completed:
- Blockers:

## Goal

Replace the fixed 2×2 account-row action layout with an adaptive action bar that measures available width, shows as many direct buttons as fit, and moves the remaining actions into a three-dot dropdown menu.

## References

- `AGENTS.md`
- `clients/admin-web/src/views/accounts/AccountsView.vue`
- `clients/admin-web/src/components/AccountRowActions.vue`

## Deliverables

1. Measure the available account action-cell width at runtime with `ResizeObserver`.
2. Keep the action priority order `详情`, `编辑`, `启用/禁用`, `重置密码`.
3. Display the maximum number of direct compact buttons that fit without crowding.
4. Show a three-dot button whenever one or more actions do not fit.
5. Place all remaining actions in an Element Plus dropdown menu.
6. Recalculate automatically when the table column or viewport width changes.
7. Preserve all existing handlers, labels, confirmation behavior, permissions, and API calls.
8. Pass Vue type checking, production build, and the repository `ci/full` gate.

## Constraints

- Work directly on `main`; do not create branches or pull requests.
- Limit behavior changes to account-row action presentation.
- Do not change backend APIs or account-management semantics.
- Do not hide actions permanently; every action must remain reachable through a direct button or dropdown item.
- Keep destructive action styling distinguishable in the dropdown.

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

- The action container measures its actual rendered width.
- The maximum number of actions that fit are displayed directly.
- Overflow actions are accessible through a visible three-dot dropdown button.
- Resizing the container updates direct and overflow actions automatically.
- No labels overlap or appear crowded.
- `重置密码` remains visually marked as dangerous in both direct and dropdown forms.
- Existing actions still call the original handlers.
- Admin Web build and final `ci/full` pass.

## Working State

### Completed

- Archived completed Goal 0012.
- Confirmed the fixed 2×2 layout remains visually dense for the account table.
- Defined direct-action priority and overflow behavior.

### In progress

- Implementing the width-aware action component and account-table integration.

### Remaining

- Add the adaptive component.
- Replace the fixed account action grid.
- Run the full verification gate.
- Record completion evidence.

## Completion Report

Not completed.
