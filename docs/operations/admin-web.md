# Admin web operations

## Local development

Start MySQL, Redis, schema, and `app-api`, then:

```bash
cd clients/admin-web
npm install
npm run dev
```

Vite serves on port 5173 and proxies `/api` to `http://127.0.0.1:8888`. The local API CORS configuration also permits the Vite origin.

## First administrator

Set a random token of at least 32 characters in `APP_ADMIN_BOOTSTRAP_TOKEN`, start the API, open `/bootstrap`, and create the first administrator. Remove the environment variable immediately after successful bootstrap.

No default username or password is shipped.

## Production container

`deploy/docker/admin-web.Dockerfile` builds the client and serves it with an unprivileged Nginx container on port 8080. Nginx proxies `/api/` to the Compose service `app-api`, so production does not require browser CORS.

The production Compose stack exposes:

- Admin web: `http://localhost:8080`
- API: `http://localhost:8888`

For Kubernetes, `deploy/kubernetes/admin-web.yaml` provides a non-root deployment and service. Supply ingress and image names in the environment-specific overlay.
