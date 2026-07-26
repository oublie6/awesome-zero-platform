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

## Completion Report

Completed on 2026-07-26.

The Admin Web uses shared layout primitives for wrapping toolbars, button groups, table actions, responsive headers, and narrow-window role layouts. The implementation checkpoint `47ab576f568c305b15fcded5f34147be209972fb` passed `ci/full` in GitHub Actions run `30211902545`; the final documentation checkpoint `9f8ecc31f954ce65695aed0edc99614f296035a8` passed `ci/full` in run `30212060260`.
