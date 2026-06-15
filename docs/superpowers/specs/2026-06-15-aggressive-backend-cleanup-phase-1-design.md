# Aggressive Backend Cleanup Phase 1 Design

Date: 2026-06-15
Status: Approved in conversation, pending written-spec review

## Context

The backend on `refactor/optimize` is already partway through a reorganization:

- `internal/platform/*` exists
- `internal/catalog/*` exists
- `internal/packaging/*` exists
- `internal/governance/http` exists
- `internal/system/http` exists
- `internal/backend/runtime.go` exists

The test suite currently passes with `go test ./...`.

The remaining friction is not that the backend lacks a target structure. The friction is that the repo still carries two active module graphs at once:

- the new `internal/*` modules
- the old `app/*`, `business/*`, and `foundation/*` compatibility modules

That duplication keeps file count high and keeps navigation expensive. A maintainer still has to bounce through alias files, forwarding files, and old coordinator modules before reaching the real implementation.

## Goals

- Reduce backend file count by deleting shallow compatibility modules
- Reduce import-path duplication so backend code reads through one primary seam
- Keep the external HTTP contract stable for Web and CLI
- Preserve current behavior while concentrating request flow into the newer internal modules
- Make the next cleanup step on `app/catalogapi` smaller and more mechanical

## Non-Goals

- No intentional HTTP route changes
- No intentional request or response shape changes
- No frontend or CLI redesign
- No database schema changes
- No broad domain rewrite in one pass

## Hard Constraints

- `go test ./...` must stay green through the cleanup
- Public auth behavior must remain stable
- Public install, publish, governance, audit, and session behavior must remain stable
- Compatibility seams may be deleted only after all in-repo callers move to the retained seam

## Problem Statement

The backend is still paying for shallow modules that no longer provide meaningful leverage:

- `app/auth/*` is mostly a compatibility facade over `internal/platform/auth`
- `app/observability/*` is mostly a compatibility facade over `internal/platform/observability`
- `foundation/artifact/*` and `foundation/persist/*` are mostly compatibility facades over `internal/platform/*`
- `app/catalogapi/binding/*` and most of `app/catalogapi/handlers/*` are compatibility facades over `internal/*/http`
- `business/catalog/{types.go,errors.go,repository.go}` and `business/packaging/{service.go,types.go}` duplicate domain vocabulary that already exists in `internal/*`

These modules are shallow:

- their interface is nearly identical to the implementation
- deleting them would not reintroduce business complexity across many callers
- they mostly preserve old import paths rather than owning real behavior

The remaining large friction point is `app/catalogapi`. It is smaller than before, but it still acts as an old coordinator module:

- `server.go`
- `server_workflows.go`
- `catalog_read_module.go`
- `access_workflow.go`
- `session_module.go`
- `router/*`

So the repo still has two active stories for request flow:

1. internal domain HTTP modules
2. old top-level catalogapi coordination

## Design Principle

Phase 1 should delete the shallow compatibility modules first, then leave one clear surviving seam for each concept.

The rule is simple:

- if a module is only aliasing types, re-exporting constructors, or forwarding one call into the real implementation, it should be deleted
- if a module still owns meaningful orchestration or compatibility behavior that the runtime depends on, it may survive Phase 1 but should be narrowed

## Scope

Phase 1 covers two ordered slices.

### Slice A: Delete compatibility facade modules

Delete or absorb modules that exist only to preserve old import paths:

- `app/auth/*`
- `app/observability/*`
- `foundation/artifact/*`
- `foundation/persist/*`
- `app/catalogapi/binding/*`
- `app/catalogapi/handlers/catalog/handlers.go`
- `app/catalogapi/handlers/governance/handlers.go`
- `app/catalogapi/handlers/packaging/handlers.go`
- `app/catalogapi/handlers/system/handlers.go`

Narrow or defer modules that still carry real behavior:

- `app/catalogapi/handlers/session/handlers.go`
- `business/catalog/read_module.go`
- `business/catalog/write_module.go`
- `business/catalog/catalog_repository.go`
- `business/catalog/sql_repository.go`
- `business/catalog/sql_state_store.go`
- `business/catalog/memory_repository.go`
- `business/packaging/published_release_workflow.go`
- `business/packaging/workspace_runtime_module.go`
- `business/packaging/workspace_source_module.go`

### Slice B: Move in-repo imports onto the retained seam

The surviving seam after Slice A should be:

- `internal/platform/*` for shared auth, observability, persistence, artifact helpers
- `internal/catalog/*` for catalog domain/application/http/infra concerns
- `internal/packaging/*` for packaging domain/application/http/infra concerns
- `internal/governance/http` for governance HTTP handling
- `internal/system/http` for system HTTP handling
- `internal/backend/runtime.go` for runtime composition

Repo-internal callers should stop using the deleted compatibility paths.

## Explicit Delete Set

The strongest immediate delete candidates are:

- `app/auth/authorizer.go`
- `app/auth/context.go`
- `app/auth/session_manager.go`
- `app/observability/http.go`
- `app/observability/metrics.go`
- `foundation/artifact/publisher.go`
- `foundation/artifact/scan.go`
- `foundation/artifact/sources.go`
- `foundation/persist/audit_store.go`
- `foundation/persist/idempotency_store.go`
- `foundation/persist/store_set.go`
- `foundation/persist/workspace_store.go`
- `app/catalogapi/binding/catalog.go`
- `app/catalogapi/binding/packaging.go`
- `app/catalogapi/handlers/catalog/handlers.go`
- `app/catalogapi/handlers/governance/handlers.go`
- `app/catalogapi/handlers/packaging/handlers.go`
- `app/catalogapi/handlers/system/handlers.go`

These files fail the deletion test: deleting them concentrates almost no business logic, because the real implementation already lives elsewhere.

## Deferred Set

Phase 1 should not aggressively delete these yet:

- `app/catalogapi/server.go`
- `app/catalogapi/server_workflows.go`
- `app/catalogapi/catalog_read_module.go`
- `app/catalogapi/access_workflow.go`
- `app/catalogapi/session_module.go`
- `app/catalogapi/router/*`
- `app/catalogapi/handlers/session/handlers.go`
- most behavior-owning files under `business/catalog/*`
- most behavior-owning files under `business/packaging/*`

Reason:

- they still own request-flow orchestration, session behavior, repository behavior, or workflow behavior
- deleting them first would mix real orchestration rewrites with low-risk compatibility cleanup

That work becomes Phase 2.

## Import Migration Rules

After Slice A:

- `cmd/skill-registry/main.go` should import `internal/platform/auth` and `internal/backend`, not `app/auth` or `foundation/*`
- `internal/backend/runtime.go` should import retained internal platform modules directly, not `business/packaging` or `foundation/*` compatibility helpers where avoidable
- `internal/*/http` should depend on retained internal seams or retained behavior-owning modules only
- new code must not add imports back to deleted compatibility paths

## Phase Boundary

Phase 1 ends when these conditions are true:

- the delete set is removed
- all in-repo imports are updated to retained seams
- tests pass
- no new compatibility alias modules are introduced to replace the deleted ones

Phase 2 can then focus on shrinking or deleting the remaining `app/catalogapi` coordinator module and the duplicated `business/*` vocabulary with less noise.

## Risks

### Import churn risk

Many files will change mechanically. The risk is broad compile breakage rather than behavioral regression.

Mitigation:

- delete in small batches grouped by seam
- run focused package tests after each seam cleanup

### Hidden behavior risk

Some modules that look like facades may still contain small but real behavior.

Mitigation:

- inspect each delete candidate before removal
- if a candidate owns behavior, move that behavior into the retained module first and then delete the wrapper

### Test indirection risk

Some tests currently prove compatibility aliases rather than real behavior.

Mitigation:

- keep or move only the tests that protect real retained seams
- delete alias-only tests together with the alias modules

## Verification Strategy

At minimum, each cleanup batch should re-run:

- `go test ./app/catalogapi/...`
- `go test ./business/catalog ./business/packaging`
- `go test ./internal/...`
- `go test ./cmd/skill-registry`

End-state verification:

- `go test ./...`

## Success Criteria

Phase 1 is successful when:

- backend navigation starts primarily from `internal/*`
- obviously shallow compatibility modules are gone
- file count drops meaningfully without product behavior changes
- Phase 2 can target the remaining `app/catalogapi` coordinator and `business/*` duplication as a smaller, clearer problem
