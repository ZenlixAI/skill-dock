SHELL := /bin/zsh

CLI_DIR := apps/skills-cli
CLI_ENTRY := $(CLI_DIR)/src/cli.ts
WEB_DIR := apps/skills-web

GO_CACHE ?= /private/tmp/go-cache
UV_CACHE_DIR ?= /private/tmp/uv-cache
CLI_HOME ?= /private/tmp/skills-cli-home
CLI_CACHE ?= /private/tmp/skills-cli-cache
CLI_STATE ?= $(CLI_HOME)/state

REGISTRY_ADDR ?= 127.0.0.1:18080
REGISTRY_URL ?= http://$(REGISTRY_ADDR)
REGISTRY_DB ?= file:/private/tmp/skill-registry.db
VERDACCIO_ADDR ?= 127.0.0.1:4873
VERDACCIO_URL ?= http://$(VERDACCIO_ADDR)
VERDACCIO_STATE_DIR ?= /private/tmp/skill-registry-verdaccio
VERDACCIO_AUTH_ADDR ?= 127.0.0.1:4874
VERDACCIO_AUTH_URL ?= http://$(VERDACCIO_AUTH_ADDR)
VERDACCIO_AUTH_STATE_DIR ?= /private/tmp/skill-registry-verdaccio-auth
VERDACCIO_AUTH_NPMRC ?= $(VERDACCIO_AUTH_STATE_DIR)/.npmrc
NEXUS_IMAGE ?= sonatype/nexus3:latest
NEXUS_ADDR ?= 127.0.0.1:8081
NEXUS_URL ?= http://$(NEXUS_ADDR)
NEXUS_STATE_DIR ?= /private/tmp/skill-registry-nexus
NEXUS_REPOSITORY_NAME ?= skills-npm
NEXUS_REPOSITORY_URL ?= $(NEXUS_URL)/repository/$(NEXUS_REPOSITORY_NAME)
NEXUS_ADMIN_PASSWORD ?= admin123456
NEXUS_NPMRC ?= $(NEXUS_STATE_DIR)/.npmrc
NEXUS_ENABLE_ANONYMOUS_READ ?= true

READER_TOKEN ?= reader-secret
AUTHORING_TOKEN ?= authoring-secret
PUBLISHER_TOKEN ?= publisher-secret

PACKAGE ?= @platform.skill/pdf-skill
QUERY ?= pdf
SKILL_PATH ?=
MANIFEST ?=
STRICT ?=
REGISTRY_BACKEND ?= sample
RESET_DB ?=
OUTPUT ?= text
NPM_BACKEND ?= direct
COMPOSE_STACK ?= production
VERDACCIO_MODE ?= public

.PHONY: help verify verify-full test test-go test-go-packaging python-sync test-cli test-cli-dist test-cli-packaging typecheck-cli build-cli web-install typecheck-web build-web web-dev docker-build compose-up compose-down registry-reset registry-run verdaccio-run verdaccio-reset verdaccio-login verdaccio-auth-reset nexus-run nexus-reset nexus-bootstrap cli-find cli-add cli-publish batch-import clean-dev

help:
	@printf "skill_registry make targets\n\n"
	@printf "Verification:\n"
	@printf "  make verify            Run the fast default suite: Go tests, CLI type-check, and CLI tests (excludes dist smoke)\n"
	@printf "  make verify-full       Run verify plus the CLI dist build smoke test\n"
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
	@printf "  make build-web         Run npm build in %s\n" "$(WEB_DIR)"
	@printf "  make web-dev           Start Vite dev server for %s\n\n" "$(WEB_DIR)"
	@printf "Local Catalog:\n"
	@printf "  make registry-reset    Remove the local Catalog sqlite files\n"
	@printf "  make registry-run REGISTRY_BACKEND=sample|mock|verdaccio|verdaccio-auth|nexus [RESET_DB=1]\n\n"
	@printf "Local Verdaccio:\n"
	@printf "  make verdaccio-run VERDACCIO_MODE=public|auth|strict\n"
	@printf "  make verdaccio-reset   Remove local Verdaccio state under %s\n" "$(VERDACCIO_STATE_DIR)"
	@printf "  make verdaccio-auth-reset Remove authenticated Verdaccio state under %s\n" "$(VERDACCIO_AUTH_STATE_DIR)"
	@printf "  make verdaccio-login   Run npm adduser against authenticated Verdaccio\n\n"
	@printf "Authenticated Verdaccio:\n"
	@printf "  use VERDACCIO_MODE=auth|strict together with registry-run REGISTRY_BACKEND=verdaccio-auth\n\n"
	@printf "Local Nexus:\n"
	@printf "  make nexus-run         Start local Nexus 3 on %s\n" "$(NEXUS_URL)"
	@printf "  make nexus-bootstrap [STRICT=1]   Configure Nexus and write %s\n" "$(NEXUS_NPMRC)"
	@printf "  make nexus-reset       Remove local Nexus state under %s\n" "$(NEXUS_STATE_DIR)"
	@printf "  use registry-run REGISTRY_BACKEND=nexus for Catalog against %s\n\n" "$(NEXUS_REPOSITORY_URL)"
	@printf "Containers:\n"
	@printf "  make docker-build      Build the production Catalog image\n"
	@printf "  make compose-up COMPOSE_STACK=production|local|nexus\n"
	@printf "  make compose-down COMPOSE_STACK=production|local|nexus\n\n"
	@printf "CLI against local registry:\n"
	@printf "  make cli-find QUERY=pdf [OUTPUT=json]\n"
	@printf "  make cli-add PACKAGE=@platform.skill/pdf-skill [NPM_BACKEND=direct|verdaccio|verdaccio-auth|nexus]\n"
	@printf "  make cli-publish SKILL_PATH=/abs/path/to/skill [NPM_BACKEND=direct|verdaccio|verdaccio-auth|nexus]\n\n"
	@printf "Batch import:\n"
	@printf "  make batch-import MANIFEST=/abs/path/to/import.json [NPM_BACKEND=direct|verdaccio|verdaccio-auth|nexus] [STRICT=1]\n\n"
	@printf "Cleanup:\n"
	@printf "  make clean-dev         Remove temp CLI, Catalog, Verdaccio, Nexus, uv cache, and .venv state\n"

verify: test-go typecheck-cli test-cli

verify-full: verify test-cli-dist

test: verify

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

build-web:
	npm --prefix $(WEB_DIR) run build

web-dev:
	npm --prefix $(WEB_DIR) run dev

docker-build:
	docker build -f zarf/docker/Dockerfile -t skill-registry:latest .

compose-up:
	@compose_file=""; \
	case "$(COMPOSE_STACK)" in \
		production) compose_file="zarf/compose/production.yml" ;; \
		local) compose_file="zarf/compose/local.yml" ;; \
		nexus) compose_file="zarf/compose/nexus.yml" ;; \
		*) echo "Unsupported COMPOSE_STACK: $(COMPOSE_STACK). Use production, local, or nexus." >&2; exit 1 ;; \
	esac; \
	docker compose -f "$$compose_file" up --build

compose-down:
	@compose_file=""; \
	case "$(COMPOSE_STACK)" in \
		production) compose_file="zarf/compose/production.yml" ;; \
		local) compose_file="zarf/compose/local.yml" ;; \
		nexus) compose_file="zarf/compose/nexus.yml" ;; \
		*) echo "Unsupported COMPOSE_STACK: $(COMPOSE_STACK). Use production, local, or nexus." >&2; exit 1 ;; \
	esac; \
	docker compose -f "$$compose_file" down

registry-reset:
	rm -f /private/tmp/skill-registry.db /private/tmp/skill-registry.db-shm /private/tmp/skill-registry.db-wal

registry-run:
	@if [ -n "$(RESET_DB)" ]; then \
		$(MAKE) registry-reset; \
	fi
	@sample_base_url="$(REGISTRY_URL)/mock-npm"; \
	npm_registry_url=""; \
	case "$(REGISTRY_BACKEND)" in \
		sample) ;; \
		mock) npm_registry_url="$(REGISTRY_URL)/mock-npm" ;; \
		verdaccio) sample_base_url="$(VERDACCIO_URL)"; npm_registry_url="$(VERDACCIO_URL)" ;; \
		verdaccio-auth) sample_base_url="$(VERDACCIO_AUTH_URL)"; npm_registry_url="$(VERDACCIO_AUTH_URL)" ;; \
		nexus) sample_base_url="$(NEXUS_REPOSITORY_URL)"; npm_registry_url="$(NEXUS_REPOSITORY_URL)" ;; \
		*) echo "Unsupported REGISTRY_BACKEND: $(REGISTRY_BACKEND). Use sample, mock, verdaccio, verdaccio-auth, or nexus." >&2; exit 1 ;; \
	esac; \
	if [ -n "$$npm_registry_url" ]; then \
		env \
			GOCACHE=$(GO_CACHE) \
			SKILL_REGISTRY_ADDR=$(REGISTRY_ADDR) \
			SKILL_REGISTRY_DATABASE_URL=$(REGISTRY_DB) \
			SKILL_REGISTRY_SAMPLE_NPM_BASE_URL="$$sample_base_url" \
			SKILL_REGISTRY_NPM_REGISTRY_URL="$$npm_registry_url" \
			SKILL_REGISTRY_READER_TOKEN=$(READER_TOKEN) \
			SKILL_REGISTRY_AUTHORING_TOKEN=$(AUTHORING_TOKEN) \
			SKILL_REGISTRY_PUBLISHER_TOKEN=$(PUBLISHER_TOKEN) \
			go run ./cmd/skill-registry; \
	else \
		env \
			GOCACHE=$(GO_CACHE) \
			SKILL_REGISTRY_ADDR=$(REGISTRY_ADDR) \
			SKILL_REGISTRY_DATABASE_URL=$(REGISTRY_DB) \
			SKILL_REGISTRY_SAMPLE_NPM_BASE_URL="$$sample_base_url" \
			SKILL_REGISTRY_READER_TOKEN=$(READER_TOKEN) \
			SKILL_REGISTRY_AUTHORING_TOKEN=$(AUTHORING_TOKEN) \
			SKILL_REGISTRY_PUBLISHER_TOKEN=$(PUBLISHER_TOKEN) \
			go run ./cmd/skill-registry; \
	fi

verdaccio-run:
	@verdaccio_addr="$(VERDACCIO_ADDR)"; \
	verdaccio_state_dir="$(VERDACCIO_STATE_DIR)"; \
	verdaccio_config="$(PWD)/zarf/docker/verdaccio/public.yaml"; \
	case "$(VERDACCIO_MODE)" in \
		public) ;; \
		auth) verdaccio_addr="$(VERDACCIO_AUTH_ADDR)"; verdaccio_state_dir="$(VERDACCIO_AUTH_STATE_DIR)"; verdaccio_config="$(PWD)/zarf/docker/verdaccio/auth.yaml" ;; \
		strict) verdaccio_addr="$(VERDACCIO_AUTH_ADDR)"; verdaccio_state_dir="$(VERDACCIO_AUTH_STATE_DIR)"; verdaccio_config="$(PWD)/zarf/docker/verdaccio/strict-auth.yaml" ;; \
		*) echo "Unsupported VERDACCIO_MODE: $(VERDACCIO_MODE). Use public, auth, or strict." >&2; exit 1 ;; \
	esac; \
	env \
		VERDACCIO_ADDR="$$verdaccio_addr" \
		VERDACCIO_STATE_DIR="$$verdaccio_state_dir" \
		VERDACCIO_CONFIG="$$verdaccio_config" \
		./zarf/scripts/run-verdaccio.sh

verdaccio-reset:
	rm -rf $(VERDACCIO_STATE_DIR)

verdaccio-login:
	env \
		VERDACCIO_URL=$(VERDACCIO_AUTH_URL) \
		VERDACCIO_STATE_DIR=$(VERDACCIO_AUTH_STATE_DIR) \
		VERDACCIO_NPMRC_PATH=$(VERDACCIO_AUTH_NPMRC) \
		./zarf/scripts/verdaccio-login.sh

verdaccio-auth-reset:
	rm -rf $(VERDACCIO_AUTH_STATE_DIR)

nexus-run:
	env \
		NEXUS_IMAGE=$(NEXUS_IMAGE) \
		NEXUS_ADDR=$(NEXUS_ADDR) \
		NEXUS_STATE_DIR=$(NEXUS_STATE_DIR) \
		./zarf/scripts/run-nexus.sh

nexus-bootstrap:
	env \
		NEXUS_URL=$(NEXUS_URL) \
		NEXUS_STATE_DIR=$(NEXUS_STATE_DIR) \
		NEXUS_REPOSITORY_NAME=$(NEXUS_REPOSITORY_NAME) \
		NEXUS_REPOSITORY_URL=$(NEXUS_REPOSITORY_URL) \
		NEXUS_ADMIN_PASSWORD=$(NEXUS_ADMIN_PASSWORD) \
		NEXUS_ENABLE_ANONYMOUS_READ=$(if $(STRICT),false,$(NEXUS_ENABLE_ANONYMOUS_READ)) \
		NEXUS_NPMRC_PATH=$(NEXUS_NPMRC) \
		./zarf/scripts/nexus-bootstrap.sh

nexus-reset:
	rm -rf $(NEXUS_STATE_DIR)

cli-find:
	@find_output_arg=""; \
	case "$(OUTPUT)" in \
		text|"") ;; \
		json) find_output_arg="--json" ;; \
		*) echo "Unsupported OUTPUT: $(OUTPUT). Use text or json." >&2; exit 1 ;; \
	esac; \
	env \
		SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
		SKILLS_REGISTRY_TOKEN=$(READER_TOKEN) \
		node $(CLI_ENTRY) find "$(QUERY)" --registry $(REGISTRY_URL) $$find_output_arg

cli-add:
	@npm_userconfig=""; \
	npm_registry=""; \
	case "$(NPM_BACKEND)" in \
		direct|"") ;; \
		verdaccio) npm_registry="$(VERDACCIO_URL)" ;; \
		verdaccio-auth) npm_userconfig="$(VERDACCIO_AUTH_NPMRC)"; npm_registry="$(VERDACCIO_AUTH_URL)" ;; \
		nexus) npm_userconfig="$(NEXUS_NPMRC)" ;; \
		*) echo "Unsupported NPM_BACKEND: $(NPM_BACKEND). Use direct, verdaccio, verdaccio-auth, or nexus." >&2; exit 1 ;; \
	esac; \
	if [ -n "$$npm_userconfig" ] && [ -n "$$npm_registry" ]; then \
		env \
			SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
			SKILLS_REGISTRY_TOKEN=$(READER_TOKEN) \
			HOME=$(CLI_HOME) \
			XDG_CACHE_HOME=$(CLI_CACHE) \
			XDG_STATE_HOME=$(CLI_STATE) \
			NPM_CONFIG_USERCONFIG="$$npm_userconfig" \
			npm_config_registry="$$npm_registry" \
			node $(CLI_ENTRY) add $(PACKAGE) -y --registry $(REGISTRY_URL); \
	elif [ -n "$$npm_userconfig" ]; then \
		env \
			SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
			SKILLS_REGISTRY_TOKEN=$(READER_TOKEN) \
			HOME=$(CLI_HOME) \
			XDG_CACHE_HOME=$(CLI_CACHE) \
			XDG_STATE_HOME=$(CLI_STATE) \
			NPM_CONFIG_USERCONFIG="$$npm_userconfig" \
			node $(CLI_ENTRY) add $(PACKAGE) -y --registry $(REGISTRY_URL); \
	elif [ -n "$$npm_registry" ]; then \
		env \
			SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
			SKILLS_REGISTRY_TOKEN=$(READER_TOKEN) \
			HOME=$(CLI_HOME) \
			XDG_CACHE_HOME=$(CLI_CACHE) \
			XDG_STATE_HOME=$(CLI_STATE) \
			npm_config_registry="$$npm_registry" \
			node $(CLI_ENTRY) add $(PACKAGE) -y --registry $(REGISTRY_URL); \
	else \
		env \
			SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
			SKILLS_REGISTRY_TOKEN=$(READER_TOKEN) \
			HOME=$(CLI_HOME) \
			XDG_CACHE_HOME=$(CLI_CACHE) \
			XDG_STATE_HOME=$(CLI_STATE) \
			node $(CLI_ENTRY) add $(PACKAGE) -y --registry $(REGISTRY_URL); \
	fi

cli-publish:
	@test -n "$(SKILL_PATH)" || (echo "SKILL_PATH is required, e.g. make cli-publish SKILL_PATH=/abs/path/to/skill [NPM_BACKEND=verdaccio|verdaccio-auth|nexus]" && exit 1)
	@npm_userconfig=""; \
	npm_registry=""; \
	case "$(NPM_BACKEND)" in \
		direct|"") ;; \
		verdaccio) npm_registry="$(VERDACCIO_URL)" ;; \
		verdaccio-auth) npm_userconfig="$(VERDACCIO_AUTH_NPMRC)"; npm_registry="$(VERDACCIO_AUTH_URL)" ;; \
		nexus) npm_userconfig="$(NEXUS_NPMRC)"; npm_registry="$(NEXUS_REPOSITORY_URL)/" ;; \
		*) echo "Unsupported NPM_BACKEND: $(NPM_BACKEND). Use direct, verdaccio, verdaccio-auth, or nexus." >&2; exit 1 ;; \
	esac; \
	if [ -n "$$npm_userconfig" ] && [ -n "$$npm_registry" ]; then \
		env \
			SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
			SKILLS_REGISTRY_TOKEN=$(PUBLISHER_TOKEN) \
			HOME=$(CLI_HOME) \
			XDG_CACHE_HOME=$(CLI_CACHE) \
			XDG_STATE_HOME=$(CLI_STATE) \
			NPM_CONFIG_USERCONFIG="$$npm_userconfig" \
			npm_config_registry="$$npm_registry" \
			node $(CLI_ENTRY) publish $(SKILL_PATH) --registry $(REGISTRY_URL); \
	elif [ -n "$$npm_userconfig" ]; then \
		env \
			SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
			SKILLS_REGISTRY_TOKEN=$(PUBLISHER_TOKEN) \
			HOME=$(CLI_HOME) \
			XDG_CACHE_HOME=$(CLI_CACHE) \
			XDG_STATE_HOME=$(CLI_STATE) \
			NPM_CONFIG_USERCONFIG="$$npm_userconfig" \
			node $(CLI_ENTRY) publish $(SKILL_PATH) --registry $(REGISTRY_URL); \
	elif [ -n "$$npm_registry" ]; then \
		env \
			SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
			SKILLS_REGISTRY_TOKEN=$(PUBLISHER_TOKEN) \
			HOME=$(CLI_HOME) \
			XDG_CACHE_HOME=$(CLI_CACHE) \
			XDG_STATE_HOME=$(CLI_STATE) \
			npm_config_registry="$$npm_registry" \
			node $(CLI_ENTRY) publish $(SKILL_PATH) --registry $(REGISTRY_URL); \
	else \
		env \
			SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
			SKILLS_REGISTRY_TOKEN=$(PUBLISHER_TOKEN) \
			HOME=$(CLI_HOME) \
			XDG_CACHE_HOME=$(CLI_CACHE) \
			XDG_STATE_HOME=$(CLI_STATE) \
			node $(CLI_ENTRY) publish $(SKILL_PATH) --registry $(REGISTRY_URL); \
	fi

batch-import:
	@test -n "$(MANIFEST)" || (echo "MANIFEST is required, e.g. make batch-import MANIFEST=/abs/path/to/import.json [NPM_BACKEND=verdaccio|verdaccio-auth|nexus]" && exit 1)
	@npm_userconfig=""; \
	npm_registry_arg=""; \
	case "$(NPM_BACKEND)" in \
		direct|"") ;; \
		verdaccio) npm_registry_arg="--npm-registry $(VERDACCIO_URL)" ;; \
		verdaccio-auth) npm_userconfig="$(VERDACCIO_AUTH_NPMRC)"; npm_registry_arg="--npm-registry $(VERDACCIO_AUTH_URL)" ;; \
		nexus) npm_userconfig="$(NEXUS_NPMRC)"; npm_registry_arg="--npm-registry $(NEXUS_REPOSITORY_URL)/" ;; \
		*) echo "Unsupported NPM_BACKEND: $(NPM_BACKEND). Use direct, verdaccio, verdaccio-auth, or nexus." >&2; exit 1 ;; \
	esac; \
	if [ -n "$$npm_userconfig" ]; then \
		env \
			SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
			SKILLS_REGISTRY_TOKEN=$(PUBLISHER_TOKEN) \
			NPM_CONFIG_USERCONFIG="$$npm_userconfig" \
			UV_CACHE_DIR=$(UV_CACHE_DIR) \
			uv run python tools/batch-import/batch_import_skills.py --manifest $(MANIFEST) --catalog-url $(REGISTRY_URL) $$npm_registry_arg $(if $(STRICT),--strict); \
	else \
		env \
			SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
			SKILLS_REGISTRY_TOKEN=$(PUBLISHER_TOKEN) \
			UV_CACHE_DIR=$(UV_CACHE_DIR) \
			uv run python tools/batch-import/batch_import_skills.py --manifest $(MANIFEST) --catalog-url $(REGISTRY_URL) $$npm_registry_arg $(if $(STRICT),--strict); \
	fi

clean-dev:
	rm -rf $(CLI_HOME) $(CLI_CACHE) $(UV_CACHE_DIR) .venv /private/tmp/skill-registry.db /private/tmp/skill-registry.db-shm /private/tmp/skill-registry.db-wal $(VERDACCIO_STATE_DIR) $(VERDACCIO_AUTH_STATE_DIR) $(NEXUS_STATE_DIR)
