SHELL := /bin/zsh

WEB_DIR := apps/skills-web

GO_CACHE ?= /private/tmp/go-cache

KIND_IMAGE ?= kindest/node:v1.33.1
KIND_CLUSTER ?= skill-registry
K8S_NAMESPACE ?= skill-registry

CATALOG_IMAGE ?= skill-registry-catalog:dev
WEB_IMAGE ?= skill-registry-web:dev
CATALOG_DOCKERFILE ?= zarf/docker/local/Dockerfile
WEB_DOCKERFILE ?= zarf/docker/local/skills-web.Dockerfile

DEV_KIND_CONFIG ?= zarf/k8s/dev/kind-config.yaml
DEV_K8S_MANIFEST ?= zarf/k8s/dev/kind-stack.yaml
DEV_POSTGRES_IMAGE ?= postgres:17
DEV_NEXUS_IMAGE ?= sonatype/nexus3:latest

NEXUS_URL ?= http://127.0.0.1:8081
NEXUS_STATE_DIR ?= /private/tmp/skill-registry-nexus
NEXUS_REPOSITORY_NAME ?= skills-npm
NEXUS_REPOSITORY_URL ?= $(NEXUS_URL)/repository/$(NEXUS_REPOSITORY_NAME)
NEXUS_NPMRC ?= $(NEXUS_STATE_DIR)/.npmrc
NEXUS_INITIAL_PASSWORD_FILE ?= $(NEXUS_STATE_DIR)/admin.password
NEXUS_ADMIN_PASSWORD ?= admin123456
NEXUS_ENABLE_ANONYMOUS_READ ?= true
STRICT ?=

.PHONY: help dev deploy build verify test-go test-web build-web clean-dev
.PHONY: _build-images _build-catalog-image _build-web-image
.PHONY: _dev-prepull _dev-up _dev-load _dev-apply _dev-bootstrap _dev-ready

help:
	@printf "skill_registry make targets\n\n"
	@printf "  make dev       Build images, create/update local kind env, apply manifests, bootstrap Nexus, wait ready\n"
	@printf "  make deploy    Deploy the latest local images into the existing kind env and wait ready\n"
	@printf "  make build     Build local catalog/web images\n"
	@printf "  make verify    Run top-level verification targets\n"
	@printf "  make test-go   Run go test ./...\n"
	@printf "  make test-web  Run web tests in %s\n" "$(WEB_DIR)"
	@printf "  make build-web Run web build in %s\n" "$(WEB_DIR)"
	@printf "  make clean-dev Remove local Nexus bootstrap state under %s\n" "$(NEXUS_STATE_DIR)"

dev: build _dev-prepull _dev-up deploy

deploy: _dev-load _dev-apply _dev-bootstrap _dev-ready

build: _build-images

verify: test-go test-web

test-go:
	env GOCACHE=$(GO_CACHE) go test ./...

test-web:
	npm --prefix $(WEB_DIR) test

build-web:
	npm --prefix $(WEB_DIR) run build

clean-dev:
	rm -rf $(NEXUS_STATE_DIR)

_build-images: _build-catalog-image _build-web-image

_build-catalog-image:
	docker build -f $(CATALOG_DOCKERFILE) -t $(CATALOG_IMAGE) .

_build-web-image:
	docker build -f $(WEB_DOCKERFILE) -t $(WEB_IMAGE) .

_dev-prepull:
	docker pull $(DEV_POSTGRES_IMAGE)
	docker pull $(DEV_NEXUS_IMAGE)

_dev-up:
	@if kind get clusters | grep -qx '$(KIND_CLUSTER)'; then \
		echo "kind cluster $(KIND_CLUSTER) already exists"; \
	else \
		kind create cluster --image $(KIND_IMAGE) --name $(KIND_CLUSTER) --config $(DEV_KIND_CONFIG); \
	fi

_dev-load:
	docker image inspect $(DEV_POSTGRES_IMAGE) >/dev/null
	docker image inspect $(DEV_NEXUS_IMAGE) >/dev/null
	kind load docker-image $(CATALOG_IMAGE) --name $(KIND_CLUSTER)
	kind load docker-image $(WEB_IMAGE) --name $(KIND_CLUSTER)
	@platform="$$(docker image inspect $(DEV_POSTGRES_IMAGE) --format '{{.Os}}/{{.Architecture}}{{if .Variant}}/{{.Variant}}{{end}}')"; \
	archive="$$(mktemp -t skill-registry-postgres-archive)"; \
	docker save --platform "$$platform" -o "$$archive" $(DEV_POSTGRES_IMAGE); \
	kind load image-archive "$$archive" --name $(KIND_CLUSTER); \
	rm -f "$$archive"
	@platform="$$(docker image inspect $(DEV_NEXUS_IMAGE) --format '{{.Os}}/{{.Architecture}}{{if .Variant}}/{{.Variant}}{{end}}')"; \
	archive="$$(mktemp -t skill-registry-nexus-archive)"; \
	docker save --platform "$$platform" -o "$$archive" $(DEV_NEXUS_IMAGE); \
	kind load image-archive "$$archive" --name $(KIND_CLUSTER); \
	rm -f "$$archive"

_dev-apply:
	kubectl apply -f $(DEV_K8S_MANIFEST)

_dev-bootstrap:
	@mkdir -p $(NEXUS_STATE_DIR)
	@kubectl -n $(K8S_NAMESPACE) rollout status deployment/nexus
	@pod_name="$$(kubectl -n $(K8S_NAMESPACE) get pods -l app=nexus -o jsonpath='{.items[0].metadata.name}')"; \
	test -n "$$pod_name" || (echo "Nexus pod not found in namespace $(K8S_NAMESPACE)" && exit 1); \
	kubectl -n $(K8S_NAMESPACE) exec "$$pod_name" -- sh -c 'until [ -f /nexus-data/admin.password ]; do sleep 2; done; cat /nexus-data/admin.password' > $(NEXUS_INITIAL_PASSWORD_FILE); \
	anonymous_read="$(NEXUS_ENABLE_ANONYMOUS_READ)"; \
	if [ -n "$(STRICT)" ] && [ "$(STRICT)" != "0" ]; then \
		anonymous_read="false"; \
	fi; \
	env \
		NEXUS_URL=$(NEXUS_URL) \
		NEXUS_STATE_DIR=$(NEXUS_STATE_DIR) \
		NEXUS_INITIAL_PASSWORD_FILE=$(NEXUS_INITIAL_PASSWORD_FILE) \
		NEXUS_REPOSITORY_NAME=$(NEXUS_REPOSITORY_NAME) \
		NEXUS_REPOSITORY_URL=$(NEXUS_REPOSITORY_URL) \
		NEXUS_ADMIN_PASSWORD=$(NEXUS_ADMIN_PASSWORD) \
		NEXUS_ENABLE_ANONYMOUS_READ="$$anonymous_read" \
		NEXUS_NPMRC_PATH=$(NEXUS_NPMRC) \
		./zarf/scripts/nexus-bootstrap.sh; \
	auth_header="$$(node -e 'const fs = require("fs"); const path = process.argv[1]; const lines = fs.readFileSync(path, "utf8").split(/\\r?\\n/); for (const raw of lines) { const line = raw.trim(); if (!line || line.startsWith("#") || line.startsWith(";")) continue; const tokenMatch = line.match(/:_authToken=(.+)$$/); if (tokenMatch) { process.stdout.write("Bearer " + tokenMatch[1].trim()); process.exit(0); } const basicMatch = line.match(/:_auth=(.+)$$/); if (basicMatch) { process.stdout.write("Basic " + basicMatch[1].trim()); process.exit(0); } } process.exit(1);' "$(NEXUS_NPMRC)")"; \
	test -n "$$auth_header" || (echo "Failed to derive npm publish authorization from $(NEXUS_NPMRC)" && exit 1); \
	encoded_auth="$$(printf '%s' "$$auth_header" | base64 | tr -d '\n')"; \
	kubectl -n $(K8S_NAMESPACE) patch secret catalog-secrets --type merge -p "{\"data\":{\"skill_registry_npm_publish_authorization\":\"$$encoded_auth\"}}"; \
	kubectl -n $(K8S_NAMESPACE) rollout restart deployment/catalog; \
	kubectl -n $(K8S_NAMESPACE) rollout status deployment/catalog

_dev-ready:
	kubectl -n $(K8S_NAMESPACE) rollout status deployment/postgres
	kubectl -n $(K8S_NAMESPACE) rollout status deployment/nexus
	kubectl -n $(K8S_NAMESPACE) rollout status deployment/catalog
	kubectl -n $(K8S_NAMESPACE) rollout status deployment/skills-web
