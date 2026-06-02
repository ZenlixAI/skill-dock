# Go Catalog Structure Alignment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reorganize the Go Catalog service toward `cmd / app / business / foundation / zarf / tools`, starting with asset relocation and path cleanup without changing runtime behavior.

**Architecture:** The migration is staged. Phase 1 relocates deployment assets and operational tools into `zarf/` and `tools/`, updates all repo entry points, and keeps behavior identical. Later phases can move Go packages after path churn is removed from the top level.

**Tech Stack:** Go, Make, Docker, Docker Compose, zsh, Markdown docs

---

### Task 1: Move deployment assets into `zarf`

**Files:**
- Create: `zarf/docker/Dockerfile`
- Create: `zarf/docker/verdaccio/config.yaml`
- Create: `zarf/compose/production.yml`
- Create: `zarf/compose/local.yml`
- Create: `zarf/scripts/run-verdaccio.sh`
- Create: `zarf/scripts/verdaccio-login.sh`
- Delete: `Dockerfile`
- Delete: `docker/verdaccio/config.yaml`
- Delete: `docker-compose.production.yml`
- Delete: `docker-compose.local.yml`
- Delete: `scripts/run_verdaccio.sh`
- Delete: `scripts/verdaccio_login.sh`

- [ ] **Step 1: Move Docker build asset into `zarf/docker`**

Mirror the existing root `Dockerfile` content into `zarf/docker/Dockerfile` and delete the root file.

- [ ] **Step 2: Move compose files into `zarf/compose`**

Mirror the current production/local compose files into `zarf/compose/production.yml` and `zarf/compose/local.yml`, then delete the root copies.

- [ ] **Step 3: Move Verdaccio config and shell helpers into `zarf`**

Place Verdaccio YAML under `zarf/docker/verdaccio/config.yaml` and shell helpers under `zarf/scripts/`.

- [ ] **Step 4: Verify file tree shape**

Run: `find zarf -maxdepth 3 -type f | sort`
Expected: Docker, compose, and Verdaccio helper files all live under `zarf/`

### Task 2: Move operational batch import tooling into `tools`

**Files:**
- Create: `tools/batch-import/batch_import_skills.py`
- Create: `tools/batch-import/example.json`
- Delete: `scripts/batch_import_skills.py`
- Delete: `scripts/batch-import.example.json`

- [ ] **Step 1: Move batch import Python entrypoint**

Mirror the current script into `tools/batch-import/batch_import_skills.py` and remove the old root `scripts/` copy.

- [ ] **Step 2: Move manifest example**

Mirror the current example manifest into `tools/batch-import/example.json` and remove the old root `scripts/` copy.

- [ ] **Step 3: Verify new tool location**

Run: `uv run python tools/batch-import/batch_import_skills.py --help`
Expected: usage text prints successfully

### Task 3: Repoint repo entry paths

**Files:**
- Modify: `Makefile`
- Modify: `.dockerignore`

- [ ] **Step 1: Update Makefile targets to use `zarf/` and `tools/`**

Change Docker build to `docker build -f zarf/docker/Dockerfile ...`, compose targets to `zarf/compose/*.yml`, Verdaccio helpers to `zarf/scripts/*`, and batch import targets to `tools/batch-import/*`.

- [ ] **Step 2: Update `.dockerignore` if any path assumptions changed**

Keep ignore coverage correct for the new top-level directories.

- [ ] **Step 3: Verify Make target help output**

Run: `make help`
Expected: targets still render and mention working commands

### Task 4: Update documentation path references

**Files:**
- Modify: `README.md`
- Modify: `docs/production-packaging-and-customer-usage.zh-CN.md`
- Modify: `docs/batch-import-skills.zh-CN.md`
- Modify: `docs/local-verdaccio-e2e.zh-CN.md`
- Modify: `docs/local-verdaccio-auth-e2e.zh-CN.md`

- [ ] **Step 1: Update path references from root assets to `zarf/` / `tools/`**

Replace all user-facing file references so docs match the new layout.

- [ ] **Step 2: Keep wording aligned with the structure design**

Make sure docs consistently explain `zarf` as deployment/runtime assets and `tools` as operational tooling.

- [ ] **Step 3: Verify no stale references remain**

Run: `rg -n "scripts/run_verdaccio|scripts/verdaccio_login|scripts/batch_import|batch-import.example|docker-compose\\.production|docker-compose\\.local|Dockerfile" README.md docs Makefile`
Expected: only new `zarf/` / `tools/` paths remain, except where root `Dockerfile` is explicitly discussed as historical context

### Task 5: Verify migration and commit

**Files:**
- Modify: `docs/superpowers/plans/2026-06-02-go-catalog-structure-alignment-implementation.md`

- [ ] **Step 1: Run focused verification**

Run:

```bash
go test ./cmd/skill-registry ./internal/httpapi ./internal/catalog -run 'TestMockNPMRegistryCanBeDisabled|TestSQLRepositoryCanSkipSeeding|TestMockNPMRegistryServesPackageMetadata'
uv run python tools/batch-import/batch_import_skills.py --help >/dev/null
ruby -e "require 'yaml'; ['zarf/compose/production.yml','zarf/compose/local.yml','zarf/docker/verdaccio/config.yaml'].each { |path| YAML.load_file(path) }; puts 'yaml-ok'"
```

Expected:

- Go tests pass
- Python help exits 0
- Ruby prints `yaml-ok`

- [ ] **Step 2: Review diff shape**

Run: `git diff --stat`
Expected: asset moves plus path rewrites, without unrelated logic churn

- [ ] **Step 3: Commit**

```bash
git add Makefile .dockerignore README.md docs zarf tools
git commit -m "Reorganize deployment assets and tools"
```
