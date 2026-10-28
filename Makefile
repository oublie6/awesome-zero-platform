GOCTL ?= goctl
SERVER_DIR := server
APP_API_DIR := $(SERVER_DIR)/apps/app-api
APP_API_SPEC := $(APP_API_DIR)/app.api
LOCAL_COMPOSE := docker compose -f deploy/local/docker-compose.yml
PRODUCTION_COMPOSE := bash scripts/production-compose.sh

.PHONY: generate run build test fmt fmt-check deps-up deps-down deps-reset schema-apply seed-apply integration-test production-config production-up production-down observability-config observability-up observability-status observability-forward observability-smoke

generate:
	cd $(SERVER_DIR) && $(GOCTL) api go --api apps/app-api/app.api --dir apps/app-api --style gozero

run:
	cd $(SERVER_DIR) && go run ./apps/app-api -f apps/app-api/etc/main-api.yaml

build:
	cd $(SERVER_DIR) && CGO_ENABLED=0 go build -p 1 -o ../build/app-api ./apps/app-api

test:
	cd $(SERVER_DIR) && go test -p 1 -parallel 1 ./...

fmt:
	cd $(SERVER_DIR) && find . -name '*.go' -type f -print0 | xargs -0 gofmt -w

fmt-check:
	@test -z "$$(cd $(SERVER_DIR) && find . -name '*.go' -type f -print0 | xargs -0 gofmt -l)" || \
		(echo "Go files require formatting; run make fmt" && exit 1)

deps-up:
	$(LOCAL_COMPOSE) up -d --wait

deps-down:
	$(LOCAL_COMPOSE) down --remove-orphans

deps-reset:
	$(LOCAL_COMPOSE) down --remove-orphans --volumes
	$(LOCAL_COMPOSE) up -d --wait

schema-apply:
	$(LOCAL_COMPOSE) exec -T mysql sh -c 'exec mysql --default-character-set=utf8mb4 -uapp_local -p"$$MYSQL_PASSWORD" awesome_zero_platform' < server/database/schema/current.sql

seed-apply:
	$(LOCAL_COMPOSE) exec -T mysql sh -c 'exec mysql --default-character-set=utf8mb4 -uapp_local -p"$$MYSQL_PASSWORD" awesome_zero_platform' < server/database/seed/development.sql

integration-test:
	cd $(SERVER_DIR) && APP_API_INTEGRATION=1 go test -count=1 -p 1 -parallel 1 -tags=integration ./...

production-config:
	$(PRODUCTION_COMPOSE) config

production-up:
	$(PRODUCTION_COMPOSE) up -d --build --wait

production-down:
	$(PRODUCTION_COMPOSE) down --remove-orphans

# Kubernetes-native, opt-in observability. This does not run Docker Compose.
# A populated K8s cluster, application Secret, MySQL/Redis Services and
# OBS_GRAFANA_ADMIN_PASSWORD are required for observability-up/smoke.
observability-config:
	bash scripts/k8s-observability.sh render

observability-up:
	bash scripts/k8s-observability.sh apply

observability-status:
	bash scripts/k8s-observability.sh status

observability-forward:
	bash scripts/k8s-observability.sh forward

observability-smoke:
	bash scripts/k8s-observability.sh smoke
