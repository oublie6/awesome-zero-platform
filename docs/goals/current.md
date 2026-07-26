# Goal 0011: Admin Layout and Button Text Stabilization

## Status

- State: in_progress
- Started: 2026-07-26
- Completed:
- Blockers:

## Goal

Fix overlapping and cramped button labels in the Vue 3 Admin Web, establish reusable button-group and toolbar layout rules, and improve the affected account, role, resource, and page-header layouts without changing Admin behavior or API contracts.

## References

- `AGENTS.md`
- `clients/admin-web/src/styles.css`
- `clients/admin-web/src/components/PageHeader.vue`
- `clients/admin-web/src/layouts/AdminLayout.vue`
- `clients/admin-web/src/views/accounts/AccountsView.vue`
- `clients/admin-web/src/views/roles/RolesView.vue`
- `clients/admin-web/src/views/authorization/StandardView.vue`
- `clients/admin-web/src/views/authorization/EngineView.vue`
- `clients/admin-web/src/views/AuditView.vue`

## Deliverables

1. Prevent Element Plus button labels from shrinking, wrapping into themselves, or visually overlapping adjacent labels.
2. Add reusable flex layouts for toolbars, page actions, table actions, and compact button groups.
3. Make dense table action areas wrap cleanly while preserving readable row heights and click targets.
4. Make PageHeader action slots and role-header actions responsive instead of forcing content into one line.
5. Remove the unnecessary global 1100px minimum page width while retaining a usable desktop Admin layout and safe horizontal table scrolling.
6. Keep all existing button text, actions, permissions, routes, and API calls unchanged.
7. Build the Vue application successfully and pass the repository `ci/full` gate.

## Constraints

- Work directly on `main`; do not create branches or pull requests.
- Do not redesign the visual identity or introduce a new UI framework.
- Do not change backend APIs, authorization behavior, or database schema.
- Prefer shared CSS and small template wrappers over page-specific width hacks.
- Preserve UTF-8 response configuration.
- Do not claim visual browser verification unless it was actually performed.

## Required Verification

```bash
cd clients/admin-web
npm install --no-audit --no-fund
npm run build
cd ../..

git diff --check
```

GitHub Actions must subsequently report `ci/full: success` for the final commit.

## Acceptance Criteria

- Button labels remain on one line inside each button.
- Adjacent buttons have explicit gaps and do not rely on Element Plus sibling margins inside flex groups.
- Account and resource table actions wrap cleanly without overlapping.
- Role header actions remain readable when horizontal space is constrained.
- Toolbars can wrap filters and actions onto additional rows.
- Page-header actions do not collide with titles or descriptions.
- Existing Admin functionality and labels remain unchanged.
- Admin Web type checking and production build pass.
- Final `ci/full` passes.

## Working State

### Completed

- Archived completed Goal 0010.
- Inspected the global stylesheet, PageHeader, Admin layout, account table, role workspace, resource table, authorization engine toolbar, and audit toolbar.
- Identified fixed global minimum width, non-wrapping toolbars, and ungrouped table action buttons as the main layout risks.

### In progress

- Implementing shared button and responsive layout rules.

### Remaining

- Update affected Vue templates and styles.
- Run frontend build verification.
- Confirm final `ci/full` success.
- Record final commit and completion evidence.

## Completion Report

Not completed.
