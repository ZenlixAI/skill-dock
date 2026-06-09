# Backend Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix backend workspace persistence, read-path pagination/auth defects, bundle memory pressure, and duplicated SQL handle setup without changing web scope or CLI command semantics.

**Architecture:** Keep the current domain seams, but push authorization into SQL-backed list queries, derive managed workspace paths from workspace handles plus configured roots, and stream bundle files where the current implementation eagerly materializes bytes. Reuse one shared `*sql.DB` across catalog and persistence layers.

**Tech Stack:** Go, `database/sql`, SQLite/Postgres adapters, net/http, existing repository/store abstractions

---

### Task 1: Shared Workspace Root And Canonical Handle Resolution

**Files:**
- Modify: `business/packaging/workspace_persistence.go`
- Modify: `business/packaging/workspace_adapter.go`
- Modify: `business/packaging/workspace_runtime_module.go`
- Modify: `cmd/skill-registry/main.go`
- Test: `business/packaging/workspace_runtime_module_test.go`

- [ ] Add failing tests covering managed workspace reopen across different configured roots that share the same workspace handle and filesystem content.
- [ ] Run only the new packaging tests and confirm they fail for the expected canonical-path mismatch.
- [ ] Implement canonical runtime path derivation from `workspaceID + workspaceRoot + target` instead of trusting persisted absolute paths as the source of truth.
- [ ] Add shared workspace root configuration to backend startup and wire it into `packaging.NewServiceWithWorkspaceStore`.
- [ ] Re-run the targeted packaging tests and confirm they pass.

### Task 2: Shared SQL Handle Wiring

**Files:**
- Modify: `business/catalog/sql_repository.go`
- Modify: `foundation/persist/store_set.go`
- Modify: `cmd/skill-registry/main.go`
- Test: `business/catalog/sql_repository_test.go`
- Test: `foundation/persist/store_set_test.go`

- [ ] Add failing tests that exercise constructors through injected shared `*sql.DB` usage instead of opening duplicate handles internally.
- [ ] Run the targeted constructor tests and verify the current API shape cannot satisfy them.
- [ ] Introduce shared-DB constructor paths for catalog repository and persistence stores.
- [ ] Update main wiring to open one `*sql.DB` and build both catalog and store set from it.
- [ ] Re-run the targeted tests and confirm they pass.

### Task 3: Package-Scoped Pagination Correctness

**Files:**
- Modify: `app/catalogapi/catalog_read_module.go`
- Modify: `business/catalog/query_store.go`
- Modify: `business/catalog/read_module.go`
- Modify: `business/catalog/sql_state_store.go`
- Modify: `business/catalog/memory_aggregate_state_store.go`
- Test: `app/catalogapi/server_test.go`
- Test: `business/catalog/read_module_test.go`
- Test: `business/catalog/sql_repository_test.go`

- [ ] Add failing tests showing package-scoped principals receive underfilled or incorrect paginated catalog pages when filtering happens after query execution.
- [ ] Run the targeted catalog tests and confirm the failure is caused by post-query filtering.
- [ ] Extend read/query APIs so list queries can accept allowed package patterns.
- [ ] Implement SQL-layer filtering for exact and prefix package scopes before pagination.
- [ ] Re-run the targeted tests and confirm page contents and page metadata are correct.

### Task 4: Curated And Audit SQL Pagination Cleanup

**Files:**
- Modify: `business/catalog/sql_state_store.go`
- Modify: `foundation/persist/audit_store.go`
- Test: `business/catalog/sql_repository_test.go`
- Test: `foundation/persist/store_set_test.go`
- Test: `app/catalogapi/audit_query_module_test.go`

- [ ] Add failing tests for curated skill pagination and audit pagination/filter combinations that expect SQL-backed paging behavior.
- [ ] Run the targeted tests and verify they fail against current in-memory or generic-query behavior.
- [ ] Rewrite curated and audit query builders to compose only active filters and use `LIMIT/OFFSET` windows plus `pageSize+1`.
- [ ] Ensure indexes and ordering remain compatible with the rewritten SQL.
- [ ] Re-run the targeted tests and confirm they pass.

### Task 5: Install-Target Query Optimization

**Files:**
- Modify: `business/catalog/query_store.go`
- Modify: `business/catalog/read_module.go`
- Modify: `business/catalog/sql_state_store.go`
- Modify: `business/catalog/memory_aggregate_state_store.go`
- Test: `business/catalog/read_module_test.go`
- Test: `business/catalog/sql_repository_test.go`

- [ ] Add failing tests for install-target resolution that expect direct lookup of exact or best resolvable version without loading all versions first.
- [ ] Run the targeted install-target tests and confirm the current query-store contract is insufficient.
- [ ] Add direct query-store methods for exact-version lookup and best resolvable latest-version lookup.
- [ ] Update read-module resolution to use those methods while preserving existing error semantics.
- [ ] Re-run the targeted tests and confirm they pass.

### Task 6: Bundle Publish And Download Streaming

**Files:**
- Modify: `app/catalogapi/packaging_access_module.go`
- Modify: `app/catalogapi/server.go`
- Modify: `business/packaging/service.go`
- Modify: `business/packaging/workspace_runtime_module.go`
- Modify: `business/packaging/workspace_adapter.go`
- Modify: `foundation/artifact/publisher.go`
- Test: `app/catalogapi/packaging_access_module_test.go`
- Test: `foundation/artifact/sources_test.go`
- Test: `business/packaging/published_release_workflow_test.go`

- [ ] Add failing tests that expect bundle download to stream from a file path and publish workflow to hand a tarball path into the publisher rather than preloading bundle bytes.
- [ ] Run the targeted packaging/artifact tests and confirm they fail on the current byte-slice path.
- [ ] Add file-path based bundle access and tarball publish inputs while keeping backward-compatible fallbacks where needed for tests.
- [ ] Implement streamed npm publish request generation and streamed bundle download response writing.
- [ ] Re-run the targeted tests and confirm they pass.

### Task 7: Full Verification

**Files:**
- Test: `./...`
- Test: `apps/skills-cli`

- [ ] Run `go test ./...` and confirm all backend tests pass.
- [ ] Run `npm test` in `apps/skills-cli` and confirm CLI tests still pass.
- [ ] Review the resulting diff for unintended scope expansion before reporting completion.
