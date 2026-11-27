FROM nginxinc/nginx-unprivileged@sha256:a6c4f61f456b85b8fdf7ec7ab28cc3e299440e6fb4a9dea520e5fd8fd440025e
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY cache.conf /etc/nginx/conf.d/00-cache.conf
COPY site /usr/share/nginx/html
LABEL org.opencontainers.image.revision="efc0699b0d62ae1284db86121619a746ce4f37ba" org.awesome-zero.cocos-source="eda1a70b1ec504d6588d79203c50d9a08416bd8c"
EXPOSE 8443
