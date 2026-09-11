# Product direction and scope

Skill Registry is a self-hosted control plane for reusable agent skills. It makes skills discoverable, publishable, installable, and governable while using an external npm-compatible repository for bundle storage.

## Responsibilities

- **Catalog server** — catalog data, release records, install resolution, packaging, visibility, governance, and audit records.
- **`skills` CLI** — source discovery, local materialization, local lock files, and command-line workflows.
- **Artifact repository** — npm metadata and immutable bundle bytes.
- **Database** — catalog, release, governance, audit, packaging-workspace, and idempotency state.

The server is authoritative for hosted releases. A bundle is not a published release until the server accepts its release record.

## Current workflows

- Find catalog skills, with optional results from the configured `skills.sh` adapter.
- Install published registry packages or skills discovered from local and git sources.
- Refresh registry installs with `update` and source installs with `sync`.
- Create, inspect, package, publish, and import skills through the CLI and catalog service.
- Apply explicit skill and version lifecycle changes.

## Release contract

A hosted skill is identified by a scoped npm-compatible package name. A release is that package name and an exact semver version.

- Package identity is stable.
- Published versions are immutable.
- The server decides whether a release is visible and installable.
- Install resolution does not guess a fallback version.
- Governance changes are explicit and auditable.

## Source installs

`skills add` accepts a local path, GitHub shorthand, or git URL. It records the source locally, and `skills sync` replays that source later. Successful local and git installs also attempt a server import by default; a failed import does not undo the completed local installation. Set `syncOnAdd` to `false` in the merged global or project `skills.config.json` to disable this import.

## Install transport

The server returns either a `proxy` or `direct` install target. This only controls tarball retrieval; it does not create a separate identity or release model.
