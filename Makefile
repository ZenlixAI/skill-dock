SHELL := /bin/zsh

SITE_DIR := apps/site
CLI_DIR := apps/skills-cli
CLI_PNPM := npx -y pnpm@10.17.1
GO_CACHE ?= /private/tmp/go-cache

KIND_IMAGE ?= kindest/node:v1.33.1
KIND_CLUSTER ?= skill-registry
DEV_KIND_CONFIG ?= zarf/k8s/dev/kind-config.yaml
DEV_MANIFEST ?= zarf/k8s/dev/kind-stack.yaml

CATALOG_IMAGE ?= skill-registry-catalog:dev
SITE_IMAGE ?= skill-registry-site:dev
CATALOG_DOCKERFILE ?= zarf/docker/local/Dockerfile
SITE_DOCKERFILE ?= zarf/docker/local/site.Dockerfile
POSTGRES_IMAGE ?= postgres:17
POSTGRES_IMAGE_ARCHIVE ?= /private/tmp/postgres-17.tar
POSTGRES_PLATFORM ?= linux/arm64/v8

ARTIFACTKEEPER_IMAGE ?= verdaccio/verdaccio:6
ARTIFACTKEEPER_CONTAINER_NAME ?= skill-registry-artifact-keeper
ARTIFACTKEEPER_STATE_DIR ?= /private/tmp/skill-registry-artifactkeeper
ARTIFACTKEEPER_CONFIG_DIR ?= $(ARTIFACTKEEPER_STATE_DIR)/conf
ARTIFACTKEEPER_STORAGE_DIR ?= $(ARTIFACTKEEPER_STATE_DIR)/storage
ARTIFACTKEEPER_TOKEN_FILE ?= $(ARTIFACTKEEPER_STATE_DIR)/publish.auth
ARTIFACTKEEPER_INTERNAL_NPM_URL ?= http://host.docker.internal:30080/
ARTIFACTKEEPER_PUBLIC_NPM_URL ?= http://127.0.0.1:18080/repository/skills-npm/
ARTIFACTKEEPER_NPM_READ_AUTHORIZATION ?=
ARTIFACTKEEPER_NPM_PUBLISH_AUTHORIZATION ?=
ARTIFACTKEEPER_PUBLISH_USERNAME ?= publisher
ARTIFACTKEEPER_PUBLISH_PASSWORD ?= publisher-secret
ARTIFACTKEEPER_PUBLISH_EMAIL ?= publisher@example.local
CLI_PUBLISH_NPMRC ?= $(ARTIFACTKEEPER_STATE_DIR)/publish-cli.npmrc
CLI_PUBLISH_NPM_CACHE ?= $(ARTIFACTKEEPER_STATE_DIR)/npm-cache
CLI_PUBLISH_VERSION_STATE ?= $(ARTIFACTKEEPER_STATE_DIR)/cli-snapshot.version
CLI_PUBLISH_DIST_TAG ?= latest
CLI_PUBLISH_REGISTRY ?= http://127.0.0.1:18080/repository/skills-npm/

.PHONY: test build dev dev-update down publish-cli-snapshot

test:
	env GOCACHE=$(GO_CACHE) go test ./...
	npm --prefix $(SITE_DIR) test
	$(CLI_PNPM) --dir $(CLI_DIR) test
	$(CLI_PNPM) --dir $(CLI_DIR) type-check

build:
	docker build --pull=false -f $(CATALOG_DOCKERFILE) -t $(CATALOG_IMAGE) .
	docker build --pull=false -f $(SITE_DOCKERFILE) -t $(SITE_IMAGE) .

dev: build
	@if kind get clusters | grep -qx '$(KIND_CLUSTER)'; then \
		echo "kind cluster $(KIND_CLUSTER) already exists"; \
	else \
		kind create cluster --image $(KIND_IMAGE) --name $(KIND_CLUSTER) --config $(DEV_KIND_CONFIG); \
	fi
	docker save --platform $(POSTGRES_PLATFORM) -o $(POSTGRES_IMAGE_ARCHIVE) $(POSTGRES_IMAGE)
	kind load image-archive $(POSTGRES_IMAGE_ARCHIVE) --name $(KIND_CLUSTER)
	kind load docker-image $(CATALOG_IMAGE) --name $(KIND_CLUSTER)
	kind load docker-image $(SITE_IMAGE) --name $(KIND_CLUSTER)
	kubectl apply -f $(DEV_MANIFEST)
	./zarf/scripts/start-artifact-keeper.sh
	artifactkeeper_auth="$$(cat $(ARTIFACTKEEPER_TOKEN_FILE))"; \
	kubectl -n skill-registry patch secret catalog-secrets --type merge -p "{\"stringData\":{\"skill_registry_artifact_read_url\":\"$(ARTIFACTKEEPER_INTERNAL_NPM_URL)\",\"skill_registry_artifact_read_authorization\":\"$$artifactkeeper_auth\",\"skill_registry_artifact_publish_authorization\":\"$$artifactkeeper_auth\",\"skill_registry_npm_read_authorization\":\"$$artifactkeeper_auth\",\"skill_registry_npm_publish_authorization\":\"$$artifactkeeper_auth\",\"skill_registry_public_install_url\":\"$(ARTIFACTKEEPER_PUBLIC_NPM_URL)\"}}"
	kubectl -n skill-registry rollout restart deployment/postgres
	kubectl -n skill-registry rollout status deployment/postgres
	kubectl -n skill-registry rollout restart deployment/catalog
	kubectl -n skill-registry rollout status deployment/catalog
	kubectl -n skill-registry rollout status deployment/site
	SKILLS_REGISTRY_URL=http://127.0.0.1:18080 \
	SKILLS_REGISTRY_TOKEN=publisher-secret \
	./zarf/scripts/initialize-test-skills.sh


dev-update: build
	@if ! kind get clusters | grep -qx '$(KIND_CLUSTER)'; then \
		echo "kind cluster $(KIND_CLUSTER) does not exist; run make dev first"; \
		exit 1; \
	fi
	kubectl apply -f $(DEV_MANIFEST)
	docker save --platform $(POSTGRES_PLATFORM) -o $(POSTGRES_IMAGE_ARCHIVE) $(POSTGRES_IMAGE)
	kind load image-archive $(POSTGRES_IMAGE_ARCHIVE) --name $(KIND_CLUSTER)
	kind load docker-image $(CATALOG_IMAGE) --name $(KIND_CLUSTER)
	kind load docker-image $(SITE_IMAGE) --name $(KIND_CLUSTER)
	kubectl -n skill-registry rollout restart deployment/postgres
	kubectl -n skill-registry rollout status deployment/postgres
	kubectl -n skill-registry rollout restart deployment/catalog
	kubectl -n skill-registry rollout status deployment/catalog
	kubectl -n skill-registry rollout restart deployment/site
	kubectl -n skill-registry rollout status deployment/site

publish-cli:
	CLI_PUBLISH_CLI_DIR="$(CLI_DIR)" \
	CLI_PUBLISH_DIST_TAG="$(CLI_PUBLISH_DIST_TAG)" \
	CLI_PUBLISH_NPM_CACHE="$(CLI_PUBLISH_NPM_CACHE)" \
	CLI_PUBLISH_NPMRC="$(CLI_PUBLISH_NPMRC)" \
	CLI_PUBLISH_REGISTRY="$(CLI_PUBLISH_REGISTRY)" \
	CLI_PUBLISH_VERSION_STATE="$(CLI_PUBLISH_VERSION_STATE)" \
	./zarf/scripts/publish-cli-snapshot.sh

down:
	kind delete cluster --name $(KIND_CLUSTER)
