# Backend Remediation Design

## Scope

This design covers only the backend and CLI-adjacent server behavior under:

- `app/catalogapi`
- `business/catalog`
- `business/packaging`
- `foundation/artifact`
- `foundation/persist`
- `cmd/skill-registry`

Out of scope:

- `apps/skills-web`
- UI copy, page behavior, and frontend tests
- introducing an external object store
- changing public CLI command semantics

## Goals

1. Make managed packaging workspaces safe to use across multiple backend replicas that share the same filesystem content, even if the mount path differs per replica.
2. Remove avoidable full-buffer bundle handling from publish and download paths.
3. Fix package-scoped catalog pagination so authorization does not distort page contents or page metadata after the fact.
4. Remove avoidable full-table or in-memory pagination paths from curated catalog and audit queries.
5. Reduce avoidable read-path overhead in install-target resolution.
6. Reuse a single SQL handle for catalog, audit, idempotency, and workspace persistence.

## Non-Goals

1. Guarantee multi-replica packaging on nodes that do not share workspace filesystem content at all.
2. Replace the npm-compatible publish protocol with a different artifact upload protocol.
3. Add fuzzy search, FTS, or a new search engine.
4. Redesign domain boundaries beyond what is required to fix the defects above.

## Design

### 1. Shared workspace persistence

Managed workspace persistence will stop trusting persisted absolute filesystem paths as the canonical location for a workspace or bundle.

Instead:

- the persisted workspace handle remains the stable identifier exposed to callers
- runtime path resolution derives the current workspace directory and bundle path from:
  - the current configured workspace root
  - the workspace handle
  - the packaging target
- persisted path fields remain for backward compatibility and diagnostics, but are no longer the source of truth for runtime lookup

This fixes the current defect where one replica persists `/mount-a/ws_123/workspace` and another replica, using the same shared filesystem content at `/mount-b`, rejects the workspace because the absolute path differs.

The backend entrypoint will also accept an explicit shared workspace root configuration so deployments are not forced onto `os.TempDir()`.

### 2. Bundle publish and download memory behavior

Download will stop loading the full bundle into memory before writing the response.

Instead:

- catalog API download resolves a server-side bundle file path
- the handler streams that file to the client

Publish will stop reading the bundle into memory inside packaging before handing it to the npm publisher.

Instead:

- packaging passes a tarball file path into the publish adapter
- the npm publisher streams the request body from disk while base64-encoding the tarball attachment into the JSON payload
- byte-slice fallback remains available for tests and any existing in-memory callers

The npm publish request will still contain base64 attachment data because the registry protocol requires it, but the backend will no longer keep both the tarball bytes and the final request payload as duplicated long-lived buffers.

### 3. Package-scoped catalog pagination

Catalog read pagination will move package authorization constraints into the query layer for list endpoints.

For principals with `AllowedPackages`:

- exact package patterns become `package_name = ?`
- prefix patterns ending in `*` become `package_name LIKE 'prefix%'`

This filtering will happen inside SQL before `LIMIT/OFFSET`, so page contents and `hasNextPage` reflect the authorized result set instead of a larger hidden result set.

This change applies to:

- `ListSkills`
- `ListCuratedSkills`

Single-package endpoints already authorize directly on the requested package and do not need the same treatment.

### 4. SQL pagination and query cleanup

Two read paths currently do unnecessary full reads:

- curated skills loads all matching rows and paginates in memory
- audit events use a generic optional-filter SQL shape that is harder for indexes to serve well

They will be changed to:

- build SQL dynamically from only the active filters
- paginate in SQL with `LIMIT pageSize+1 OFFSET cursor`
- return page metadata from the loaded window

This preserves current API semantics while removing unnecessary in-memory growth and improving index use.

### 5. Install-target resolution

Install-target resolution currently loads all versions for a package and scans them in Go.

It will be replaced with direct query-store methods that:

- select the exact requested version when a version is supplied
- select the best resolvable published version directly in SQL when version is omitted

Governance lookup for replacement recommendations remains on the error path only.

### 6. Shared SQL handle

Backend startup will create one `*sql.DB` and pass it into:

- catalog state store
- audit store
- workspace store
- idempotency store

This removes duplicate pools against the same database URL and reduces lock contention risk, especially for SQLite.

## Testing Strategy

Each fix is implemented through failing tests first:

1. workspace persistence tests proving a workspace created under one root can be reopened under another root when the workspace handle and shared content are the same
2. startup/store wiring tests proving catalog and persistence layers reuse the same `*sql.DB`
3. catalog read tests proving package-scoped principals receive correctly paginated authorized pages
4. curated/audit store tests proving pagination is SQL-backed and stable
5. install-target tests proving omitted-version resolution does not require loading all versions
6. artifact publisher and catalog API tests proving publish/download operate on file paths or streamed responses without eager full-buffer service reads

## Risks

1. The workspace fix still assumes all replicas can see the same underlying workspace content; if there is no shared filesystem, packaging remains impossible across replicas.
2. Streaming npm publish requires careful JSON framing around base64 output; tests must validate payload correctness.
3. Query-layer authorization must preserve existing wildcard semantics exactly.
