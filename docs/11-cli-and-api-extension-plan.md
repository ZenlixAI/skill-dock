# CLI and API extension plan for external sources

## Purpose

This document proposes the smallest contract changes needed to make the registry useful before it has a large native catalog.

The goal is to support onboarding existing skills from npm registries, git sources, and local directories while keeping the server authoritative and preserving source provenance.

## Design constraints

- Keep the current command model stable where possible.
- Keep the server authoritative for hosted catalog state.
- Do not require users to migrate a skill before they can discover it.
- Use standard npm configuration before introducing any new registry configuration format.
- Make provenance visible in both CLI output and API responses.

## Relationship to other docs

- [External sources, migration, and search](10-external-sources-and-search.md)
- [CLI contract and command model](02-cli-contract.md)
- [CLI command and parameter specification](12-cli-command-spec.md)
- [API endpoint specification for external sources](13-api-endpoint-spec.md)
- [Server architecture and configuration](01-server-architecture.md)
- [Package and release model](05-package-and-release-model.md)
- [Publish and import workflows](07-publish-and-import-workflows.md)
- [Install resolution and refresh](08-install-resolution-and-refresh.md)

## CLI plan

### 1. Extend `find` to cover external sources

`skills find` should remain the default discovery entry point.

The default behavior should still search the registry catalog first. External sources should be opt-in through flags so existing users do not lose the current mental model.

Proposed additions:

- `--include-external` to merge external results into the output
- `--source <catalog|npm|git|local|all>` to narrow or widen the search domain
- `--registry <url>` to point discovery at a specific npm registry for the current invocation
- `--scope <scope>` to filter by package scope when searching npm-style sources
- `--origin <url>` to filter by a specific upstream registry or repository
- `--json` should include provenance fields, not just display labels

Search results should identify:

- package name
- version, if known
- source kind
- origin registry or repository
- whether the result is hosted, mirrored, or external-only
- whether the result is installable through the registry

### 2. Add an npm import command

The current import model covers local and git sources. It should gain a first-class npm source path.

Proposed command:

- `skills import-npm <package>`

Proposed flags:

- `--registry <url>`
- `--version <version>`
- `--mirror` to keep the imported release tied to the upstream source
- `--hosted` to promote the import into the registry's hosted release model
- `--json`

Semantics:

- `import-npm` resolves package metadata from an npm-compatible registry.
- The server validates the source and decides whether the result is mirrored or hosted.
- The imported release should preserve the original package name and version.
- The import should not silently rewrite package identity.

### 3. Keep `add` as the install command

`skills add` should continue to install registry-backed skills and source-discovered skills.

For npm-backed installs, the CLI should:

- resolve through the server when the package is already in the catalog
- fall back to user-configured npm registry settings only when the workflow explicitly targets an external source
- honor `NPM_CONFIG_USERCONFIG` and a local `.npmrc`
- respect scope-specific registry mappings from npm config

`--registry` should remain an invocation-level override, not a new persistent registry state.

### 4. Do not introduce a parallel registry config model in phase 1

The easiest user path is to reuse npm configuration for registry selection and credentials.

The CLI may later add a command for managing search sources, but that command should write npm-compatible configuration or deployment-local configuration rather than inventing a second registry syntax.

## API plan

### 1. Enrich release and search payloads with provenance

The server should expose enough metadata for the CLI to explain where a skill came from.

Proposed fields on release and search responses:

- `source_kind`
- `source_locator`
- `origin_registry_url`
- `origin_revision`
- `import_mode`
- `visibility`
- `installable`
- `governance_state`

These fields should be additive. They should not change package identity or version semantics.

### 2. Normalize import input

The server should accept a normalized import payload so local, git, and npm sources all follow the same contract shape.

A proposed import request should contain:

- a `source` object with `kind` and locator details
- an optional `mode` such as `mirror` or `hosted`
- optional source-specific hints such as package name, version, ref, or subdirectory

This keeps the server contract extensible without creating one endpoint per source type.

### 3. Extend search API filters

Search should support filtering by source and provenance.

Useful server-side filters include:

- source kind
- visibility
- installability
- package scope
- origin registry

If a deployment chooses to support server-side aggregation of external registries, that logic should live behind the search API, not in the CLI.

### 4. Preserve install resolution behavior

The install target model should remain unchanged.

- `proxy` still means the CLI fetches through the server-rewritten tarball URL.
- `direct` still means the CLI uses the upstream npm-compatible registry URL.
- npm credentials still matter only for direct fetches.

## Suggested rollout sequence

### Phase 1: payload enrichment

- Add provenance fields to search and release responses.
- Keep all current commands working.
- Make `find` display external provenance when available.

### Phase 2: npm import

- Add `skills import-npm`.
- Reuse the existing publish/import server path.
- Preserve package identity and source provenance.

### Phase 3: external discovery in CLI

- Let the CLI query user-configured npm registries for search.
- Merge external results with server catalog results.
- Keep the catalog result set authoritative when the same package/version exists in both places.

### Phase 4: optional server-managed external sources

- If the deployment wants shared external registries, add them as server-managed adapters.
- Keep user-specific npm config on the client side.

## Open decisions

- Should `find` merge external results by default, or only when `--include-external` is passed?
- Should `import-npm` default to `mirror` or require an explicit `--mirror` or `--hosted` flag?
- Should external source aggregation remain CLI-only, or should the server also support deployment-configured external sources?

## Recommended first implementation slice

- Add provenance fields to API responses.
- Add `skills import-npm`.
- Keep `skills find` catalog-first, with external sources opt-in.
- Reuse npm config for registry selection and auth.
- Avoid a new persistent registry config format until there is a concrete need.
