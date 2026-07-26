# Goal 0011: Admin Layout and Button Text Stabilization

## Status

- State: completed
- Started: 2026-07-26
- Completed: 2026-07-26
- Blockers: None.

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
- Replaced the global 1100px body minimum with flexible page sizing while keeping a smaller Admin-shell safety width for the desktop control plane.
- Added reusable wrapping layouts for toolbars, generic button groups, and table actions.
- Prevented Element Plus button labels from shrinking or breaking across lines.
- Added explicit gap handling so grouped buttons no longer inherit conflicting sibling margins.
- Made PageHeader copy/actions wrap independently and added responsive stacking at constrained widths.
- Wrapped account and resource table actions in reusable action containers and replaced rigid operation widths with responsive minimum widths.
- Made the role workspace, role header actions, permission checkbox matrix, and narrow-window role list responsive.
- Hardened the Admin top bar against title, role badge, and user-name collisions through ellipsis and fixed action sizing.
- Preserved all existing labels, routes, click handlers, API calls, permissions, and UTF-8 Nginx configuration.

### In progress

- None.

### Remaining

- None.

### Verification status

- Implementation checkpoint: `47ab576f568c305b15fcded5f34147be209972fb`.
- GitHub Actions run `30211902545` reported `ci/full: success`.
- Admin Web dependency installation, Vue type checking, and production build passed.
- Clean client source verification passed.
- Repository module, generated-code, formatting, Go unit, race, and build checks passed.
- MySQL 5.7/Redis integration and clustered authorization tests passed.
- Production Compose runtime acceptance passed.
- No backend API, authorization rule, database schema, or runtime secret changed.
- Direct browser screenshot comparison was not available in this execution environment; completion is based on source inspection, responsive layout contracts, Vue production build, and the repository runtime gate.

## Completion Report

Completed on 2026-07-26.

The Admin Web now uses shared layout primitives instead of relying on implicit Element Plus button margins and rigid one-line containers. Button labels are kept intact, adjacent controls use explicit gaps, dense table actions can wrap onto additional rows, and page/role headers can reflow without text collisions.

Changed areas:

- `clients/admin-web/src/styles.css`
- `clients/admin-web/src/components/PageHeader.vue`
- `clients/admin-web/src/layouts/AdminLayout.vue`
- `clients/admin-web/src/views/accounts/AccountsView.vue`
- `clients/admin-web/src/views/roles/RolesView.vue`
- `clients/admin-web/src/views/authorization/StandardView.vue`

The implementation was committed directly to `main`. The full repository gate passed on implementation commit `47ab576f568c305b15fcded5f34147be209972fb` in GitHub Actions run `30211902545`.
