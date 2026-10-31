# Goal 0044: Admin Web quality and Kubernetes acceptance

## Status

- State: in_progress
- Started: 2026-10-31
- Completed:
- Blockers:

## Goal

Audit and improve the existing Awesome Zero Platform Admin Web across UI, responsive behavior, API/session correctness, accessibility, interaction recovery, performance and maintainability; prove the result with automated tests and real Chromium checks at the required viewport sizes; then safely deploy a full-Git-SHA image to the existing Kubernetes integration cluster and verify the live application without changing unrelated backend or data services.

## References

- `AGENTS.md`
- `docs/architecture/admin-platform.md`
- `docs/api/admin.md`
- `docs/operations/admin-web.md`
- `docs/operations/production-deployment.md`
- `deploy/README.md`
- `deploy/kubernetes/admin-web.yaml`
- `deploy/kubernetes-overlays/observability-server/README.md`
- `clients/admin-web/`
- `deploy/docker/admin-web.Dockerfile`
- `clients/admin-web/nginx.conf`

## Deliverables

1. Audit the Admin Web layouts, all views, components, API client/types, Pinia store, router, global styles, build configuration, Docker/Nginx assets, Kubernetes manifests, and current live deployment. Reuse existing Element Plus capabilities and avoid unrelated business/backend changes.
2. Fix verified UI, responsive, interaction, API/session, authorization presentation, accessibility, and maintainability defects. Preserve server-side authorization as authoritative and do not introduce HttpOnly-cookie authentication in this goal.
3. Add deterministic automated coverage for login/token expiry, API errors/retry, routing permissions, account search/pagination/create/edit/disable, role/permission operations, dangerous-action confirmations, Dashboard failures, mobile navigation, and responsive table/modal/drawer behavior.
4. Validate all listed Admin Web routes at 1920x1080, 1440x900, 1024x768, 768x1024, 390x844, and 360x800 with real Chromium. Capture key pages and representative loading, empty, error, dialog/drawer states. Inspect console/network failures, horizontal overflow and DOM bounds. Clearly distinguish simulated-browser results from live-cluster browser results.
5. Correct build/type/test issues and the existing >500 kB chunk warning using maintainable route/component splitting without increasing warning thresholds or harming initial load.
6. Safely deploy a traceable Admin Web image built from the full source SHA via the server-specific Kubernetes profile. Verify Nginx `/api` proxy and SPA fallback, healthy deployment, static assets, API reachability, and live browser flows. Do not modify app-api, MySQL, Redis, Elasticsearch, Jaeger, or persistent volumes.
7. Record changed files, test and browser evidence, before/after image SHA, deployment status, rollback procedure, limitations and mandatory self-review in this goal report; archive Goal 0044, reset the current template, commit and push to `main`.

## Constraints

- Follow `AGENTS.md`; stay on clean, synchronized `main` and preserve existing work.
- Do not duplicate work already delivered by Admin Web commit `bd18019b` or expand scope into unrelated business/backend changes.
- Do not add another UI framework or large dependency without a concrete need. Do not hide overflow globally or merely raise chunk warning limits.
- Keep API contracts and backend authorization intact. The browser is not an authorization boundary.
- Do not modify app-api, MySQL, Redis, Elasticsearch, Jaeger configuration, or their persistent volumes during deployment.
- Review exact rendered Kubernetes changes, use the server-specific profile, identify a previous-image/manifest rollback path, and stop if the environment cannot be changed safely.
- Do not claim real-cluster browser behavior based only on local/simulated browser runs.

## Acceptance Criteria

- Repository inventory covers every named Admin Web subsystem and all eleven required routes; verified findings and fixes are documented.
- All six specified viewport sizes are checked with real Chromium. Relevant screenshots and evidence confirm no unintended page-level horizontal overflow, clipped controls, inaccessible navigation, broken dialogs/drawers, or out-of-bounds content. Console and network are reviewed.
- Automated tests cover all ten required behavioral categories: authentication/token expiry; API failure/retry; route authorization; account list/search/pagination and mutations; role/permission operations; dangerous-action confirmation; Dashboard error state; mobile nav; and responsive tables/dialogs/drawers.
- TypeScript typecheck, production build, unit/browser test suites, and repository CI pass. Any chunk warning above 500 kB is resolved through justified splitting and verified in build output.
- API endpoint/method/payload/response and token refresh/session handling match the documented current server contract, including single-flight refresh and logout cleanup. No frontend check is treated as server authorization.
- Server overlay render and live diff show only intended Admin Web resources. Image tag contains the full source SHA, is available on the target node, and does not use floating `latest` for the updated rollout.
- Deployment reaches Ready; `/health`, static assets, SPA history fallback and `/api` proxy work. Live Chromium validates login/bootstrap as safely available, permissions, account/role/audit routes and key operations without destructive production-data changes.
- Rollback is documented and remains executable. Other service deployments, configuration and persistent volumes remain unchanged.
- Goal report distinguishes local tests, simulated Chromium evidence, live Kubernetes/browser evidence, unavailable checks and risks; self-review is complete; final report is archived and pushed to `main`; required Actions are green.

## Agent Strategy

Primary agent owns implementation, test design/execution, deployment, integration, self-review and final verification. Keep resource-heavy commands sequential.

## Working State

### Completed

- Read the full `AGENTS.md` and current-goal template; confirmed Goal 0043 is archived and no other current goal was active.
- Confirmed a clean `main` at `43da912`, then ran `git pull --ff-only`; synchronized to `bd18019` with clean worktree.
- Read the Admin platform architecture, Admin API contract, Admin Web operations/deployment instructions, Kubernetes Admin Web manifest and server profile docs. Confirmed the known latest Admin Web change at `bd18019b` is a mobile sidebar fix.

### In progress

- Source/deployment inventory and gap analysis. Confirmed live cluster currently has no Admin Web Deployment or Service, so deployment needs an isolated server profile rather than applying the default two-replica base.
- Checking available local/containerized Node and Chromium tooling, and inspecting all screens, layouts, API/session code, Docker/Nginx and deployment assets.

### Remaining

- Complete source/API/build/deploy audit and resolve browser tool availability.
- Implement and verify fixes and automated coverage.
- Run real Chromium viewport/state acceptance, correct issues, deploy SHA-pinned image and run live acceptance.
- Complete self-review, archive goal, commit/push and verify CI.

### Verification status

- Initialization, repository state and referenced documentation: passed.
- Frontend, browser and Kubernetes acceptance: not started.

## Completion Report

Not started.
