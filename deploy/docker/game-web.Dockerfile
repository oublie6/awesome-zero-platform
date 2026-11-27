FROM nginxinc/nginx-unprivileged@sha256:a6c4f61f456b85b8fdf7ec7ab28cc3e299440e6fb4a9dea520e5fd8fd440025e
ARG APP_REVISION
ARG PACKAGING_REVISION
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY cache-policy.nginx.conf /etc/nginx/conf.d/00-cache.conf
COPY gzip-static.nginx.conf /etc/nginx/static-gzip.conf
COPY precompression-manifest.json /etc/nginx/precompression-manifest.json
COPY site /usr/share/nginx/html
LABEL org.opencontainers.image.revision="${APP_REVISION}" org.awesome-zero.packaging-revision="${PACKAGING_REVISION}" org.awesome-zero.web-packaging="gzip-precompressed-v1"
EXPOSE 8443
