FROM node:22-alpine AS build
WORKDIR /workspace
COPY clients/admin-web/package.json ./package.json
RUN npm install --no-audit --no-fund
COPY clients/admin-web/ ./
RUN npm run build

FROM nginxinc/nginx-unprivileged:1.29-alpine
COPY clients/admin-web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /workspace/dist /usr/share/nginx/html
EXPOSE 8080
