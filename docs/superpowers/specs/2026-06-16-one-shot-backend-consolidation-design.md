# One-Shot Backend Consolidation Design

Date: 2026-06-16
Status: Approved in conversation, pending written-spec review

## Context

The backend has already moved part of its runtime onto the newer `internal/*`
module graph:

- `internal/platform/*` exists
- `internal/catalog/*` exists
- `internal/packaging/*` exists
- `internal/governance/http` exists
- `internal/system/http` exists
- `internal/backend/runtime.go` exists

The current problem is no longer "what should the target structure be." The
problem is that the repository still carries multiple active stories for the
same backend concepts:

- `app/*`
- `business/*`
- `foundation/*`
- `internal/*`

That duplication keeps file count high, makes navigation expensive, and forces a
maintainer to bounce between compatibility files, duplicated vocabulary, and old
coordinator seams before reaching the retained implementation.

The user explicitly chose an aggressive cleanup standard and then chose a
one-shot merge rather than incremental phase work. This design therefore treats
the remaining backend cleanup as a full consolidation onto one retained module
graph.

## Goals

- Eliminate the parallel backend stories under `app/*`, `business/*`, and
  `foundation/*`
- Make `internal/*` the only retained backend structure for domain and platform
  concerns
- Keep the external HTTP contract stable for Web UI and CLI consumers
- Reduce backend file count by deleting obsolete facade, alias, forwarding, and
  duplicated-vocabulary files
- Concentrate runtime composition into one explicit owner instead of preserving
  the old `app/catalogapi` coordinator story

## Non-Goals

- No intentional HTTP route changes
- No intentional request or response shape changes
- No frontend or CLI redesign
- No database schema redesign
- No new long-lived compatibility alias layer

## Hard Constraints

- `go test ./...` must pass at the end of the consolidation
- Public auth behavior must remain stable
- Public install, publish, governance, audit, and session behavior must remain
  stable
- No concept may keep two long-term owners across `business/*` and `internal/*`
- No new temporary alias files may be introduced as the final retained seam

## Design Principle

This cleanup keeps exactly one retained story for each backend concept.

The retained navigation model is:

- HTTP entrypoints live in `internal/*/http`
- workflow orchestration lives in `internal/*/application`
- business vocabulary, invariants, and seams live in `internal/*/domain`
- concrete adapters live in `internal/*/infra` or `internal/platform/*`
- runtime composition lives in `internal/backend` and `cmd/skill-registry`

Any module that only preserves an old import path, re-exports a type, or
forwards into the retained implementation fails the deletion test and must not
survive the consolidation.

## Target Structure

```text
cmd/
  skill-registry/

internal/
  backend/

  platform/
    artifact/
    auth/
    http/
    observability/
    persistence/

  catalog/
    domain/
    application/
    http/
    infra/

  packaging/
    domain/
    application/
    http/
    infra/

  governance/
    http/

  system/
    http/
```

## Retained Owners

### Shared platform concerns

These stay under `internal/platform/*`:

- auth adapters and session support
- shared HTTP concerns such as metadata, middleware, JSON helpers, rate
  limiting, and read-cache support
- shared persistence adapters
- shared artifact adapters
- shared observability adapters

### Catalog

These stay under `internal/catalog/*`:

- core catalog types and errors
- catalog repository and aggregate seams
- catalog read/write workflows
- catalog SQL and memory adapters

### Packaging

These stay under `internal/packaging/*`:

- packaging types, warnings, and bundle rules
- packaging workflow orchestration
- publish workflow orchestration
- workspace adapters and file-system-backed helpers

### Runtime composition

`internal/backend/runtime.go` and `cmd/skill-registry/main.go` become the only
retained runtime-composition story. They may assemble modules, but they must not
preserve old business workflow wrappers just to keep `app/catalogapi` alive.

## Delete Set

### Delete entire directories

The following directories should not survive the consolidation:

- `app/auth`
- `app/observability`
- `foundation/artifact`
- `foundation/persist`
- `business/catalog`
- `business/packaging`

These are deleted after all real behavior has been re-homed to the retained
owners below.

### Collapse `app/catalogapi`

`app/catalogapi` is not deleted blindly. It still contains some real behavior,
so the directory is collapsed by re-homing its surviving responsibilities and
then deleting the leftover coordinator story.

After consolidation, `app/catalogapi` should either disappear entirely or be
reduced to zero backend ownership. The preferred outcome is full deletion.

## Migration Boundaries

### `app/auth` and `app/observability`

These move into `internal/platform/auth` and
`internal/platform/observability`. No compatibility wrappers remain.

### `foundation/artifact` and `foundation/persist`

These move into `internal/platform/artifact` and
`internal/platform/persistence`. They are treated as shared adapters, not
independent product-domain stories.

### `business/catalog`

Responsibilities are re-homed as follows:

- vocabulary, errors, repository seams, and aggregate seams ->
  `internal/catalog/domain`
- read and write workflows -> `internal/catalog/application`
- SQL, memory, and state-store adapters -> `internal/catalog/infra`

Once migrated, `business/catalog` is deleted without keeping aliases.

### `business/packaging`

Responsibilities are re-homed as follows:

- types, warnings, and bundle rules -> `internal/packaging/domain`
- service orchestration, publish workflow, and workspace orchestration ->
  `internal/packaging/application`
- workspace source, workspace backing, pack helpers, metadata, and locking
  adapters -> `internal/packaging/infra`

Once migrated, `business/packaging` is deleted without keeping aliases.

### `app/catalogapi`

`app/catalogapi` currently mixes several kinds of responsibility:

- server/runtime composition
- session handling
- audit querying and recording
- old router/common/middleware/read-cache/request-limiter story

These are re-homed with the following rules:

- shared HTTP concerns such as `common/*`, `middleware/*`, `response/*`,
  `request_limiter.go`, and `read_cache.go` move into `internal/platform/http`
- routing ownership moves to domain `internal/*/http/routes.go` modules instead
  of one old top-level router story
- session behavior moves into a retained owner such as
  `internal/platform/auth` plus a narrow runtime assembly point
- audit query/recording behavior moves into an explicit retained owner instead
  of remaining under `app/catalogapi` by default
- `server.go`, `server_workflows.go`, `server_lifecycle.go`, and
  `server_mode.go` may retain runtime-composition behavior only until that
  behavior is absorbed by `internal/backend/runtime.go`; they must not remain as
  an old business coordinator story

## Execution Strategy

The user chose a one-shot merge, but execution still needs disciplined internal
phases so failures remain diagnosable.

### Phase 1: Map each old behavior to a retained owner

Before deleting files, identify where every surviving behavior from
`app/catalogapi`, `business/catalog`, and `business/packaging` will live in the
retained structure.

Purpose:

- prevent orphaning a workflow during the merge
- avoid discovering missing ownership only after deletion

### Phase 2: Migrate real behavior first

Move surviving behavior into retained owners and update callers before removing
the old files.

This includes at least:

- session behavior
- audit behavior
- catalog workflows
- packaging workflows
- runtime assembly that still lives in old coordinator files

Purpose:

- make the retained seam complete before old files vanish

### Phase 3: Delete the old stories

Once imports and runtime assembly use the retained seams, delete the obsolete
directories and files in one sweep:

- `app/auth`
- `app/observability`
- `foundation/*`
- `business/catalog`
- `business/packaging`
- the remaining `app/catalogapi` files that no longer own behavior

### Phase 4: Import and vocabulary cleanup

Finish by enforcing these end-state rules:

- no remaining imports from deleted directories
- no duplicated long-term owners for the same concept
- no compatibility facade files reintroduced during the merge

## Verification Strategy

This design does not allow verification only at the final step. Focused package
tests must run during the migration as real behavior moves or old stories are
deleted.

Minimum verification coverage:

- `go test ./app/catalogapi/...`
- `go test ./business/catalog ./business/packaging`
  until those directories are deleted
- `go test ./internal/...`
- `go test ./cmd/skill-registry`
- final end-state verification: `go test ./...`

Each migration batch should run the narrowest relevant package tests first,
followed by the final full-suite verification once the old graph is gone.

## Risks

### Compile-break risk

One-shot import churn can break many packages at once.

Mitigation:

- migrate behavior before deletion
- run focused tests after each migration batch

### Hidden-behavior risk

Some files that look like facades may still own small but important behavior.

Mitigation:

- inspect surviving workflow files before deletion
- move real behavior explicitly into the retained owner

### Regression-diagnosis risk

If verification happens only at the end, it becomes hard to tell whether a
failure came from auth/session, routing, catalog workflows, or packaging
workflows.

Mitigation:

- keep internal execution phases even though the user chose a one-shot merge
- run focused package tests after each behavior migration or deletion batch

## Success Criteria

The consolidation is successful when all of the following are true:

- backend navigation starts from `internal/*`, `internal/backend`, and `cmd/*`
  only
- `app/*`, `business/*`, and `foundation/*` no longer own the backend's primary
  behavior
- repo-internal imports no longer depend on obsolete facade, alias, or old
  coordinator paths
- duplicated catalog and packaging vocabulary is removed from the old module
  graph
- `go test ./...` passes

At that point the backend problem changes from "too many parallel stories" to
"how to deepen the retained internal modules further," which is a smaller and
clearer follow-up problem.
