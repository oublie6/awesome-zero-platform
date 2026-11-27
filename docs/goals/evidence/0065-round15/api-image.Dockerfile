FROM awesome-zero-platform/goal0063-api:9581630
COPY app-api /app/app-api
COPY runtime/lib/x86_64-linux-gnu/libc.so.6 /lib/x86_64-linux-gnu/libc.so.6
COPY runtime/lib64/ld-linux-x86-64.so.2 /lib64/ld-linux-x86-64.so.2
LABEL org.opencontainers.image.revision="efc0699b0d62ae1284db86121619a746ce4f37ba" org.awesome-zero.server-tree="a8f14fec9d7703c0fe68ac286bbd4aa9081fefbe"
