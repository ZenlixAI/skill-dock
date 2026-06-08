SHELL := /bin/zsh

CLI_DIR := apps/skills-cli
CLI_ENTRY := $(CLI_DIR)/src/cli.ts
WEB_DIR := apps/skills-web
WEB_E2E_PREVIEW_PORT ?= 4174
SESSION_E2E_CATALOG_PORT ?= 18081

GO_CACHE ?= /private/tmp/go-cache
UV_CACHE_DIR ?= /private/tmp/uv-cache
CLI_HOME ?= /private/tmp/skills-cli-home
HOST_HOME ?= $(HOME)
HOST_XDG_STATE_HOME ?= $(XDG_STATE_HOME)
CLI_CACHE ?= /private/tmp/skills-cli-cache
CLI_STATE ?= $(CLI_HOME)/state

REGISTRY_ADDR ?= 127.0.0.1:18080
REGISTRY_URL ?= http://$(REGISTRY_ADDR)
NEXUS_ADDR ?= 127.0.0.1:8081
NEXUS_URL ?= http://$(NEXUS_ADDR)
NEXUS_STATE_DIR ?= /private/tmp/skill-registry-nexus
NEXUS_REPOSITORY_NAME ?= skills-npm
NEXUS_REPOSITORY_URL ?= $(NEXUS_URL)/repository/$(NEXUS_REPOSITORY_NAME)
NEXUS_NPM_REGISTRY ?= $(NEXUS_REPOSITORY_URL)/
NEXUS_ADMIN_PASSWORD ?= admin123456
NEXUS_NPMRC ?= $(NEXUS_STATE_DIR)/.npmrc
NEXUS_INITIAL_PASSWORD_FILE ?= $(NEXUS_STATE_DIR)/admin.password
NEXUS_ENABLE_ANONYMOUS_READ ?= true

READER_TOKEN ?= reader-secret
AUTHORING_TOKEN ?= authoring-secret
PUBLISHER_TOKEN ?= publisher-secret

PACKAGE ?= @company.skill/pdf-skill
VERSION ?=
QUERY ?= pdf
SKILL_PATH ?=
MANIFEST ?=
OUTPUT ?= text
NPM_BACKEND ?= default
NPM_USERCONFIG ?=
STRICT ?=
CLI_ARGS ?=
NAME ?=

NPM_BACKENDS := default|nexus
OUTPUT_FORMATS := text|json
CLI_PUBLISH_EXAMPLE ?= /abs/path/to/skill
CLI_IMPORT_EXAMPLE_PACKAGE ?= @company.skill/your-skill
CLI_IMPORT_EXAMPLE_VERSION ?= 0.1.0
BATCH_IMPORT_MANIFEST_EXAMPLE ?= /abs/path/to/import.json
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

VERIFY_FAST_TARGETS := test-go typecheck-cli test-cli
VERIFY_FULL_TARGETS := $(VERIFY_FAST_TARGETS) test-cli-dist
STRICT_FLAG = $(if $(STRICT),--strict)
WEB_E2E_ENV = env CI=1 VITE_SKILL_REGISTRY_URL=$(REGISTRY_URL) VITE_SKILL_REGISTRY_TOKEN=$(PUBLISHER_TOKEN)
SESSION_E2E_CATALOG_COMMAND := $(abspath zarf/tools/start-session-e2e-catalog.sh)
BATCH_IMPORT_ENV = env SKILLS_REGISTRY_URL=$(REGISTRY_URL) SKILLS_REGISTRY_TOKEN=$(PUBLISHER_TOKEN) UV_CACHE_DIR=$(UV_CACHE_DIR)
CLI_TOKEN ?=
MAKE_CLI = $(MAKE) --no-print-directory cli

define resolve_npm_backend
npm_registry=""; \
userconfig="$(NPM_USERCONFIG)"; \
case "$(NPM_BACKEND)" in \
	""|default) ;; \
	nexus) npm_registry="$(NEXUS_NPM_REGISTRY)"; [ -n "$$userconfig" ] || userconfig="$(NEXUS_NPMRC)" ;; \
	*) echo "Unsupported NPM_BACKEND='$(NPM_BACKEND)'. Use default or nexus."; exit 1 ;; \
esac
endef

.PHONY: help verify verify-full test-go test-go-packaging python-sync test-cli test-cli-dist test-cli-packaging typecheck-cli build-cli web-install typecheck-web test-web build-web test-web-e2e test-web-e2e-login web-dev build-catalog-image build-web-image build-images dev-prepull dev-up dev-down dev-load dev-apply dev-delete dev-bootstrap dev-ready dev-run dev-reset dev-update dev-update-apply dev-status cli cli-find cli-add cli-publish cli-import-local cli-init cli-inspect cli-list cli-remove cli-update batch-import clean-dev

help:
	@printf "skill_registry make targets\n\n"
	@printf "Verification:\n"
	@printf "  make verify            Fast local-dev gate: %s\n" "$(VERIFY_FAST_TARGETS)"
	@printf "  make verify-full       Extended gate: %s\n" "$(VERIFY_FULL_TARGETS)"
	@printf "  make test-go           Run go test ./...\n"
	@printf "  make test-go-packaging Run focused Go tests for packaging + packaging HTTP handlers\n"
	@printf "  make python-sync       Create/update the uv-managed Python environment\n"
	@printf "  make typecheck-cli     Run npm type-check in %s\n" "$(CLI_DIR)"
	@printf "  make test-cli          Run the fast CLI suite in %s\n" "$(CLI_DIR)"
	@printf "  make test-cli-dist     Run the CLI dist build smoke test in %s\n" "$(CLI_DIR)"
	@printf "  make test-cli-packaging Run focused CLI packaging / publish tests in %s\n" "$(CLI_DIR)"
	@printf "  make build-cli         Run npm build in %s\n\n" "$(CLI_DIR)"
	@printf "Web UI:\n"
	@printf "  make web-install       Install npm dependencies in %s\n" "$(WEB_DIR)"
	@printf "  make typecheck-web     Run npm type-check in %s\n" "$(WEB_DIR)"
	@printf "  make test-web          Run npm test in %s\n" "$(WEB_DIR)"
	@printf "  make build-web         Run npm build in %s\n" "$(WEB_DIR)"
	@printf "  make test-web-e2e      Build %s for real-mode smoke tests, then run Playwright against local Catalog on %s\n" "$(WEB_DIR)" "$(REGISTRY_URL)"
	@printf "  make test-web-e2e-login Build %s for session-login E2E, auto-clean stale E2E listeners, then run Playwright against a local session-enabled Catalog on 127.0.0.1:%s\n" "$(WEB_DIR)" "$(SESSION_E2E_CATALOG_PORT)"
	@printf "  make web-dev           Start Vite dev server for %s\n\n" "$(WEB_DIR)"
	@printf "Local KIND Development:\n"
	@printf "  make build-images      Build local dev images: %s and %s\n" "$(CATALOG_IMAGE)" "$(WEB_IMAGE)"
	@printf "  make dev-prepull       Pull external runtime images needed by the kind stack\n"
	@printf "  make dev-up            Create kind cluster %s using %s\n" "$(KIND_CLUSTER)" "$(DEV_KIND_CONFIG)"
	@printf "  make dev-load          Load %s and %s into kind\n" "$(CATALOG_IMAGE)" "$(WEB_IMAGE)"
	@printf "  make dev-apply         Apply %s into namespace %s\n" "$(DEV_K8S_MANIFEST)" "$(K8S_NAMESPACE)"
	@printf "  make dev-bootstrap     Bootstrap in-kind Nexus and write %s\n" "$(NEXUS_NPMRC)"
	@printf "  make dev-ready         Wait until postgres, nexus, catalog, and skills-web are Ready\n"
	@printf "  make dev-run           Build images, create kind, load images, apply manifests, bootstrap Nexus\n"
	@printf "  make dev-reset         Delete kind cluster, wipe local dev state, then run dev-run\n"
	@printf "  make dev-update        Rebuild images, reload them, and rollout-restart deployments\n"
	@printf "  make dev-update-apply  Rebuild images, reload them, and re-apply manifests\n"
	@printf "  make dev-status        Show pods and services in namespace %s\n" "$(K8S_NAMESPACE)"
	@printf "  make dev-down          Delete kind cluster %s\n\n" "$(KIND_CLUSTER)"
	@printf "Docker Images:\n"
	@printf "  make build-catalog-image  Build %s using %s\n" "$(CATALOG_IMAGE)" "$(CATALOG_DOCKERFILE)"
	@printf "  make build-web-image      Build %s using %s\n\n" "$(WEB_IMAGE)" "$(WEB_DOCKERFILE)"
	@printf "CLI against local kind stack:\n"
	@printf "  make cli ARGS=\"find pdf\" [CLI_TOKEN=auto] [NPM_BACKEND=%s] [NPM_USERCONFIG=/abs/path/.npmrc]\n" "$(NPM_BACKENDS)"
	@printf "  make cli-find QUERY=%s [OUTPUT=%s]  # compatibility alias over make cli\n" "$(QUERY)" "$(OUTPUT_FORMATS)"
	@printf "  make cli-add PACKAGE=%s [NPM_BACKEND=%s] [NPM_USERCONFIG=/abs/path/.npmrc]  # compatibility alias\n" "$(PACKAGE)" "$(NPM_BACKENDS)"
	@printf "  make cli-publish SKILL_PATH=%s [NPM_BACKEND=%s] [NPM_USERCONFIG=/abs/path/.npmrc]  # compatibility alias\n" "$(CLI_PUBLISH_EXAMPLE)" "$(NPM_BACKENDS)"
	@printf "  make cli-import-local SKILL_PATH=%s PACKAGE=%s VERSION=%s [NPM_BACKEND=%s] [NPM_USERCONFIG=/abs/path/.npmrc]  # compatibility alias\n" "$(CLI_PUBLISH_EXAMPLE)" "$(CLI_IMPORT_EXAMPLE_PACKAGE)" "$(CLI_IMPORT_EXAMPLE_VERSION)" "$(NPM_BACKENDS)"
	@printf "  make cli-list [CLI_ARGS=\"--json\"]\n"
	@printf "  make cli-remove CLI_ARGS=\"my-skill -y\"\n"
	@printf "  make cli-update [CLI_ARGS=\"my-skill -g\"]\n"
	@printf "  make cli-init NAME=my-skill\n"
	@printf "  make cli-inspect SKILL_PATH=%s\n\n" "$(CLI_PUBLISH_EXAMPLE)"
	@printf "Batch import:\n"
	@printf "  make batch-import MANIFEST=%s [NPM_BACKEND=%s] [NPM_USERCONFIG=/abs/path/.npmrc] [STRICT=1]\n\n" "$(BATCH_IMPORT_MANIFEST_EXAMPLE)" "$(NPM_BACKENDS)"
	@printf "Cleanup:\n"
	@printf "  make clean-dev         Remove temp CLI, Nexus bootstrap artifacts, uv cache, and .venv state\n"

verify: $(VERIFY_FAST_TARGETS)

verify-full: $(VERIFY_FULL_TARGETS)

test-go:
	env GOCACHE=$(GO_CACHE) go test ./...

test-go-packaging:
	env GOCACHE=$(GO_CACHE) go test ./business/packaging ./app/catalogapi

python-sync:
	env UV_CACHE_DIR=$(UV_CACHE_DIR) uv sync

typecheck-cli:
	env HOME=$(CLI_HOME) XDG_CACHE_HOME=$(CLI_CACHE) npm --prefix $(CLI_DIR) run type-check

test-cli:
	env HOME=$(CLI_HOME) XDG_CACHE_HOME=$(CLI_CACHE) npm --prefix $(CLI_DIR) test

test-cli-dist:
	env HOME=$(CLI_HOME) XDG_CACHE_HOME=$(CLI_CACHE) npm --prefix $(CLI_DIR) run test:dist

test-cli-packaging:
	env HOME=$(CLI_HOME) XDG_CACHE_HOME=$(CLI_CACHE) npm --prefix $(CLI_DIR) run test:packaging

build-cli:
	env HOME=$(CLI_HOME) XDG_CACHE_HOME=$(CLI_CACHE) npm --prefix $(CLI_DIR) run build

web-install:
	npm --prefix $(WEB_DIR) install

typecheck-web:
	npm --prefix $(WEB_DIR) run type-check

test-web:
	npm --prefix $(WEB_DIR) test

build-web:
	npm --prefix $(WEB_DIR) run build

build-catalog-image:
	docker build -f $(CATALOG_DOCKERFILE) -t $(CATALOG_IMAGE) .

build-web-image:
	docker build -f $(WEB_DOCKERFILE) -t $(WEB_IMAGE) .

build-images: build-catalog-image build-web-image

test-web-e2e:
	@pids="$$(lsof -tiTCP:$(WEB_E2E_PREVIEW_PORT) -sTCP:LISTEN 2>/dev/null || true)"; \
	if [ -n "$$pids" ]; then \
		echo "Stopping stale web E2E preview listener(s) on $(WEB_E2E_PREVIEW_PORT): $$pids"; \
		kill $$pids; \
		sleep 1; \
	fi
	$(WEB_E2E_ENV) npm --prefix $(WEB_DIR) run build
	$(WEB_E2E_ENV) npm --prefix $(WEB_DIR) run test:e2e

test-web-e2e-login:
	@for port in $(SESSION_E2E_CATALOG_PORT) $(WEB_E2E_PREVIEW_PORT); do \
		pids="$$(lsof -tiTCP:$$port -sTCP:LISTEN 2>/dev/null || true)"; \
		if [ -n "$$pids" ]; then \
			echo "Stopping stale session-login E2E listener(s) on $$port: $$pids"; \
			kill $$pids; \
			sleep 1; \
		fi; \
	done
	env \
		CI=1 \
		VITE_SKILL_REGISTRY_URL=http://127.0.0.1:$(SESSION_E2E_CATALOG_PORT) \
		VITE_SKILL_REGISTRY_TOKEN= \
		VITE_SKILL_REGISTRY_ROLE= \
		VITE_SKILL_REGISTRY_ACTOR= \
		PLAYWRIGHT_CATALOG_COMMAND=$(SESSION_E2E_CATALOG_COMMAND) \
		PLAYWRIGHT_CATALOG_PORT=$(SESSION_E2E_CATALOG_PORT) \
		npm --prefix $(WEB_DIR) run build
	env \
		CI=1 \
		VITE_SKILL_REGISTRY_URL=http://127.0.0.1:$(SESSION_E2E_CATALOG_PORT) \
		VITE_SKILL_REGISTRY_TOKEN= \
		VITE_SKILL_REGISTRY_ROLE= \
		VITE_SKILL_REGISTRY_ACTOR= \
		PLAYWRIGHT_CATALOG_COMMAND=$(SESSION_E2E_CATALOG_COMMAND) \
		PLAYWRIGHT_CATALOG_PORT=$(SESSION_E2E_CATALOG_PORT) \
		npm --prefix $(WEB_DIR) run test:e2e -- tests/e2e/session-login.spec.ts

web-dev:
	npm --prefix $(WEB_DIR) run dev

dev-prepull:
	docker pull $(DEV_POSTGRES_IMAGE)
	docker pull $(DEV_NEXUS_IMAGE)

dev-up:
	kind create cluster \
		--image $(KIND_IMAGE) \
		--name $(KIND_CLUSTER) \
		--config $(DEV_KIND_CONFIG)

dev-down:
	kind delete cluster --name $(KIND_CLUSTER)

dev-load:
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

dev-apply:
	kubectl apply -f $(DEV_K8S_MANIFEST)

dev-delete:
	kubectl delete -f $(DEV_K8S_MANIFEST) --ignore-not-found

dev-bootstrap:
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
		./zarf/scripts/nexus-bootstrap.sh

dev-ready:
	kubectl -n $(K8S_NAMESPACE) rollout status deployment/postgres
	kubectl -n $(K8S_NAMESPACE) rollout status deployment/nexus
	kubectl -n $(K8S_NAMESPACE) rollout status deployment/catalog
	kubectl -n $(K8S_NAMESPACE) rollout status deployment/skills-web

dev-run: build-images dev-prepull dev-up dev-load dev-apply dev-bootstrap dev-ready

dev-reset:
	$(MAKE) --no-print-directory dev-down
	$(MAKE) --no-print-directory clean-dev
	$(MAKE) --no-print-directory dev-run

dev-update: build-images dev-load
	kubectl -n $(K8S_NAMESPACE) rollout restart deployment/nexus deployment/catalog deployment/skills-web

dev-update-apply: build-images dev-load dev-apply

dev-status:
	kubectl -n $(K8S_NAMESPACE) get pods,svc -o wide

cli:
	@test -n '$(strip $(ARGS))' || (echo "ARGS is required, e.g. make cli ARGS=\"find pdf\"" && exit 1)
	@eval "set -- $(ARGS)"; \
	command="$$1"; \
	cli_token="$(CLI_TOKEN)"; \
	runtime_home="$(CLI_HOME)"; \
	runtime_state="$(CLI_STATE)"; \
	case "$$command" in \
		add|a|list|ls|remove|rm|update) runtime_home="$(HOST_HOME)"; runtime_state="$(HOST_XDG_STATE_HOME)" ;; \
	esac; \
	if [ -z "$$cli_token" ]; then \
		case "$$command" in \
			find|add|a|list|ls|remove|rm|update) cli_token="$(READER_TOKEN)" ;; \
			init|inspect) cli_token="$(AUTHORING_TOKEN)" ;; \
			publish|import-local) cli_token="$(PUBLISHER_TOKEN)" ;; \
			*) cli_token="$(PUBLISHER_TOKEN)" ;; \
		esac; \
	fi; \
	needs_registry=0; \
	case "$$command" in \
		find|add|a|inspect|publish|import-local) needs_registry=1 ;; \
	esac; \
	if [ "$$needs_registry" = "1" ]; then \
		has_registry=0; \
		for arg in "$$@"; do \
			if [ "$$arg" = "--registry" ]; then has_registry=1; fi; \
		done; \
		if [ "$$has_registry" = "0" ]; then \
			set -- "$$@" --registry "$(REGISTRY_URL)"; \
		fi; \
	fi; \
	if [ "$$command" = "import-local" ] && [ "$(NPM_BACKEND)" = "nexus" ]; then \
		has_npm_registry=0; \
		for arg in "$$@"; do \
			if [ "$$arg" = "--npm-registry" ]; then has_npm_registry=1; fi; \
		done; \
		if [ "$$has_npm_registry" = "0" ]; then \
			set -- "$$@" --npm-registry "$(NEXUS_NPM_REGISTRY)"; \
		fi; \
	fi; \
	$(resolve_npm_backend); \
	state_env=""; \
	if [ -n "$$runtime_state" ]; then state_env="XDG_STATE_HOME=$$runtime_state"; fi; \
	env \
		SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
		SKILLS_REGISTRY_TOKEN="$$cli_token" \
		HOME="$$runtime_home" \
		XDG_CACHE_HOME=$(CLI_CACHE) \
		$$state_env \
		NPM_CONFIG_USERCONFIG="$$userconfig" \
		npm_config_registry="$$npm_registry" \
		node $(CLI_ENTRY) "$$@"

cli-find:
	@case "$(OUTPUT)" in \
		""|text) output_flag="" ;; \
		json) output_flag="--json" ;; \
		*) echo "Unsupported OUTPUT='$(OUTPUT)'. Use text or json."; exit 1 ;; \
	esac; \
	$(MAKE_CLI) CLI_TOKEN=$(READER_TOKEN) ARGS="find \"$(QUERY)\" $$output_flag $(CLI_ARGS)"

cli-add:
	@$(MAKE_CLI) \
		CLI_TOKEN=$(READER_TOKEN) \
		NPM_BACKEND=$(NPM_BACKEND) \
		NPM_USERCONFIG="$(NPM_USERCONFIG)" \
		ARGS="add $(PACKAGE) -y $(CLI_ARGS)"

cli-publish:
	@test -n "$(SKILL_PATH)" || (echo "SKILL_PATH is required, e.g. make cli-publish SKILL_PATH=/abs/path/to/skill" && exit 1)
	@$(MAKE_CLI) \
		NPM_BACKEND=$(NPM_BACKEND) \
		NPM_USERCONFIG="$(NPM_USERCONFIG)" \
		ARGS="publish $(SKILL_PATH) $(CLI_ARGS)"

cli-import-local:
	@test -n "$(SKILL_PATH)" || (echo "SKILL_PATH is required, e.g. make cli-import-local SKILL_PATH=/abs/path/to/skill PACKAGE=@company.skill/your-skill VERSION=0.1.0" && exit 1)
	@test -n "$(PACKAGE)" || (echo "PACKAGE is required" && exit 1)
	@test -n "$(VERSION)" || (echo "VERSION is required" && exit 1)
	@args="import-local $(SKILL_PATH) --package-name $(PACKAGE) --version $(VERSION)"; \
	if [ -n "$(STRICT)" ]; then args="$$args --strict"; fi; \
	$(MAKE_CLI) \
		NPM_BACKEND=$(NPM_BACKEND) \
		NPM_USERCONFIG="$(NPM_USERCONFIG)" \
		ARGS="$$args $(CLI_ARGS)"

cli-init:
	@args="init"; \
	if [ -n "$(NAME)" ]; then args="$$args $(NAME)"; fi; \
	$(MAKE_CLI) CLI_TOKEN=$(AUTHORING_TOKEN) ARGS="$$args $(CLI_ARGS)"

cli-inspect:
	@test -n "$(SKILL_PATH)" || (echo "SKILL_PATH is required, e.g. make cli-inspect SKILL_PATH=/abs/path/to/skill" && exit 1)
	@$(MAKE_CLI) CLI_TOKEN=$(AUTHORING_TOKEN) ARGS="inspect $(SKILL_PATH) $(CLI_ARGS)"

cli-list:
	@$(MAKE_CLI) CLI_TOKEN=$(READER_TOKEN) ARGS="list $(CLI_ARGS)"

cli-remove:
	@$(MAKE_CLI) CLI_TOKEN=$(READER_TOKEN) ARGS="remove $(CLI_ARGS)"

cli-update:
	@$(MAKE_CLI) CLI_TOKEN=$(READER_TOKEN) ARGS="update $(CLI_ARGS)"

batch-import:
	@test -n "$(MANIFEST)" || (echo "MANIFEST is required, e.g. make batch-import MANIFEST=/abs/path/to/import.json" && exit 1)
	@$(resolve_npm_backend); \
	args="uv run python zarf/tools/batch-import/batch_import_skills.py --manifest $(MANIFEST) --catalog-url $(REGISTRY_URL)"; \
	if [ -n "$$npm_registry" ]; then args="$$args --npm-registry $$npm_registry"; fi; \
	if [ -n "$(STRICT_FLAG)" ]; then args="$$args $(STRICT_FLAG)"; fi; \
	extra_env=""; \
	if [ -n "$$userconfig" ]; then extra_env="NPM_CONFIG_USERCONFIG=$$userconfig"; fi; \
	$(BATCH_IMPORT_ENV) $$extra_env $$args

clean-dev:
	rm -rf $(CLI_HOME) $(CLI_CACHE) $(UV_CACHE_DIR) .venv $(NEXUS_STATE_DIR)
