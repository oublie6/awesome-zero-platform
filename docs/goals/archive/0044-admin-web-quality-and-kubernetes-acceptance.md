# Goal 0044: Admin Web quality and Kubernetes acceptance

## Status

- State: completed
- Started: 2026-10-31
- Completed: 2026-10-31
- Blockers: None. The user explicitly authorized a new acceptance account; live authenticated acceptance passed.

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

- Audited every named client subsystem and all eleven required routes against the referenced architecture, API and deployment contracts. Changes are client integration and deployment configuration; no platform/business dependencies or server contracts changed.
- Corrected responsive navigation, tablet/profile grids, tables/dialogs/drawers, accessible controls, form validation, pending/empty/error states, retries, confirmations and permission presentation. Kept server authorization authoritative.
- Corrected refresh single-flight, stale-response/session cleanup, cancellation and authentication-operation ordering. Added deterministic regressions for late refresh after logout and after a subsequent login.
- Completed production typecheck/build, 18 automated Chromium tests, final-source repository CI, full live browser acceptance, native browser zoom/loading checks, and two live refresh-race checks.
- Built/imported/deployed the final application source 776be5e4e478a85016fb8b8caff57d94505dcab8. Only the Admin Web Deployment/Service changed; live profile diff is empty and deployment is Ready 1/1.
- Created the explicitly user-authorized dedicated read-only acceptance administrator through public module services, preserving password hashing, audit and authorization synchronization. Credential values remain outside Git.
- Completed a distinct second-pass self-review, fixed its findings, reverified, and cleaned up temporary image import resources, browser containers and port forwarding. Resource usage is healthy; caches and rollback images are retained as requested by the user.

### Remaining

No implementation or live-acceptance work remains. Publish the archive/current-template reset and pinned deployment profile on main, then verify the publication commit's Actions. Final publication SHA and Actions result are reported in the closing response, avoiding a self-referential report commit.

### Verification status

| Scope | Evidence |
| --- | --- |
| Local typecheck/build | Production npm run build passed, including TypeScript checks. Largest JavaScript asset 86.17 kB (29.32 kB gzip), entry 57.47 kB; no >500 kB warning. Full-SHA Docker build passed. |
| Local simulated Chromium | Complete 18/18 suite passed in 5.4m after session-race fixes. Subsequent validation-compatibility-only changes passed build and both affected account/role tests (2/2, 31.3s). All eleven routes covered at 1920x1080, 1440x900, 1024x768, 768x1024, 390x844, 360x800. |
| Final application-source Actions | Source 776be5e4e478a85016fb8b8caff57d94505dcab8: CI 41176360143 and Fair Doudizhu 41176360258 completed successfully. CI includes all 18 Admin Web tests, build/typecheck, backend unit/race/build, secure envelope, integration and Production Compose runtime checks. |
| CI browser artifact | 13032831647, admin-web-browser-results, 7-day retention. CI evidence uses simulated API responses. Earlier successful artifact screenshots were downloaded and visually reviewed; live evidence is separate. |
| Live Chromium | Final source, real Nginx/API/auth/session path: 11 routes × six viewports = 66 checks, no page overflow, console/page errors or failed requests. Account search asserts response total=1, exact acceptance account ID and one rendered row before screenshots. Reload, independent-tab isolation and logout passed. |
| Live zoom/loading | Native chrome.tabs.setZoom 125% and 200%: DPR 1.25/2, effective layout widths 1152/720, document widths equal. Visible loading captured by temporarily holding a real overview request, forwarding unchanged and verifying completion. |
| Live session races | Both delayed real-refresh scenarios passed: cannot restore logout; cannot replace a later login. Concurrent 401 requests produced one refresh. Real upstream refresh returned 200. Test-owned sessions explicitly logged out with 200 responses. |
| Kubernetes | Ready 1/1 on hk-worker; final kubectl diff -k server profile empty; health/static/history/API proxy passed. Running image ID matches the built/imported image. No import helper Pods remain. |
| Safety | app-api UID f04fe952-7afa-49b0-8efe-0daa92c30705, generation 6, original app-api/Filebeat images. Existing MySQL/Redis/Elasticsearch/Prometheus/Jaeger PVC bindings remain Bound and unchanged. |

## Completion Report

### Changes and findings

- clients/admin-web/src/api, stores/auth.ts and router/index.ts: single-flight refresh, stale-token retry, logout cleanup, version guards, abortable reads, permission-aware routing and explicit forbidden state. Authentication generations reject refresh success/failure/finalizers belonging to previous sessions, including retries and stale logout cleanup.
- Layout, shared actions, global styles and views: reachable mobile navigation, tablet cards, bounded tables/dialogs/drawers, readable identifiers, focus/touch behavior, form validation, pending states, dangerous-action confirmation and recoverable failures. Added ForbiddenView.vue.
- Self-review found four account validation rules without matching Element Plus prop bindings. Bound username/displayName/email/phone and added a regression proving invalid fields issue no write and valid input issues exactly one correctly shaped write.
- Contract review removed unsupported client restrictions: local-domain email validation@localhost and dotted/hyphenated role code support-operator.v2 are supported by the owner APIs. Basic email/ASCII/length, trimmed phone and non-whitespace display-name checks remain; full email parsing and persistence constraints are server-authoritative.
- Vite/main/components declarations, package.json/lockfile: maintained on-demand Element Plus resolution and route splitting replace global registration. Docker uses pinned Node 22.23.4 Debian slim for compatible native Vite packages. No chunk warning threshold was increased.
- Playwright configuration/tests, CI and operations docs: production-preview testing covers authentication/expiry, errors/retry, route permissions, account queries/pagination/mutations, role/permission flows, confirmations, Dashboard failures, mobile navigation and responsive bounds. CI preserves browser evidence. Fixed synchronization for refresh completion, subpixel bounds, drawer animation and reused table rows without dropping exact account-ID assertions.
- deploy/kubernetes/overlays/admin-web-server: isolated one-replica Recreate profile, node-local pinned image, imagePullPolicy Never, restricted container and ephemeral writable directories. No backend code, schema or unrelated runtime configuration changed.

### Deployment and rollback

Before this goal, Admin Web Deployment and Service were absent. The previous accepted application source was 30db007875f7318fdcc748366042d63e3e3ddbf0. Final deployed image is awesome-zero-platform/admin-web:776be5e4e478a85016fb8b8caff57d94505dcab8, built from that complete source commit. Image ID is sha256:97cb3bad9715212d426ef72cc38dc388c14740c226cc6e59e1653c16a735316a; containerd manifest digest is sha256:e4124c19c2266864c7f9a1df16794e7954112be3e2e891027419227819e34833.

Deployment UID ca6209bf-f24d-42b8-9c34-c66d67ec65e0, generation 3, is Ready 1/1; Pod awesome-zero-admin-web-58f8c655d-zql29 runs on hk-worker. Final publication changes only the manifest pin and reports, so its commit SHA differs from the deployed application-source SHA.

Revision 2 and node-local image awesome-zero-platform/admin-web:30db007875f7318fdcc748366042d63e3e3ddbf0 are retained. Roll back with kubectl -n awesome-zero-platform rollout undo deployment/awesome-zero-admin-web while that revision is retained, then reconcile the manifest pin. To restore the pre-goal absent state, delete only this profile's Deployment/Service with kubectl delete -k deploy/kubernetes/overlays/admin-web-server. Never delete the namespace or apply the default two-replica base as rollback. The target node has one filesystem-backed disk; this stateless application adds no PV/PVC/data mount.

### Live functional evidence and limits

The user explicitly authorized creating a test administrator after existing credentials blocked acceptance. A one-time local operator program invoked existing public Identity/Admin services and the clustered authorization coordinator, with normal Argon2 hashing and audit; no raw cross-module SQL, forged JWT or modification of existing accounts was used.

Account azp_acceptance_0044 has ID 01a1907b-df95-7828-bc3b-dc68b96fb4ff and role goal0044_acceptance_20261031 with GET /admin/* only. Credentials remain in /root/.config/awesome-zero-platform/goal0044-admin.json, mode 0600, parent 0700; no passwords/tokens are committed or logged. The account remains active for later read-only checks.

Real login, protected pages, account filtering/detail, role/permission views, audit/system reads, same-tab reload, independent-tab isolation and logout all passed. Write controls are hidden, and an actual account POST with the read-only token returned 403 as expected. Existing records were not edited/disabled/deleted. Password resets, arbitrary session revocations and policy replacement against existing production records were deliberately excluded; simulated browser tests cover those UI write paths, and Actions covers backend tests. Bootstrap layout/state was inspected without attempting to bootstrap an already initialized system.

The live race checks used an invalid access token to trigger actual backend 401 responses and the valid refresh token to obtain a real 200 refresh response. Delivery was held until after logout or a subsequent login. This verifies the real refresh/cleanup path, not a claim that a naturally expiring JWT was left running for its full lifetime. Expected 401s in these fault scenarios are recorded separately from the clean 66-check run. All sessions owned by these checks were logged out; no unrelated sessions were revoked.

Private evidence directory /root/.config/awesome-zero-platform/goal0044-evidence (0700; files 0600) contains final reports, screenshots and operator/browser scripts. Earlier 30db007 evidence is retained in checkpoint-30db007; it is not presented as the final source result. Final report SHA256 values:

- report.json: a6ddc44c1ab4a78544bd19ad0e19219cce6cd50f637b3230d02cf7c7c782e6a1
- supplement.json: 29dcb159a090f77b9e42ce7d80ab2647dd02d286559a925b24cfe0a1e34e7de7
- live-refresh-races.json: 8e9566f64832bf486f09aacbfc6e7b6723c97db37afa6542132bfdab7b6966fb

### Resource use and cleanup

At inspection, local available RAM was about 4.3 GiB with zero swap use; root/data disks were 43%/44% used. hk-worker memory working set was about 2.7 GiB with 5.05 GiB available; filesystem used 24.65 GiB of 96.73 GiB (about 26%, 72.07 GiB available). No pressure justified cache pruning. Per the user's follow-up, retain build/browser caches and rollback images. Removed temporary image transfer archives and import/helper Pods, stopped owned browser containers and port forwarding, and left existing services running. Restricted credentials and acceptance evidence are intentionally retained.

### Mandatory self-review

This was self-review by the implementing agent, not independent review. Re-read the goal/contracts, reviewed the integrated diff and traced successful login/read flows plus API failure/retry, forbidden writes, refresh failure, logout/new-login races and cleanup. Checked empty/boundary form inputs, stale reads, single-flight/version ordering, old-promise finalizers, retry cancellation, confirmation behavior, server versus frontend authorization, module ownership, secrets, responsive/zoom bounds, build resources, deployment scope, shutdown and rollback.

The review found the missing form bindings and two late-refresh defects. Both race regressions failed before correction (2/2) and passed afterward; strengthened account validation and compatibility regressions also passed. Rebuilt/redeployed the corrected source and repeated live acceptance before archival. Final screenshot review strengthened live account search evidence to await the specific query response, loading completion and one rendered matching row, preventing an old visible row from yielding a false pass.

No acceptance blocker remains. No independent or external supplementary executor was used. Live mutation limits above are intentional safety boundaries, not simulated results relabeled as live tests. Browser evidence has no unexpected console/network failures; CI artifacts expire after seven days. Node replacement requires re-importing the pinned local image. Changing the acceptance account to permit writes requires a separate authorized operation.
