SHELL := /bin/sh
.DEFAULT_GOAL := help

APP         := v2home
VERSION     ?= 0.1.0
COMMIT      ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME  ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GOARCH      ?= amd64
GOARM       ?=
GOMIPS      ?= softfloat
OPENWRT_TAG ?= x86-64-24.10.0
DOCKER      ?= docker
NODE_IMAGE  ?= node:22-alpine
GO_IMAGE    ?= golang:1.23-alpine

export OPENWRT_TAG

BUILD_ARGS := \
	--build-arg VERSION=$(VERSION) \
	--build-arg COMMIT=$(COMMIT) \
	--build-arg BUILD_TIME=$(BUILD_TIME) \
	--build-arg TARGETARCH=$(GOARCH) \
	--build-arg GOARM=$(GOARM) \
	--build-arg GOMIPS=$(GOMIPS)

.PHONY: help dev down logs build build-openwrt test vet frontend-build lock openwrt-test clean

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage: make <target>\n\n"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@printf "\n"

dev: ## Run backend + frontend dev servers in Docker
	$(DOCKER) compose up --build

down: ## Stop dev containers
	$(DOCKER) compose down

logs: ## Follow dev container logs
	$(DOCKER) compose logs -f

build: ## Build the production image (GOARCH=$(GOARCH))
	$(DOCKER) build $(BUILD_ARGS) -f deploy/docker/Dockerfile -t $(APP):$(VERSION) .

build-openwrt: ## Export a static binary to dist/$(GOARCH) (GOARCH=mipsle|arm|arm64|amd64)
	rm -rf dist/$(GOARCH)
	$(DOCKER) build $(BUILD_ARGS) --target export --output type=local,dest=dist/$(GOARCH) -f deploy/docker/Dockerfile .

test: ## Run Go tests in a container
	$(DOCKER) run --rm -v "$(CURDIR):/work" -w /work/backend $(GO_IMAGE) go test ./...

vet: ## Run go vet in a container
	$(DOCKER) run --rm -v "$(CURDIR):/work" -w /work/backend $(GO_IMAGE) go vet ./...

frontend-build: ## Build the Vue bundle into backend/webui/dist
	$(DOCKER) run --rm -v "$(CURDIR):/work" -v v2home-node-modules:/work/frontend/node_modules -w /work/frontend $(NODE_IMAGE) sh -c "npm install --no-audit --no-fund && npm run build"

lock: ## Generate frontend/package-lock.json inside Docker
	$(DOCKER) run --rm -v "$(CURDIR):/work" -w /work/frontend $(NODE_IMAGE) npm install --package-lock-only --no-audit --no-fund

openwrt-test: ## Build amd64 binary and smoke-test it inside an OpenWrt 24.10 rootfs
	sh ./scripts/openwrt-test.sh

clean: ## Remove local build artifacts
	rm -rf dist
	rm -rf backend/webui/dist
	mkdir -p backend/webui/dist
	touch backend/webui/dist/.gitkeep
