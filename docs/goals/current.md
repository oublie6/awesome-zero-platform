# Goal 0044: Admin Web quality and Kubernetes acceptance

## Status

- State: in_progress
- Started: 2026-10-31
- Completed:
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

- Completed the referenced architecture/API/source/deployment audit and all eleven Admin Web routes. Kept platform/business boundaries and server API contracts intact; changes are client integration and deployment configuration.
- Improved token refresh/logout cleanup, auth-state concurrency, route/action permission presentation, stale-request cancellation, errors/retries, account and role workflows, confirmations, form validation, focus/touch behavior and mobile/tablet layouts.
- Added 15 deterministic Chromium tests and CI artifact preservation. Fixed failures from asynchronous refresh timing, 768px profile overflow, Chromium subpixel bounds, drawer entry animation and reused table rows during creation refresh. Retained exact account-ID request checks and document-width checks.
- Typechecked and built production assets. Replaced global Element Plus registration with maintained on-demand resolution and route splitting; largest JS asset is 86.17 kB (29.33 kB gzip), with no >500 kB warning.
- Deployed only Admin Web Deployment/Service through the server profile; one replica on hk-worker, Recreate, node-local full-SHA image, no registry or PV/PVC requirement. Health, static assets, SPA fallback and Nginx API proxy passed.
- Resolved the credential blocker after the user explicitly requested a new administrator. Used existing public Identity/Admin services and clustered authorization coordinator for the one-time local operator action, preserving password hashing, audit and policy synchronization. No direct cross-module SQL or fabricated access token was used.
- Created azp_acceptance_0044, account ID 01a1907b-df95-7828-bc3b-dc68b96fb4ff, role goal0044_acceptance_20261031 with GET-only /admin/* permission. Random credentials remain in /root/.config/awesome-zero-platform/goal0044-admin.json (0600, parent directory 0700). No credential values are committed or included in reports.
- Completed live authenticated Chromium acceptance: 11 routes × six exact required viewport sizes = 66 checks, real API reads, account search/detail, role/permission and audit views, hidden write controls, backend 403 for a write attempt, same-tab reload, independent-tab session isolation and logout cleanup. Console/page errors and failed network requests were empty.
- Verified actual browser zoom through chrome.tabs.setZoom at 125% (devicePixelRatio 1.25, layout width 1152) and 200% (devicePixelRatio 2, width 720); document width matched the effective layout viewport. Captured loading state by holding a real overview request locally, then forwarding its response unchanged and observing loading clear.
- Rechecked Admin Web live diff (empty), app-api UID/generation/images (unchanged), PVC/PV bindings (unchanged), and successful rollback revision/image retention. User-authorized provisioning changed only the dedicated account/role/policy and associated audit records, through module services.

### In progress

- Final self-review found account form rules missing matching Element Plus prop bindings for username/displayName/email/phone. The premature archive is reverted while preserving the completed live evidence. Bind the four fields, add a regression proving invalid inputs prevent writes, rebuild/redeploy the source-SHA image and repeat affected live acceptance before final archival.

### Remaining

- Verify the account-validation regression and full browser suite; commit the corrected source, build/import/deploy its full-SHA image, repeat affected live acceptance, then archive/reset and verify final Actions.

### Verification status

- Reopened account-form correction: production typecheck/build passed, and all 16 local Chromium tests passed (4.5m), including the new invalid-input/no-write regression and the six-viewport route matrix. The final archive was reverted before declaring the thread goal complete; updated source deployment/CI is pending.
| Scope | Evidence |
| --- | --- |
| Local typecheck/build | npm run build passed, 1,721 modules, no >500 kB warning; source image build also passed |
| Local mocked browser | 15/15; all 11 routes at 1920x1080, 1440x900, 1024x768, 768x1024, 390x844, 360x800; focused account and drawer regressions passed after fixes |
| CI source/test checkpoint | 585dbab4a3e16218cd8cacd028604f78a86dfaff: CI 41172033543 and Fair Doudizhu 41172033666 passed, including browser, unit/race, integration and Production Compose runtime |
| Latest pre-report checkpoint | bf9b838a4119247ddaa6857ebf47d3c6d3a20952: CI 41172535213 and Fair Doudizhu 41172535203 passed |
| Browser artifact | 13031465926, admin-web-browser-results, 18 screenshots, 7-day retention; downloaded and visually reviewed. It contains simulated API evidence, not live cluster results |
| Live cluster/browser | Secure local evidence directory /root/.config/awesome-zero-platform/goal0044-evidence (0700, files 0600), report.json and supplement.json, 66 route/viewport checks, authenticated API/UI and session flows, native zoom and loading screenshots |
| Live report hashes | report.json SHA256 71783a040515a25da9064ffdedb31e590239cdef4238b3937dd86f18e754d6c4; supplement.json SHA256 0d7ebe56a84c7bf67796392af89f4a7dd435cce2221465eb67ee5ce4864c9d9d |
| Kubernetes | Admin Web Ready 1/1 on hk-worker; final kustomize diff empty; /health, JS asset, history route and API proxy return 200 |
| Deployment safety | app-api UID f04fe952-7afa-49b0-8efe-0daa92c30705, generation 6, original app-api/Filebeat images; existing PVC/PV bindings remain Bound and unchanged |

## Completion Report

Reopened during final self-review: the account form's existing validation rules were not bound to four fields. The previous archive commit is being reverted, the bindings and regression coverage are being corrected, and the report below is retained as evidence for the prior deployed checkpoint until the corrected source is redeployed and verified.

The Admin Web quality and deployment acceptance is complete. Final publication consists only of this report, archival and current-goal reset; its Actions result is checked after pushing and reported in the final response.

### Changed files and rationale

- clients/admin-web/src/api, stores/auth.ts and router/index.ts: single-flight refresh, stale-token retry, logout cleanup, auth generation guards, abortable reads, permission-filtered routes and explicit 403 handling.
- Layout, account actions, global styles and views: reachable mobile navigation, tablet cards, table/dialog/drawer bounds, form validation, pending states, confirmations and recoverable API failures. Added ForbiddenView.vue.
- Vite/main/components declarations, package.json and lockfile: on-demand Element Plus, maintainable route/component splitting and reproducible dependencies. Docker build uses pinned Node 22.23.4 Debian slim to support the Vite native package.
- Playwright configuration/tests and CI: production-preview browser testing, required behavioral categories and viewport matrix; synchronization fixes retain meaningful assertions; preserve screenshots/diagnostics as CI artifacts. Operations docs describe the build/test behavior.
- deploy/kubernetes/overlays/admin-web-server: isolated one-replica profile and explicit rollback procedure. No backend source, schema or unrelated deployment configuration changed.

### Deployment and rollback

Before the goal, no Admin Web Deployment or Service existed. Current image is awesome-zero-platform/admin-web:30db007875f7318fdcc748366042d63e3e3ddbf0, the full application source SHA; running image ID is sha256:5bf67c73071caeadc7a46342cd4cc077ae9d04ccf305c9d5665a4b14e2cd11b5. Later commits only change tests, CI and reports, so rebuilding runtime assets was unnecessary. The target node has one filesystem-backed disk; Admin Web is stateless and uses node ephemeral storage, adding no data mount or persistent volume.

The prior replica revision and imported image tag awesome-zero-platform/admin-web:ba06f6d1d971fb7339cc4c2e7cfac0897d18bcd5 are retained for kubectl rollout undo deployment/awesome-zero-admin-web. That interim tag is recorded as the exact prior manifest tag, not asserted to be a valid source commit SHA. To restore the pre-goal absent state, delete only this profile's Admin Web Deployment/Service; never delete the namespace or apply the default two-replica base. See the server-profile README for commands.

### Live functional result and limits

The dedicated read-only test administrator logged in through the actual Nginx/API/authentication/session path. All protected pages loaded at six viewports; account filtering/detail, role permissions, audit and system reads succeeded. Write controls were unavailable and an actual POST to the account endpoint returned 403. This is expected authorization isolation for the GET-only test role. Existing account data was not edited, disabled or deleted. Production mutations such as password reset, session revocation and raw-policy replacement were intentionally not executed against existing records; deterministic mocked tests cover those paths. User authorization permitted the new acceptance account and dedicated role; they remain active for future read-only revalidation. Both test sessions were logged out, and the password can be obtained from its 0600 credential file.

### Self-review and residual operational considerations

This was self-review by the implementing agent, not independent review. Reviewed API/DTO compatibility, token/session transitions, sibling refresh ordering, abort and stale-response cleanup, form boundaries, dangerous-operation cancellation, permission presentation versus server enforcement, module ownership, secret handling, responsive/zoom bounds, build resources, exact Kubernetes scope, lifecycle cleanup and rollback. CI traces exposed timing/precision assumptions and the wrong-row test race; these were fixed without deleting UUID-specific assertions or hiding overflow. Live 403 checks confirm frontend hiding does not replace server authorization.

No acceptance blocker remains. Credentials and live screenshots stay outside Git with restricted permissions; CI screenshots expire after seven days. Node-local image placement requires re-importing the pinned image if the node is replaced. The account is deliberately read-only; granting write rights later is a separate authorized operation. Temporary browser/provisioning processes and port forwarding are stopped after verification. No unsupported external integrations were claimed tested.
