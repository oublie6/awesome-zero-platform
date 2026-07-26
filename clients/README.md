# Clients

Client applications consume the platform API while keeping user experience and platform-specific implementation details isolated.

Current client:

- `admin-web/` — Vue 3 administration control plane with account, role, standard authorization, expert authorization, audit, and operations views.

Planned client categories:

- `user-web/` — independent Vue 3 user-facing client;
- `miniapp/` — WeChat Mini Program;
- `h5/` — mobile web client;
- `app/` — native or cross-platform app client;
- `packages/` — stable API client, authentication, contracts, permission helpers, and small pure utilities shared only when reuse is proven.

The Admin layout, tables, menus, and Element Plus components are not shared with future user clients. Authentication and API boundaries are designed for extraction into `clients/packages` when the second client begins.
