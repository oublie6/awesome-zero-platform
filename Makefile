GOCTL ?= goctl
SERVER_DIR := server
APP_API_DIR := $(SERVER_DIR)/apps/app-api
APP_API_SPEC := $(APP_API_DIR)/app.api
LOCAL_COMPOSE := docker compose -f deploy/local/docker-compose.yml

.PHONY: generate run test fmt deps-up deps-down deps-reset schema-apply seed-apply integration-test

generate:
	cd $(SERVER_DIR) && $(GOCTL) api go --api apps/app-api/app.api --dir apps/app-api --style gozero

run:
	cd $(SERVER_DIR) && go run ./apps/app-api -f apps/app-api/etc/main-api.yaml

test:
	cd $(SERVER_DIR) && go test -p 1 -parallel 1 ./...

fmt:
	cd $(SERVER_DIR) && find . -name '*.go' -type f -print0 | xargs -0 gofmt -w

deps-up:
	$(LOCAL_COMPOSE) up -d --wait

deps-down:
	$(LOCAL_COMPOSE) down --remove-orphans

deps-reset:
	$(LOCAL_COMPOSE) down --remove-orphans --volumes
	$(LOCAL_COMPOSE) up -d --wait

schema-apply:
	$(LOCAL_COMPOSE) exec -T postgres psql -U app_local -d awesome_zero_platform < server/database/schema/current.sql

seed-apply:
	$(LOCAL_COMPOSE) exec -T postgres psql -U app_local -d awesome_zero_platform < server/database/seed/development.sql

integration-test:
	cd $(SERVER_DIR) && APP_API_INTEGRATION=1 go test -count=1 -p 1 -parallel 1 -tags=integration ./...
