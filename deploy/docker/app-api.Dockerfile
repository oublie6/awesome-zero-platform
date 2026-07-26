FROM golang:1.25.8-alpine AS build

WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -p 1 -trimpath -ldflags="-s -w" -o /out/app-api ./apps/app-api

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/app-api /app/app-api
COPY server/apps/app-api/etc/production.yaml /app/etc/production.yaml

EXPOSE 8888
USER nonroot:nonroot
ENTRYPOINT ["/app/app-api", "-f", "/app/etc/production.yaml"]
