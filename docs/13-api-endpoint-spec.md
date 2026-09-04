# API endpoint specification for external sources

## Purpose

This document defines the API-level changes needed to support external skill sources, provenance-aware search, and gradual migration from external registries into the hosted catalog.

## Scope

This document covers:

- read endpoints used by the CLI and web app
- write endpoints used by publish and import workflows
- request and response fields needed for provenance
- search and discovery filters for external sources

This document does not cover:

- database schema
- storage internals
- UI behavior
- CLI flag syntax

## Design goals

- Keep the existing API surface stable where possible.
- Prefer additive fields over breaking route changes.
- Keep the server authoritative for hosted catalog truth.
- Let external sources participate in discovery without forcing a full migration first.
- Make provenance visible in every response that can influence install or import decisions.

## Relationship to other docs

- [External sources, migration, and search](10-external-sources-and-search.md)
- [CLI and API extension plan for external sources](11-cli-and-api-extension-plan.md)
- [CLI command and parameter specification](12-cli-command-spec.md)
- [Package and release model](05-package-and-release-model.md)
- [Publish and import workflows](07-publish-and-import-workflows.md)
- [Install resolution and refresh](08-install-resolution-and-refresh.md)
- [Server architecture and configuration](01-server-architecture.md)

## Current API shape

The current server already exposes these read and write routes:

- `GET /api/v1/skills`
- `GET /api/v1/skills/search`
- `GET /api/v1/skills/curated`
- `GET /api/v1/skills/:package`
- `GET /api/v1/skills/:package/versions`
- `GET /api/v1/skills/:package/versions/:version`
- `GET /api/v1/skills/:package/install-target`
- `POST /api/v1/skill-packaging/import-review`
- `POST /api/v1/skill-packaging/publish`
- `PUT /api/v1/skills/:package/versions/:version/release-record`
- `POST /api/v1/skills/:package/governance-changes`
- `POST /api/v1/skills/:package/versions/:version/governance-changes`

The external-source work should extend this surface primarily through additive parameters and response fields.

## Read API changes

### 1. Search and catalog listing

`GET /api/v1/skills` and `GET /api/v1/skills/search` should accept external-source filters.

#### Proposed query parameters

- `q` — free-text query, unchanged
- `source_type` — existing catalog source filter, unchanged
- `include_retired` — existing flag, unchanged
- `include_external` — when true, merge external-source matches into the result set
- `source` — one of `catalog`, `npm`, `git`, `local`, or `all`
- `registry` — npm-compatible registry URL to search for this request
- `scope` — package scope filter for npm-style sources
- `origin` — upstream registry or repository filter
- `cursor` — pagination cursor, unchanged
- `pageSize` — page size, unchanged

#### Behavior

- `catalog` remains the default search domain.
- `include_external` is a convenience flag for broad discovery.
- `source` takes precedence over `include_external` when both are present.
- `registry` is request-scoped and does not persist.
- `origin` should work for both npm and git sources.
- The server should not hide provenance when returning external matches.

#### Response changes

Search and list responses should include provenance metadata for each item.

Proposed additive fields on `SkillSummary` and related list items:

- `sourceKind`
- `sourceLocator`
- `originRegistryUrl`
- `originRepositoryUrl`
- `importMode`
- `hosted`
- `mirrored`
- `installable`

These fields are informational and should not alter package identity.

### 2. Skill detail and version detail

`GET /api/v1/skills/:package`, `GET /api/v1/skills/:package/versions`, and `GET /api/v1/skills/:package/versions/:version` should expose the same provenance fields as the search response.

For version detail, the response should also show:

- whether the version was mirrored or hosted
- the origin registry or repository that supplied the release
- whether the version can be installed directly

### 3. Install target

`GET /api/v1/skills/:package/install-target` should keep the current `proxy` / `direct` install contract.

Additive response fields should explain provenance when relevant:

- `sourceKind`
- `sourceLocator`
- `originRegistryUrl`
- `originRepositoryUrl`
- `importMode`

The install target must continue to be deterministic and must not guess a fallback release.

## Write API changes

### 1. Import review

`POST /api/v1/skill-packaging/import-review` should accept external npm sources in addition to local and git sources.

#### Request shape

The request should keep a normalized `source` object and extend it to support:

- `kind: npm`
- `kind: git`
- `kind: local`
- `kind: archive` when applicable

The source object should be able to express:

- npm registry URL
- package name
- version or version range
- git repository URL
- git ref or commit
- local filesystem path
- optional subdirectory or selector

#### Response shape

The response should include:

- package name
- version
- warnings and errors
- a manifest digest when available
- provenance fields
- import mode, when the source can be mirrored or hosted

### 2. Publish

`POST /api/v1/skill-packaging/publish` should continue to publish hosted releases through the server.

The publish request should accept the same normalized source envelope as import review.

#### Required behavior

- The server validates the source and decides whether the result is publishable.
- The server records provenance on the resulting release record.
- The publish response should indicate whether the release was newly created, replayed, mirrored, or hosted.

### 3. Release record creation

`PUT /api/v1/skills/:package/versions/:version/release-record` should remain the server-owned entry point for creating release truth.

The request body should be able to carry provenance fields from the packaging flow, including:

- source kind
- source locator
- origin registry URL or repository URL
- import mode
- bundle digest or manifest digest

The endpoint should still reject requests that try to silently rewrite package identity.

## Governance API compatibility

The governance endpoints do not need new routes for external sources.

However, their payloads and returned state should remain compatible with provenance-aware releases.

### Existing routes

- `POST /api/v1/skills/:package/governance-changes`
- `POST /api/v1/skills/:package/versions/:version/governance-changes`

### Expected compatibility

- governance actions should still apply to the same package and version identity
- provenance fields should remain visible in surrounding read APIs
- governance records should continue to explain state transitions for mirrored or hosted releases

## Suggested response model additions

The following additive fields are useful across multiple endpoints:

- `sourceKind`
- `sourceLocator`
- `originRegistryUrl`
- `originRepositoryUrl`
- `originRevision`
- `importMode`
- `hosted`
- `mirrored`
- `installable`
- `registryAccessMode`

The server may choose different combinations for different endpoints, but the same terms should be reused consistently.

## Compatibility rule

Prefer extension over replacement.

If an endpoint can support external sources by adding fields or query parameters, keep the existing route and behavior intact rather than introducing a parallel endpoint that duplicates the same contract.

## Recommended implementation order

1. Add provenance fields to list, detail, version, and install-target responses.
2. Extend `GET /api/v1/skills` and `GET /api/v1/skills/search` with external-source filters.
3. Normalize the import and publish request envelopes to accept npm sources.
4. Record provenance in release record creation.
5. Keep governance endpoints compatible without adding new routes.
