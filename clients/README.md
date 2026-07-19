# Clients

Client applications consume a shared server API while keeping platform-specific implementation details isolated.

Planned client categories:

- `admin-web/` — Vue 3 administration client
- `miniapp/` — WeChat Mini Program
- `h5/` — mobile web client
- `app/` — native or cross-platform app client
- `shared/` — API schemas, generated client types, error codes, and design tokens

Only create a client directory when that client is actually under development. Avoid committing empty application scaffolds for speculative future clients.
