FROM awesome-zero-platform/goal0065-web:efc0699
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY gzip-static.nginx.conf /etc/nginx/static-gzip.conf
LABEL org.awesome-zero.web-gzip-config-sha256="aa6bf84f13e614d1bc4e36b7289e2bc0e90fe56c19f642bb9e19fd0f0fec21e2"
