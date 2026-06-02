SHELL := /bin/zsh

CLI_DIR := apps/skills-cli
CLI_ENTRY := $(CLI_DIR)/src/cli.ts

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

READER_TOKEN ?= reader-secret
AUTHORING_TOKEN ?= authoring-secret
PUBLISHER_TOKEN ?= publisher-secret

PACKAGE ?= @platform.skill/pdf-skill
QUERY ?= pdf
SKILL_PATH ?=
MANIFEST ?=

.PHONY: help verify test test-go python-sync test-cli typecheck-cli build-cli docker-build compose-production-up compose-production-down compose-local-up compose-local-down registry-reset registry-run registry-run-fresh registry-run-mock verdaccio-run verdaccio-reset registry-run-verdaccio verdaccio-run-auth verdaccio-run-auth-strict verdaccio-login verdaccio-auth-reset registry-run-verdaccio-auth cli-find cli-find-json cli-add cli-publish cli-publish-verdaccio cli-publish-verdaccio-auth batch-import batch-import-verdaccio batch-import-verdaccio-auth clean-dev

help:
	@printf "skill_registry make targets\n\n"
	@printf "Verification:\n"
	@printf "  make verify            Run Go tests, CLI type-check, and CLI tests\n"
	@printf "  make test-go           Run go test ./...\n"
	@printf "  make python-sync       Create/update the uv-managed Python environment\n"
	@printf "  make typecheck-cli     Run npm type-check in %s\n" "$(CLI_DIR)"
	@printf "  make test-cli          Run npm test in %s\n" "$(CLI_DIR)"
	@printf "  make build-cli         Run npm build in %s\n\n" "$(CLI_DIR)"
	@printf "Local Catalog:\n"
	@printf "  make registry-reset    Remove the local Catalog sqlite files\n"
	@printf "  make registry-run      Start Catalog with sample tarballs served from local /mock-npm\n"
	@printf "  make registry-run-fresh Reset the local DB, then start Catalog with sample tarballs\n"
	@printf "  make registry-run-mock Start Catalog with tarballs and package metadata served from local /mock-npm\n\n"
	@printf "Local Verdaccio:\n"
	@printf "  make verdaccio-run     Start public local Verdaccio on %s\n" "$(VERDACCIO_URL)"
	@printf "  make verdaccio-reset   Remove local Verdaccio state under %s\n" "$(VERDACCIO_STATE_DIR)"
	@printf "  make registry-run-verdaccio Start Catalog against local Verdaccio\n\n"
	@printf "Authenticated Verdaccio:\n"
	@printf "  make verdaccio-run-auth Start Verdaccio with authenticated publish on %s\n" "$(VERDACCIO_AUTH_URL)"
	@printf "  make verdaccio-run-auth-strict Start Verdaccio with authenticated publish and read on %s\n" "$(VERDACCIO_AUTH_URL)"
	@printf "  make verdaccio-login   Run npm adduser against authenticated Verdaccio\n"
	@printf "  make verdaccio-auth-reset Remove authenticated Verdaccio state under %s\n" "$(VERDACCIO_AUTH_STATE_DIR)"
	@printf "  make registry-run-verdaccio-auth Start Catalog against authenticated Verdaccio\n\n"
	@printf "Containers:\n"
	@printf "  make docker-build      Build the production Catalog image\n"
	@printf "  make compose-production-up   Run Catalog + Postgres via docker compose\n"
	@printf "  make compose-production-down Stop Catalog + Postgres compose stack\n"
	@printf "  make compose-local-up        Run Catalog + Postgres + Verdaccio locally\n"
	@printf "  make compose-local-down      Stop local compose stack\n\n"
	@printf "CLI against local registry:\n"
	@printf "  make cli-find QUERY=pdf\n"
	@printf "  make cli-find-json QUERY=pdf\n"
	@printf "  make cli-add PACKAGE=@platform.skill/pdf-skill   # installs into ./.agents/skills and updates skills-lock.json\n"
	@printf "  make cli-publish SKILL_PATH=/abs/path/to/skill\n"
	@printf "  make cli-publish-verdaccio SKILL_PATH=/abs/path/to/skill\n"
	@printf "  make cli-publish-verdaccio-auth SKILL_PATH=/abs/path/to/skill\n\n"
	@printf "Batch import:\n"
	@printf "  make batch-import MANIFEST=/abs/path/to/import.json            # uses uv-managed Python\n"
	@printf "  make batch-import-verdaccio MANIFEST=/abs/path/to/import.json  # uses uv-managed Python\n"
	@printf "  make batch-import-verdaccio-auth MANIFEST=/abs/path/to/import.json\n\n"
	@printf "Cleanup:\n"
	@printf "  make clean-dev         Remove temp CLI, Catalog, Verdaccio, uv cache, and .venv state\n"

verify: test-go typecheck-cli test-cli

test: verify

test-go:
	env GOCACHE=$(GO_CACHE) go test ./...

python-sync:
	env UV_CACHE_DIR=$(UV_CACHE_DIR) uv sync

typecheck-cli:
	env HOME=$(CLI_HOME) XDG_CACHE_HOME=$(CLI_CACHE) npm --prefix $(CLI_DIR) run type-check

test-cli:
	env HOME=$(CLI_HOME) XDG_CACHE_HOME=$(CLI_CACHE) npm --prefix $(CLI_DIR) test

build-cli:
	env HOME=$(CLI_HOME) XDG_CACHE_HOME=$(CLI_CACHE) npm --prefix $(CLI_DIR) run build

docker-build:
	docker build -f zarf/docker/Dockerfile -t skill-registry:latest .

compose-production-up:
	docker compose -f zarf/compose/production.yml up --build

compose-production-down:
	docker compose -f zarf/compose/production.yml down

compose-local-up:
	docker compose -f zarf/compose/local.yml up --build

compose-local-down:
	docker compose -f zarf/compose/local.yml down

registry-reset:
	rm -f /private/tmp/skill-registry.db /private/tmp/skill-registry.db-shm /private/tmp/skill-registry.db-wal

registry-run:
	env \
		GOCACHE=$(GO_CACHE) \
		SKILL_REGISTRY_ADDR=$(REGISTRY_ADDR) \
		SKILL_REGISTRY_DATABASE_URL=$(REGISTRY_DB) \
		SKILL_REGISTRY_SAMPLE_NPM_BASE_URL=$(REGISTRY_URL)/mock-npm \
		SKILL_REGISTRY_READER_TOKEN=$(READER_TOKEN) \
		SKILL_REGISTRY_AUTHORING_TOKEN=$(AUTHORING_TOKEN) \
		SKILL_REGISTRY_PUBLISHER_TOKEN=$(PUBLISHER_TOKEN) \
		go run ./cmd/skill-registry

registry-run-fresh: registry-reset
	env \
		GOCACHE=$(GO_CACHE) \
		SKILL_REGISTRY_ADDR=$(REGISTRY_ADDR) \
		SKILL_REGISTRY_DATABASE_URL=$(REGISTRY_DB) \
		SKILL_REGISTRY_SAMPLE_NPM_BASE_URL=$(REGISTRY_URL)/mock-npm \
		SKILL_REGISTRY_READER_TOKEN=$(READER_TOKEN) \
		SKILL_REGISTRY_AUTHORING_TOKEN=$(AUTHORING_TOKEN) \
		SKILL_REGISTRY_PUBLISHER_TOKEN=$(PUBLISHER_TOKEN) \
		go run ./cmd/skill-registry

registry-run-mock:
	env \
		GOCACHE=$(GO_CACHE) \
		SKILL_REGISTRY_ADDR=$(REGISTRY_ADDR) \
		SKILL_REGISTRY_DATABASE_URL=$(REGISTRY_DB) \
		SKILL_REGISTRY_SAMPLE_NPM_BASE_URL=$(REGISTRY_URL)/mock-npm \
		SKILL_REGISTRY_NPM_REGISTRY_URL=$(REGISTRY_URL)/mock-npm \
		SKILL_REGISTRY_READER_TOKEN=$(READER_TOKEN) \
		SKILL_REGISTRY_AUTHORING_TOKEN=$(AUTHORING_TOKEN) \
		SKILL_REGISTRY_PUBLISHER_TOKEN=$(PUBLISHER_TOKEN) \
		go run ./cmd/skill-registry

verdaccio-run:
	env \
		VERDACCIO_ADDR=$(VERDACCIO_ADDR) \
		VERDACCIO_STATE_DIR=$(VERDACCIO_STATE_DIR) \
		./zarf/scripts/run-verdaccio.sh

verdaccio-reset:
	rm -rf $(VERDACCIO_STATE_DIR)

registry-run-verdaccio:
	env \
		GOCACHE=$(GO_CACHE) \
		SKILL_REGISTRY_ADDR=$(REGISTRY_ADDR) \
		SKILL_REGISTRY_DATABASE_URL=$(REGISTRY_DB) \
		SKILL_REGISTRY_SAMPLE_NPM_BASE_URL=$(VERDACCIO_URL) \
		SKILL_REGISTRY_NPM_REGISTRY_URL=$(VERDACCIO_URL) \
		SKILL_REGISTRY_READER_TOKEN=$(READER_TOKEN) \
		SKILL_REGISTRY_AUTHORING_TOKEN=$(AUTHORING_TOKEN) \
		SKILL_REGISTRY_PUBLISHER_TOKEN=$(PUBLISHER_TOKEN) \
		go run ./cmd/skill-registry

verdaccio-run-auth:
	env \
		VERDACCIO_ADDR=$(VERDACCIO_AUTH_ADDR) \
		VERDACCIO_STATE_DIR=$(VERDACCIO_AUTH_STATE_DIR) \
		VERDACCIO_CONFIG=$(PWD)/zarf/docker/verdaccio/auth.yaml \
		./zarf/scripts/run-verdaccio.sh

verdaccio-run-auth-strict:
	env \
		VERDACCIO_ADDR=$(VERDACCIO_AUTH_ADDR) \
		VERDACCIO_STATE_DIR=$(VERDACCIO_AUTH_STATE_DIR) \
		VERDACCIO_CONFIG=$(PWD)/zarf/docker/verdaccio/strict-auth.yaml \
		./zarf/scripts/run-verdaccio.sh

verdaccio-login:
	env \
		VERDACCIO_URL=$(VERDACCIO_AUTH_URL) \
		VERDACCIO_STATE_DIR=$(VERDACCIO_AUTH_STATE_DIR) \
		VERDACCIO_NPMRC_PATH=$(VERDACCIO_AUTH_NPMRC) \
		./zarf/scripts/verdaccio-login.sh

verdaccio-auth-reset:
	rm -rf $(VERDACCIO_AUTH_STATE_DIR)

registry-run-verdaccio-auth:
	env \
		GOCACHE=$(GO_CACHE) \
		SKILL_REGISTRY_ADDR=$(REGISTRY_ADDR) \
		SKILL_REGISTRY_DATABASE_URL=$(REGISTRY_DB) \
		SKILL_REGISTRY_SAMPLE_NPM_BASE_URL=$(VERDACCIO_AUTH_URL) \
		SKILL_REGISTRY_NPM_REGISTRY_URL=$(VERDACCIO_AUTH_URL) \
		SKILL_REGISTRY_READER_TOKEN=$(READER_TOKEN) \
		SKILL_REGISTRY_AUTHORING_TOKEN=$(AUTHORING_TOKEN) \
		SKILL_REGISTRY_PUBLISHER_TOKEN=$(PUBLISHER_TOKEN) \
		go run ./cmd/skill-registry

cli-find:
	env \
		SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
		SKILLS_REGISTRY_TOKEN=$(READER_TOKEN) \
		node $(CLI_ENTRY) find "$(QUERY)" --registry $(REGISTRY_URL)

cli-find-json:
	env \
		SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
		SKILLS_REGISTRY_TOKEN=$(READER_TOKEN) \
		node $(CLI_ENTRY) find "$(QUERY)" --registry $(REGISTRY_URL) --json

cli-add:
	env \
		SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
		SKILLS_REGISTRY_TOKEN=$(READER_TOKEN) \
		HOME=$(CLI_HOME) \
		XDG_CACHE_HOME=$(CLI_CACHE) \
		XDG_STATE_HOME=$(CLI_STATE) \
		node $(CLI_ENTRY) add $(PACKAGE) -y --registry $(REGISTRY_URL)

cli-publish:
	@test -n "$(SKILL_PATH)" || (echo "SKILL_PATH is required, e.g. make cli-publish SKILL_PATH=/abs/path/to/skill" && exit 1)
	env \
		SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
		SKILLS_REGISTRY_TOKEN=$(PUBLISHER_TOKEN) \
		HOME=$(CLI_HOME) \
		XDG_CACHE_HOME=$(CLI_CACHE) \
		XDG_STATE_HOME=$(CLI_STATE) \
		node $(CLI_ENTRY) publish $(SKILL_PATH) --registry $(REGISTRY_URL)

cli-publish-verdaccio:
	@test -n "$(SKILL_PATH)" || (echo "SKILL_PATH is required, e.g. make cli-publish-verdaccio SKILL_PATH=/abs/path/to/skill" && exit 1)
	env \
		SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
		SKILLS_REGISTRY_TOKEN=$(PUBLISHER_TOKEN) \
		HOME=$(CLI_HOME) \
		XDG_CACHE_HOME=$(CLI_CACHE) \
		XDG_STATE_HOME=$(CLI_STATE) \
		npm_config_registry=$(VERDACCIO_URL) \
		node $(CLI_ENTRY) publish $(SKILL_PATH) --registry $(REGISTRY_URL)

cli-publish-verdaccio-auth:
	@test -n "$(SKILL_PATH)" || (echo "SKILL_PATH is required, e.g. make cli-publish-verdaccio-auth SKILL_PATH=/abs/path/to/skill" && exit 1)
	env \
		SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
		SKILLS_REGISTRY_TOKEN=$(PUBLISHER_TOKEN) \
		HOME=$(CLI_HOME) \
		XDG_CACHE_HOME=$(CLI_CACHE) \
		XDG_STATE_HOME=$(CLI_STATE) \
		NPM_CONFIG_USERCONFIG=$(VERDACCIO_AUTH_NPMRC) \
		npm_config_registry=$(VERDACCIO_AUTH_URL) \
		node $(CLI_ENTRY) publish $(SKILL_PATH) --registry $(REGISTRY_URL)

batch-import:
	@test -n "$(MANIFEST)" || (echo "MANIFEST is required, e.g. make batch-import MANIFEST=/abs/path/to/import.json" && exit 1)
	env \
		SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
		SKILLS_REGISTRY_TOKEN=$(PUBLISHER_TOKEN) \
		UV_CACHE_DIR=$(UV_CACHE_DIR) \
		uv run python tools/batch-import/batch_import_skills.py --manifest $(MANIFEST) --catalog-url $(REGISTRY_URL)

batch-import-verdaccio:
	@test -n "$(MANIFEST)" || (echo "MANIFEST is required, e.g. make batch-import-verdaccio MANIFEST=/abs/path/to/import.json" && exit 1)
	env \
		SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
		SKILLS_REGISTRY_TOKEN=$(PUBLISHER_TOKEN) \
		UV_CACHE_DIR=$(UV_CACHE_DIR) \
		uv run python tools/batch-import/batch_import_skills.py --manifest $(MANIFEST) --catalog-url $(REGISTRY_URL) --npm-registry $(VERDACCIO_URL)

batch-import-verdaccio-auth:
	@test -n "$(MANIFEST)" || (echo "MANIFEST is required, e.g. make batch-import-verdaccio-auth MANIFEST=/abs/path/to/import.json" && exit 1)
	env \
		SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
		SKILLS_REGISTRY_TOKEN=$(PUBLISHER_TOKEN) \
		NPM_CONFIG_USERCONFIG=$(VERDACCIO_AUTH_NPMRC) \
		UV_CACHE_DIR=$(UV_CACHE_DIR) \
		uv run python tools/batch-import/batch_import_skills.py --manifest $(MANIFEST) --catalog-url $(REGISTRY_URL) --npm-registry $(VERDACCIO_AUTH_URL)

clean-dev:
	rm -rf $(CLI_HOME) $(CLI_CACHE) $(UV_CACHE_DIR) .venv /private/tmp/skill-registry.db /private/tmp/skill-registry.db-shm /private/tmp/skill-registry.db-wal $(VERDACCIO_STATE_DIR) $(VERDACCIO_AUTH_STATE_DIR)
