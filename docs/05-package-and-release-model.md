# Package and release model

## Identity

A hosted skill is identified by a scoped npm-compatible package name such as `@company.skill/example`. A release is the pair of that package name and an exact semver version.

The CLI authoring and import workflows require a package name in the scoped `@<company>.skill/<skill-name>` form and an exact semver version. Source installs can remain local even when they have no package metadata.

## Release truth

The server stores release metadata and decides whether a release is published, visible, and installable. The external npm-compatible artifact repository stores package metadata and bundle bytes. A stored bundle without a server release record is not a published release.

A release record includes or derives:

- package name and version;
- registry metadata and integrity;
- skill manifest digest and components;
- version status and install policy;
- publication time;
- source and provenance metadata when available.

## Invariants

- package identity is stable;
- a published version is immutable;
- release history is retained;
- visibility and lifecycle changes are explicit;
- governance changes are auditable;
- the latest resolvable release is selected by the server, never guessed by the CLI.

## Source metadata

Catalog responses may expose `sourceKind`, `sourceLocator`, `originRegistryUrl`, `originRepositoryUrl`, `originRevision`, `importMode`, `hosted`, `mirrored`, and `installable`. These fields explain provenance and hosting state; they do not create another identity model.

## Related workflows

- [Publish and import workflows](07-publish-and-import-workflows.md)
- [Install resolution and refresh](08-install-resolution-and-refresh.md)
- [Governance and visibility](09-governance-and-visibility.md)
