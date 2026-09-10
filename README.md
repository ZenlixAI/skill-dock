# Skill Registry

[English](README.md) | [简体中文](README.zh-CN.md)

> A self-hosted control plane for discovering, packaging, installing, and governing reusable agent skills.

Skill Registry has two operational surfaces:

- the **catalog server**, which owns catalog truth, release records, visibility, governance, packaging, and install resolution;
- the **`skills` CLI**, which discovers, installs, refreshes, packages, imports, and publishes skills.

Bundles are stored in an external npm-compatible artifact repository. The repository is replaceable; a bundle is not a published release until the server has accepted its release record.

## Current workflows

| Workflow | Command | Authority |
| --- | --- | --- |
| Catalog discovery | `skills find` | Server catalog; optional `skills.sh` results |
| Installation | `skills add` | Server for registry packages; local/git source for source installs |
| Local inventory | `skills list`, `skills remove` | CLI lock files and installed files |
| Registry refresh | `skills update` | Server release resolution |
| Source refresh | `skills sync` | Recorded local or git source |
| Authoring | `skills init`, `skills inspect` | Local filesystem |
| Hosted publication | `skills publish` | Server packaging and release workflow |
| Source import | `skills import-local`, `skills import-git` | Server packaging and release workflow |
| npm import | `skills import-npm` | CLI downloads from npm, then server packages the archive |

When a local or git source is installed with `skills add`, the CLI also sends the installed source through the server's local import flow by default. Its optional configuration is loaded from `${XDG_CONFIG_HOME:-~/.config}/skills/config.json` and the nearest project `skills.config.json`; project values override global values. This best-effort import does not turn a source install into a registry install. Set `syncOnAdd` to `false` to keep the install local.

## Boundaries

- **Server** — authoritative catalog, release, install, visibility, governance, audit, and packaging decisions.
- **CLI** — command contract, source discovery, local materialization, lock files, and client-side npm import download.
- **Artifact repository** — npm metadata and immutable bundle bytes.
- **Database** — catalog, release, governance, audit, workspace, and idempotency state.

A skill is identified by a scoped package name and an exact version for a release. Package identity and release version are not silently rewritten.

## Install target modes

For registry-backed installs, the server returns either:

- `proxy`: fetch the tarball through the server's npm proxy;
- `direct`: fetch the upstream npm-compatible URL using credentials from npm configuration when required.

This is transport policy only. It does not create a second skill identity or release model.

## Local development

```bash
make dev
make dev-images
make down
```

`make dev` creates the local stack and initializes sample data. `make dev-images` reloads rebuilt service images without rerunning data initialization when the kind cluster already exists. The default stack contains the catalog server, PostgreSQL, and an external Artifact Keeper repository.

## Documentation

- [Product direction and scope](docs/00-product-direction.md)
- [Server architecture and configuration](docs/01-server-architecture.md)
- [CLI contract and command model](docs/02-cli-contract.md)
- [Deployment and local runtime](docs/03-deployment.md)
- [Auth and access](docs/04-auth-and-access.md)
- [Package and release model](docs/05-package-and-release-model.md)
- [Artifact resolution](docs/06-artifact-resolution.md)
- [Publish and import workflows](docs/07-publish-and-import-workflows.md)
- [Install resolution and refresh](docs/08-install-resolution-and-refresh.md)
- [Governance and visibility](docs/09-governance-and-visibility.md)
- [External sources and discovery](docs/10-external-sources-and-search.md)
- [External-source implementation status](docs/11-cli-and-api-extension-plan.md)
- [CLI command reference](docs/12-cli-command-spec.md)
- [API endpoint reference](docs/13-api-endpoint-spec.md)
- [CLI package README](apps/skills-cli/README.md)

## Configuration quick look

### Server

| Variable | Purpose |
| --- | --- |
| `SKILL_REGISTRY_ADDR` | HTTP listen address |
| `SKILL_REGISTRY_DATABASE_URL` | SQLite or PostgreSQL URL |
| `SKILL_REGISTRY_SHARED_WORKSPACE_ROOT` | Writable packaging workspace root |
| `SKILL_REGISTRY_NPM_REGISTRY_URL` | Private npm-compatible artifact repository |
| `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` | Artifact URL exposed to clients |
| `SKILL_REGISTRY_READER_TOKEN` | Read token |
| `SKILL_REGISTRY_AUTHORING_TOKEN` | Packaging/import token |
| `SKILL_REGISTRY_PUBLISHER_TOKEN` | Publish and governance token |
| `SKILL_REGISTRY_INSTALL_TARGET_MODE` | `proxy` or `direct` |
| `SKILL_REGISTRY_SKILLS_SH_URL` | Optional `skills.sh` search endpoint; defaults to `https://skills.sh` |

See [Server architecture and configuration](docs/01-server-architecture.md) for the complete configuration surface and precedence rules.

### CLI

| Variable | Purpose |
| --- | --- |
| `SKILLS_REGISTRY_URL` | Catalog server base URL |
| `SKILLS_REGISTRY_TOKEN` | Bearer token for catalog requests |
| `SKILLS_REGISTRY_ROLE` | Optional role header for local role-based auth |
| `NPM_CONFIG_USERCONFIG` | npm credentials file used for direct npm requests |

## License

MIT
