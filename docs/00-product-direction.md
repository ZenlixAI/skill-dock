# Product direction and scope

Skill Registry is a self-hosted control plane for reusable agent skills. It makes skills discoverable and governable without taking ownership of the npm-compatible repository that stores bundle bytes.

## Product boundaries

- **Catalog server** owns catalog truth, release records, install resolution, visibility, governance, audit, and server-side packaging.
- **`skills` CLI** owns the command surface, local and git source discovery, local materialization, lock files, and client-side npm package download for `import-npm`.
- **Artifact repository** owns npm metadata and immutable bundle storage.
- **Database** persists catalog, release, governance, audit, packaging workspace, and idempotency state.

The server is authoritative for hosted releases. The artifact repository is not a second catalog.

## Why this product exists

Agent skills are easy to share as folders but difficult to operate as a reliable ecosystem. The product provides:

- catalog discovery and search
- reproducible registry installs
- local and git source installs
- registry and source refresh workflows
- validation, packaging, publishing, and importing
- visibility and lifecycle governance
- auditability
- private npm-compatible repository integration

## Non-goals

This documentation does not define:

- web UI behavior
- a replacement artifact storage backend
- a social marketplace or ranking system
- per-agent runtime policy
- execution tracing for skill consumers
- server-side aggregation of arbitrary npm registries, git repositories, or local directories

## Core identity and release contract

A hosted skill is identified by a scoped npm-compatible package name. A release is identified by that package name and an exact semver version.

- package identity is stable;
- a published version is immutable;
- the server decides whether a release is published, visible, and installable;
- a bundle without an accepted release record is not a published release;
- governance changes are explicit and auditable;
- install resolution does not guess a fallback version.

## Current workflows

- `find` searches the server catalog and can opt into the configured `skills.sh` external search provider;
- `add` installs a registry package or discovers skills from a local path, GitHub shorthand, or git URL;
- `list` and `remove` operate on local installed state;
- `update` refreshes registry-installed skills;
- `sync` refreshes source-installed skills from their recorded source;
- `init` and `inspect` support local authoring;
- `publish`, `import-local`, and `import-git` package and publish through the server;
- `import-npm` downloads an npm package in the CLI and submits the extracted source through the server import flow.

## Automatic source import on add

A successful local or git source installation runs a best-effort server import by default. The CLI merges the optional global `${XDG_CONFIG_HOME:-~/.config}/skills/config.json` with the nearest project `skills.config.json` found from the current working directory upward. Project values override global values. Either file can set:

- `syncOnAdd` — set to `false` to disable the automatic import;
- `packageNamePrefix` — supplies a scoped package prefix when source metadata has no package name;
- `defaultVersion` — supplies an exact version when source metadata has no version;
- `registry` — selects the catalog server for the automatic import.

The automatic import does not change the local source lock entry or the meaning of `skills sync`. If it fails, the local installation remains installed and the CLI reports a warning.

## Install transport

The server may return a `proxy` or `direct` install target. This is only a tarball transport decision; both modes resolve the same server-owned release contract.
