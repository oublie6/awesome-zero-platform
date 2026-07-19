GOCTL ?= goctl
SERVER_DIR := server
APP_API_DIR := $(SERVER_DIR)/apps/app-api
APP_API_SPEC := $(APP_API_DIR)/app.api

.PHONY: generate run test fmt

generate:
	cd $(SERVER_DIR) && $(GOCTL) api go --api apps/app-api/app.api --dir apps/app-api --style gozero

run:
	cd $(SERVER_DIR) && go run ./apps/app-api -f apps/app-api/etc/main-api.yaml

test:
	cd $(SERVER_DIR) && go test ./...

fmt:
	cd $(SERVER_DIR) && find . -name '*.go' -type f -print0 | xargs -0 gofmt -w
