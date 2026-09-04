# Skill Registry

[English](README.md) | [简体中文](README.zh-CN.md)

> A self-hosted registry control plane for reusable agent skills.

Skill Registry documents the server and CLI that manage reusable skills across an organization. The system keeps the catalog authoritative, stores bundles in an external npm-compatible repository, and exposes a CLI for install, refresh, publish, and import workflows.

## What is in scope

This documentation set focuses on:

- the **skills server** as the source of truth for discovery, release records, visibility, and governance
- the **`skills` CLI** as the user-facing contract for install and publish workflows
- the **external artifact repository** as replaceable bundle storage
- deployment and configuration needed to run those two pieces together

UI details are intentionally omitted from this document set.

## Product shape

- **Catalog/server** — owns search, metadata, release visibility, and lifecycle state
- **CLI** — installs, refreshes, publishes, and imports skills
- **Artifact repository** — stores immutable bundles
- **Database** — persists catalog state, releases, and audit data

A skill is addressed by its package name and versioned release. The server decides what is published and installable; the artifact repository stores the bundle.

## Core workflows

- Discover skills with `skills find`
- Install skills with `skills add`
- Refresh registry installs with `skills update`
- Refresh source installs with `skills sync`
- Publish local skills with `skills publish`
- Import local or git sources with `skills import-local` and `skills import-git`
- Inspect and scaffold skills with `skills inspect` and `skills init`

## Install behavior

Registry-backed installs can be resolved in two access modes:

- `proxy` — the server rewrites tarball URLs to its own npm proxy mount
- `direct` — the server returns upstream npm-compatible registry URLs unchanged, so the CLI can use the user's own npm configuration and credentials

This is a transport choice for install resolution, not a separate skill identity model. When the server returns `direct`, the CLI may read npm auth from the user's npm config file (`NPM_CONFIG_USERCONFIG` or the local `.npmrc`) to download the tarball. It does not use npm config for `proxy` targets.

## Local development

The supported local path uses the repository's development scripts.

```bash
make dev
make dev-images
make down
```

Use `make dev` to create the local stack and initialize sample data. Use `make dev-images` when the kind cluster already exists and you only need to reload rebuilt service images without re-running the data initialization step.

The default local stack uses the catalog server, PostgreSQL, and an external Artifact Keeper repository.

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
- [External sources, migration, and search](10-external-sources-and-search.md)
- [CLI and API extension plan for external sources](11-cli-and-api-extension-plan.md)
- [CLI command and parameter specification](12-cli-command-spec.md)
- [API endpoint specification for external sources](13-api-endpoint-spec.md)
- [CLI reference](apps/skills-cli/README.md)

## Configuration quick look

### Server

| Variable | Purpose |
| --- | --- |
| `SKILL_REGISTRY_ADDR` | Server listen address |
| `SKILL_REGISTRY_DATABASE_URL` | Database URL |
| `SKILL_REGISTRY_NPM_REGISTRY_URL` | Artifact repository used by the server |
| `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` | Artifact repository URL visible to clients |
| `SKILL_REGISTRY_READER_TOKEN` | Read token |
| `SKILL_REGISTRY_AUTHORING_TOKEN` | Authoring token |
| `SKILL_REGISTRY_PUBLISHER_TOKEN` | Publisher token |
| `SKILL_REGISTRY_NPM_READ_AUTHORIZATION` | Authorization header for private registry reads |
| `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION` | Authorization header for server-side publication |
| `SKILL_REGISTRY_INSTALL_TARGET_MODE` | Install target mode returned by the server (`proxy` or `direct`) |

### CLI

| Variable | Purpose |
| --- | --- |
| `SKILLS_REGISTRY_URL` | Catalog base URL |
| `SKILLS_REGISTRY_TOKEN` | Bearer token for CLI requests |
| `SKILLS_REGISTRY_ROLE` | Optional role header for role-based local auth |

## License

MIT
