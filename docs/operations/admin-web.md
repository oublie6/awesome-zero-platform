# Admin web operations

## Local development

Start MySQL, Redis, schema, and `app-api`, then:

```bash
cd clients/admin-web
npm ci
npm run dev
```

Vite serves on port 5173 and proxies `/api` to `http://127.0.0.1:8888`. The local API CORS configuration also permits the Vite origin.

Run the Chromium browser suite and a production build with:

```bash
npm run build
npm test
```

The Playwright suite serves the production build with mocked `/api` responses, so it does not require a database or administrator credentials. Build before running tests. It covers the supported routes at desktop, tablet, and mobile widths and saves screenshots and diagnostics under `test-results/` (ignored by Git). CI builds the assets, installs Chromium, and runs this suite.

## First administrator

Set a random token of at least 32 characters in `APP_ADMIN_BOOTSTRAP_TOKEN`, start the API, open `/bootstrap`, and create the first administrator. Remove the environment variable immediately after successful bootstrap.

No default username or password is shipped.

## Production container

`deploy/docker/admin-web.Dockerfile` builds the client and serves it with an unprivileged Nginx container on port 8080. Nginx proxies `/api/` to the Compose service `app-api`, so production does not require browser CORS.

The production Compose stack exposes:

- Admin web: `http://localhost:8080`
- API: `http://localhost:8888`

For Kubernetes, `deploy/kubernetes/admin-web.yaml` provides a non-root deployment and service. Supply ingress and image names in the environment-specific overlay.

The Docker build uses Node 22.23.4 on Debian slim because the Vite/Rolldown native package requires its matching Linux binary; Alpine/musl does not provide that binary. The runtime remains the unprivileged Nginx image. The Vite build resolves Element Plus components and directives on demand to keep browser chunks small.
