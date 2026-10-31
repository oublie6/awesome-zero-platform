# Goal 0044: Admin Web quality and Kubernetes acceptance

## Status

- State: in_progress
- Started: 2026-10-31
- Completed:
- Blockers: Authenticated live browser acceptance is pending a known safe Admin test account. The live bootstrap endpoint reports `available: false`, and no test credentials were available in this run.

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

- Read the full `AGENTS.md`, this goal and all referenced architecture, API, operations, deployment, manifest and server-profile documents. Started from a clean, synchronized `main` at `bd18019`.
- Audited the Admin Web API client, token vault, auth store, route guards, layout/navigation, all eleven views, Element Plus usage, styles, Vite/build configuration, Docker/Nginx configuration and Kubernetes baseline.
- Fixed verified gaps across session refresh and logout cleanup, stale/cancelled reads, permission-filtered navigation/actions, route denial handling, error/retry states, account/role/authorization workflows, confirmations, form validation, keyboard focus, mobile navigation and responsive layouts. Kept server authorization authoritative and retained the existing API contract.
- Added deterministic Playwright coverage for login, concurrent expired-token refresh/retry, refresh failure, logout failure, route permissions, account CRUD/search/pagination/session revocation, roles/permissions, policy confirmation, Dashboard/bootstrap retries, mobile navigation and narrow dialog/table layouts.
- Reduced the initial JS bundle through route/component splitting and on-demand Element Plus registration. The production build typechecks and emits no >500 kB chunk warning; largest emitted JS asset is 86.17 kB (29.33 kB gzip).
- Added an isolated Admin Web server profile containing only the Admin Web Deployment and Service, one replica pinned to `hk-worker`, `Recreate`, `imagePullPolicy: Never`, and a full-source-SHA image. The cluster node has one filesystem-backed disk; this stateless UI uses node ephemeral storage and adds no PV, PVC or data mount.
- Built and imported the full-SHA image into the target node's containerd, deployed it, and verified rollout Ready at 1/1 on `hk-worker`. Updated after CI exposed a 768px profile overflow and an asynchronous refresh-test race; the tablet layout and polling assertion are fixed and the updated image is now deployed.
- Verified live `/health` (200), static JS (200), SPA `/accounts` fallback (200), Nginx `/api` proxy to the real bootstrap status endpoint (`available: false`), and unauthenticated `/admin/me`, `/admin/accounts`, `/admin/roles` and `/admin/audit/events` API calls (401). Real Chromium checked login/bootstrap at all six required widths with no console, page or request errors and no page-level horizontal overflow. Protected `/accounts` redirects to login.
- Confirmed `app-api` Deployment UID, generation and images are unchanged; PVC/PV inventory is unchanged; temporary image-import and browser-runtime containers/pods were removed. Rollback is documented, and the prior Admin Web ReplicaSet remains available to `kubectl rollout undo`.

### In progress

- Resumed on user request. Latest report-only checkpoint `2be84fd` exposed a Chromium bounding-box precision failure (`360.000015px` versus `360px`); fixing only the assertion tolerance to 0.001 CSS pixel while also checking both drawer edges. The integer document-width assertion stays exact. Its other 14 browser tests and Go jobs passed.
- Final live acceptance is limited to authenticated Admin Web flows. Bootstrap is disabled or already completed, and no safe existing Admin test credentials were available; no account was created and no production data was changed.

### Remaining

- Obtain an authorized, non-destructive Admin test account (or a secure way to use one) and run live login, permission, account/role/audit route and safe read-only operation checks. Do not create or mutate production accounts to bypass the unavailable credentials.
- Complete the goal only after that live authenticated acceptance passes; then archive Goal 0044 and reset the current-goal template.

### Verification status

- Resume verification: focused Chromium table/dialog/drawer test passed (1/1, 21.2s) after adding the subpixel precision tolerance and edge checks. The first local launch lacked the correct browser-cache environment path; rerunning with the existing Chromium cache succeeded. This change affects tests only; deployed application assets are unchanged. Full CI rerun follows the checkpoint push.
- Local `npm run build` (TypeScript typecheck + Vite production build): passed; no >500 kB chunk warning.
- Local Playwright Chromium: 15/15 passed, including all 11 routes across 1920x1080, 1440x900, 1024x768, 768x1024, 390x844 and 360x800; captured representative login, bootstrap, account, error, empty, modal and drawer states under ignored `clients/admin-web/test-results/`.
- GitHub Actions: commit `ba06f6dfe4062de920b2cfb07707e9ac77b844f3` initially failed the refresh-count timing assertion and found `/profile` overflow at 768px. Fixed both in `30db007875f7318fdcc748366042d63e3e3ddbf0`; its CI run `41162217250` completed successfully, including Admin Web browser tests, Go unit/race, integration, secure-envelope and Production Compose runtime acceptance.
- Kustomize render and live diff: exactly the intended Deployment and Service; final diff is empty after applying the profile. Deployment image tag is the full source SHA `30db007875f7318fdcc748366042d63e3e3ddbf0`; containerd image ID is `sha256:5bf67c73071caeadc7a46342cd4cc077ae9d04ccf305c9d5665a4b14e2cd11b5`.
- Live cluster HTTP/Nginx and real-Chromium checks: passed for health, static asset, history fallback, API proxy and safe public login/bootstrap flow at all six widths. Authenticated routes and operations remain unverified because no test credentials were available; mocked browser tests do not substitute for those checks.
- Before/after safety checks: `app-api` remains UID `f04fe952-7afa-49b0-8efe-0daa92c30705`, generation 6 and the same app-api/Filebeat images; PVC/PV inventory unchanged. No MySQL, Redis, Elasticsearch, Jaeger or persistent volume resources were modified.
- `git diff --check`: passed. All Actions for server-profile/report checkpoint `60984449420ed1f85fdf46c30fc80af0dfd5c2f6` passed: CI `41163066284` (including browser, unit/race, integration and runtime), Fair Doudizhu `41163066272`, and Kubernetes Observability Manifests `41163066331`. The original Go setup steps eventually completed without restarting the runs.

## Completion Report

Goal is blocked pending authenticated live acceptance. The same missing test-credential condition was confirmed across three consecutive goal turns; no authorized test account or secure credential source has been provided. Changes are reusable client integration and deployment configuration; no backend/module/database contracts were changed. Source commits pushed to `main`: `ba06f6d` and `30db007`; deployment/report checkpoint `6098444` is pushed and all its Actions passed. The corrected source is deployed from its full SHA. Do not archive or mark the goal completed until the remaining live authenticated checks pass.

Self-review covered auth/token state transitions, coalesced refresh and retry ordering, stale requests and abort cleanup, API payloads against the documented contract, route/action presentation guards, validation boundaries, destructive-action confirmations, recovery/error paths, mobile/tablet bounds, Docker/Nginx assets, exact rendered Kubernetes resources, node placement and rollback. The second-pass review fixed the 768px two-column overflow and made the refresh test wait for the asynchronous retry. No frontend condition is treated as an authorization boundary. No production account operation was performed.

Remaining uncertainty: the live login and authorized account/role/audit workflows could not be exercised without an authorized test account; server-side access was verified to return 401 without credentials. Local simulated-browser coverage and live unauthenticated checks are reported separately.
