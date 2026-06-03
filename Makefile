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

.PHONY: help verify verify-full test test-go test-go-packaging python-sync test-cli test-cli-dist test-cli-packaging typecheck-cli build-cli web-install typecheck-web build-web web-dev docker-build compose-production-up compose-production-down compose-local-up compose-local-down compose-nexus-up compose-nexus-down registry-reset registry-run registry-run-fresh registry-run-mock verdaccio-run verdaccio-reset registry-run-verdaccio verdaccio-run-auth verdaccio-run-auth-strict verdaccio-login verdaccio-auth-reset nexus-run nexus-reset nexus-bootstrap nexus-bootstrap-strict registry-run-nexus registry-run-verdaccio-auth cli-find cli-find-json cli-add cli-add-nexus cli-publish cli-publish-verdaccio cli-publish-verdaccio-auth cli-publish-nexus batch-import batch-import-verdaccio batch-import-verdaccio-auth batch-import-nexus clean-dev

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
	@printf "Local Nexus:\n"
	@printf "  make nexus-run         Start local Nexus 3 on %s\n" "$(NEXUS_URL)"
	@printf "  make nexus-bootstrap   Change admin password, configure anonymous read, create npm hosted repo, and write %s\n" "$(NEXUS_NPMRC)"
	@printf "  make nexus-bootstrap-strict Same as nexus-bootstrap but disable anonymous read for tarball/metadata probes\n"
	@printf "  make nexus-reset       Remove local Nexus state under %s\n" "$(NEXUS_STATE_DIR)"
	@printf "  make registry-run-nexus Start Catalog against local Nexus npm repo %s\n\n" "$(NEXUS_REPOSITORY_URL)"
	@printf "Containers:\n"
	@printf "  make docker-build      Build the production Catalog image\n"
	@printf "  make compose-production-up   Run Catalog + Postgres via docker compose\n"
	@printf "  make compose-production-down Stop Catalog + Postgres compose stack\n"
	@printf "  make compose-local-up        Run Catalog + Postgres + Verdaccio locally\n"
	@printf "  make compose-local-down      Stop local compose stack\n"
	@printf "  make compose-nexus-up        Run Catalog + Postgres + Nexus locally\n"
	@printf "  make compose-nexus-down      Stop local Nexus compose stack\n\n"
	@printf "CLI against local registry:\n"
	@printf "  make cli-find QUERY=pdf\n"
	@printf "  make cli-find-json QUERY=pdf\n"
	@printf "  make cli-add PACKAGE=@platform.skill/pdf-skill   # installs into ./.agents/skills and updates skills-lock.json\n"
	@printf "  make cli-add-nexus PACKAGE=@platform.skill/pdf-skill  # reuses %s for authenticated Nexus tarball reads\n" "$(NEXUS_NPMRC)"
	@printf "  make cli-publish SKILL_PATH=/abs/path/to/skill\n"
	@printf "  make cli-publish-verdaccio SKILL_PATH=/abs/path/to/skill\n"
	@printf "  make cli-publish-verdaccio-auth SKILL_PATH=/abs/path/to/skill\n"
	@printf "  make cli-publish-nexus SKILL_PATH=/abs/path/to/skill\n\n"
	@printf "Batch import:\n"
	@printf "  make batch-import MANIFEST=/abs/path/to/import.json            # uses uv-managed Python\n"
	@printf "  make batch-import-verdaccio MANIFEST=/abs/path/to/import.json  # uses uv-managed Python\n"
	@printf "  make batch-import-verdaccio-auth MANIFEST=/abs/path/to/import.json\n"
	@printf "  make batch-import-nexus MANIFEST=/abs/path/to/import.json\n"
	@printf "  append STRICT=1 to any batch-import target to disable normalization during prepare\n\n"
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

compose-production-up:
	docker compose -f zarf/compose/production.yml up --build

compose-production-down:
	docker compose -f zarf/compose/production.yml down

compose-local-up:
	docker compose -f zarf/compose/local.yml up --build

compose-local-down:
	docker compose -f zarf/compose/local.yml down

compose-nexus-up:
	docker compose -f zarf/compose/nexus.yml up --build

compose-nexus-down:
	docker compose -f zarf/compose/nexus.yml down

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
		NEXUS_ENABLE_ANONYMOUS_READ=$(NEXUS_ENABLE_ANONYMOUS_READ) \
		NEXUS_NPMRC_PATH=$(NEXUS_NPMRC) \
		./zarf/scripts/nexus-bootstrap.sh

nexus-bootstrap-strict:
	env \
		NEXUS_ENABLE_ANONYMOUS_READ=false \
		$(MAKE) nexus-bootstrap

nexus-reset:
	rm -rf $(NEXUS_STATE_DIR)

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

registry-run-nexus:
	env \
		GOCACHE=$(GO_CACHE) \
		SKILL_REGISTRY_ADDR=$(REGISTRY_ADDR) \
		SKILL_REGISTRY_DATABASE_URL=$(REGISTRY_DB) \
		SKILL_REGISTRY_SAMPLE_NPM_BASE_URL=$(NEXUS_REPOSITORY_URL) \
		SKILL_REGISTRY_NPM_REGISTRY_URL=$(NEXUS_REPOSITORY_URL) \
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

cli-add-nexus:
	env \
		SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
		SKILLS_REGISTRY_TOKEN=$(READER_TOKEN) \
		HOME=$(CLI_HOME) \
		XDG_CACHE_HOME=$(CLI_CACHE) \
		XDG_STATE_HOME=$(CLI_STATE) \
		NPM_CONFIG_USERCONFIG=$(NEXUS_NPMRC) \
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

cli-publish-nexus:
	@test -n "$(SKILL_PATH)" || (echo "SKILL_PATH is required, e.g. make cli-publish-nexus SKILL_PATH=/abs/path/to/skill" && exit 1)
	env \
		SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
		SKILLS_REGISTRY_TOKEN=$(PUBLISHER_TOKEN) \
		HOME=$(CLI_HOME) \
		XDG_CACHE_HOME=$(CLI_CACHE) \
		XDG_STATE_HOME=$(CLI_STATE) \
		NPM_CONFIG_USERCONFIG=$(NEXUS_NPMRC) \
		npm_config_registry=$(NEXUS_REPOSITORY_URL)/ \
		node $(CLI_ENTRY) publish $(SKILL_PATH) --registry $(REGISTRY_URL)

batch-import:
	@test -n "$(MANIFEST)" || (echo "MANIFEST is required, e.g. make batch-import MANIFEST=/abs/path/to/import.json" && exit 1)
	env \
		SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
		SKILLS_REGISTRY_TOKEN=$(PUBLISHER_TOKEN) \
		UV_CACHE_DIR=$(UV_CACHE_DIR) \
		uv run python tools/batch-import/batch_import_skills.py --manifest $(MANIFEST) --catalog-url $(REGISTRY_URL) $(if $(STRICT),--strict)

batch-import-verdaccio:
	@test -n "$(MANIFEST)" || (echo "MANIFEST is required, e.g. make batch-import-verdaccio MANIFEST=/abs/path/to/import.json" && exit 1)
	env \
		SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
		SKILLS_REGISTRY_TOKEN=$(PUBLISHER_TOKEN) \
		UV_CACHE_DIR=$(UV_CACHE_DIR) \
		uv run python tools/batch-import/batch_import_skills.py --manifest $(MANIFEST) --catalog-url $(REGISTRY_URL) --npm-registry $(VERDACCIO_URL) $(if $(STRICT),--strict)

batch-import-verdaccio-auth:
	@test -n "$(MANIFEST)" || (echo "MANIFEST is required, e.g. make batch-import-verdaccio-auth MANIFEST=/abs/path/to/import.json" && exit 1)
	env \
		SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
		SKILLS_REGISTRY_TOKEN=$(PUBLISHER_TOKEN) \
		NPM_CONFIG_USERCONFIG=$(VERDACCIO_AUTH_NPMRC) \
		UV_CACHE_DIR=$(UV_CACHE_DIR) \
		uv run python tools/batch-import/batch_import_skills.py --manifest $(MANIFEST) --catalog-url $(REGISTRY_URL) --npm-registry $(VERDACCIO_AUTH_URL) $(if $(STRICT),--strict)

batch-import-nexus:
	@test -n "$(MANIFEST)" || (echo "MANIFEST is required, e.g. make batch-import-nexus MANIFEST=/abs/path/to/import.json" && exit 1)
	env \
		SKILLS_REGISTRY_URL=$(REGISTRY_URL) \
		SKILLS_REGISTRY_TOKEN=$(PUBLISHER_TOKEN) \
		NPM_CONFIG_USERCONFIG=$(NEXUS_NPMRC) \
		UV_CACHE_DIR=$(UV_CACHE_DIR) \
		uv run python tools/batch-import/batch_import_skills.py --manifest $(MANIFEST) --catalog-url $(REGISTRY_URL) --npm-registry $(NEXUS_REPOSITORY_URL)/ $(if $(STRICT),--strict)

clean-dev:
	rm -rf $(CLI_HOME) $(CLI_CACHE) $(UV_CACHE_DIR) .venv /private/tmp/skill-registry.db /private/tmp/skill-registry.db-shm /private/tmp/skill-registry.db-wal $(VERDACCIO_STATE_DIR) $(VERDACCIO_AUTH_STATE_DIR) $(NEXUS_STATE_DIR)
