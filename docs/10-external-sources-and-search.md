# External sources, migration, and search

## Purpose

Skill Registry should be useful before it has a large native catalog.

This document defines how the registry can onboard existing skills from external npm registries, git sources, and local directories, while keeping the server authoritative and preserving source provenance.

## Goals

- Let users discover skills even when the registry starts with little or no native content.
- Import existing skills without forcing a full rewrite of their source or package identity.
- Support search across the local catalog and user-configured external registries.
- Respect the user's npm registry configuration for direct installs and registry-backed discovery.
- Preserve provenance so a mirrored skill can later be promoted to a hosted release.

## Non-goals

- Replacing npm, git, or local source workflows with a new package ecosystem.
- Introducing a marketplace or social ranking layer.
- Defining UI behavior for search results, filters, or import dialogs.
- Changing the package identity model defined in [Package and release model](05-package-and-release-model.md).

## Related docs

- [Package and release model](05-package-and-release-model.md)
- [Publish and import workflows](07-publish-and-import-workflows.md)
- [Install resolution and refresh](08-install-resolution-and-refresh.md)
- [Artifact resolution](06-artifact-resolution.md)
- [Server architecture and configuration](01-server-architecture.md)
- [CLI contract and command model](02-cli-contract.md)
- [CLI and API extension plan for external sources](11-cli-and-api-extension-plan.md)
- [CLI command and parameter specification](12-cli-command-spec.md)
- [API endpoint specification for external sources](13-api-endpoint-spec.md)

## Core idea

The registry should treat source origin and catalog truth as separate concerns.

- The **source** tells the system where a skill came from.
- The **release record** tells the system what is published, visible, and installable.
- The **artifact repository** stores immutable bundles when the registry hosts them.

A skill can be discoverable before it is hosted, and it can be hosted after it has been discovered from somewhere else.

## Source kinds

The registry should recognize three primary source kinds:

### npm registry source

A skill may already exist as a package in an npm-compatible registry.

This source kind is identified by:

- package name
- version or version range
- registry URL
- optional scope mapping
- optional authorization headers or npm config

### git source

A skill may live in a git repository and be discoverable from a branch, tag, or commit.

This source kind is identified by:

- repository URL
- ref or commit
- optional skill selector
- optional subdirectory path

### local source

A skill may begin as a local directory.

This source kind is identified by:

- filesystem path
- package metadata in the source tree
- optional injected package name and version during import

## Import and migration model

### 1. Mirror first

When an external skill is imported, the registry may begin by storing a mirrored release record.

A mirrored release record should keep:

- package name
- version
- source kind
- source locator
- source registry URL or git commit
- bundle digest or tarball reference when available
- governance state
- visibility state

Mirror mode is useful when the goal is to make an existing ecosystem searchable and installable with minimal disruption.

### 2. Promote later

A mirrored release can later be promoted to a hosted release.

Promotion means the registry becomes the authoritative owner of the bundle storage for that release, while preserving the original source provenance in audit history.

Promotion should not change the package name or version.

### 3. Preserve identity

The same package version should not silently fork into multiple releases.

If a release already exists, a later import should either:

- resolve to the same release record when the bundle and identity match, or
- fail with an explicit conflict when the bundle or provenance differs

The registry should not guess which upstream is canonical.

## Search model

Search should start with the registry catalog, then optionally expand to user-configured external sources.

### Search sources

The search layer may include:

- the local registry catalog
- one or more npm registries configured by the user or deployment
- mirrored external releases already indexed by the server

### Search result requirements

Every search result should carry enough metadata to explain what it is and where it came from:

- package name
- version, if known
- source kind
- origin registry or repository
- visibility
- installability
- whether the result is local, mirrored, or hosted

### Ranking rules

Keep ranking simple and predictable:

1. exact package-name matches
2. scope matches
3. text matches in metadata or description
4. recency or version relevance

Search should not hide provenance behind ranking.

### Filtering rules

Search should allow filtering by:

- source kind
- scope
- visibility
- installability
- origin registry

## User-configured npm registry

Users should be able to use their own npm registry configuration without changing the registry contract.

### Direct installs

When the server returns `direct` install targets, the CLI may use the user's npm configuration to fetch tarballs.

The CLI should honor:

- `NPM_CONFIG_USERCONFIG`
- a local `.npmrc`
- scope-specific registry mappings
- npm authorization entries already present in the user's config

### Search and discovery

The same npm configuration may also be used to discover packages from additional registries.

This means a user can keep their personal or organization registry preferences in standard npm config instead of learning a separate registry syntax.

### Configuration principle

Do not invent a second registry configuration model unless it solves a problem npm config cannot solve.

If a deployment needs custom defaults, they should layer on top of npm config rather than replacing it.

## Proposed workflow shape

### Discover

The user searches the catalog.

If the requested skill is not yet hosted locally, the registry may still show an external match with provenance attached.

### Import

The user imports the selected source.

The server validates the source, records provenance, and decides whether the result is mirrored or hosted.

### Install

The CLI installs from the release record returned by the server.

Install resolution still follows the existing `proxy` / `direct` rules from [Install resolution and refresh](08-install-resolution-and-refresh.md).

### Refresh

- registry-installed releases continue to use `update`
- source-installed releases continue to use `sync`
- mirrored releases should not be refreshed by silently guessing a new upstream source

## Operational implications

- A newly open-sourced registry does not need to start with a large native catalog.
- External skill ecosystems become searchable without sacrificing governance.
- Users can bring their own registry settings and still use the same CLI and server contract.
- The system can migrate gradually from external provenance to hosted ownership.

## Design rule

Treat external sources as onboarding inputs, not as a separate registry universe.

The server remains authoritative for release records, visibility, and installability, while external registries and git sources remain discoverable inputs that can be promoted into the catalog.
