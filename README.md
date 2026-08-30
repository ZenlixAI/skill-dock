# Skill Registry

[English](README.md) | [简体中文](README.zh-CN.md)

> An open-source registry and distribution platform for reusable agent skills.

Skill Registry helps teams publish, discover, install, and govern reusable capabilities for coding agents. It combines a catalog API, a web interface, a command-line client, and an npm-compatible artifact repository into one self-hostable platform.

An **agent skill** is a reusable capability packaged around a `SKILL.md` file and its supporting resources. Skill Registry gives each published skill a stable package name, versioned releases, searchable metadata, and a controlled installation path.

## Why Skill Registry?

Agent skills are easy to share as folders and difficult to operate as a reliable ecosystem. Skill Registry provides the missing operational layer:

- **Discoverability** — search and browse skills through the catalog or web UI.
- **Reproducible installation** — resolve an exact package and version, then install it through the `skills` CLI.
- **Authoring and publishing** — validate, package, and publish local or imported skills.
- **Lifecycle governance** — deprecate, yank, or retire skills and versions without silently deleting history.
- **Self-hosting** — use a deployment's own database, authentication, and npm-compatible artifact repository.
- **Agent compatibility** — install skills into the directories used by supported coding agents.

## Project Status

Skill Registry is under active development. The core workflows are implemented, while deployment contracts, APIs, and user-facing features may continue to evolve. Treat the current API and package format as versioned project interfaces, not as a promise of backwards compatibility for every pre-release build.

## Architecture

The project deliberately separates product truth from artifact storage:

| Component | Responsibility |
| --- | --- |
| **Catalog API** | Owns discovery, metadata, version visibility, install resolution, and governance state. |
| **Skills CLI** | Installs, searches, lists, updates, synchronizes, validates, imports, and publishes skills. |
| **Web UI** | Provides catalog browsing and authoring/import workflows. |
| **Artifact repository** | Stores immutable skill bundles in an npm-compatible repository such as Nexus or Artifact Keeper. |
| **Database** | Persists catalog state, release records, governance changes, audit data, and workflow state. |

The catalog is the source of truth for what users can discover and install. The npm-compatible repository stores the bundle; an artifact existing in that repository is not considered a published release until it has a corresponding catalog release record.

## Design Principles

- **One public identity:** a skill is addressed by its npm-compatible **Package Name**.
- **Immutable releases:** each published version is a distinct release that can be resolved and verified.
- **Explicit governance:** lifecycle changes record an action and a reason.
- **Separated concerns:** discovery and policy belong to the catalog; bundle storage belongs to the artifact repository.
- **Portable source intake:** skills can be authored locally or imported from a local archive or Git source.
- **Operationally observable:** authentication, readiness endpoints, idempotent publishing, and audit records are part of the platform boundary.

## Quick Start

### Prerequisites

For the local development stack, install:

- Docker
- [kind](https://kind.sigs.k8s.io/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- Go 1.26 or later
- Node.js 18 or later and npm
- [uv](https://docs.astral.sh/uv/)

### Start the local stack

The supported local path runs the web UI, Catalog, and PostgreSQL in kind, while npm access is pointed at an Artifact Keeper repository running separately on `http://localhost:30080`:

```bash
make dev
make down
```

By default, Catalog reads from `http://host.docker.internal:30080/`; CLI/browser installs use the Catalog proxy at `http://127.0.0.1:18080/repository/skills-npm/`.

To override those URLs for a different Artifact Keeper host:

```bash
ARTIFACTKEEPER_INTERNAL_NPM_URL=http://host.docker.internal:30080/ \
ARTIFACTKEEPER_PUBLIC_NPM_URL=http://127.0.0.1:18080/repository/skills-npm/ \
make dev
```

The default local endpoints are:

| Service | URL |
| --- | --- |
| Web UI | <http://127.0.0.1:8080> |
| Catalog API | <http://127.0.0.1:18080> |
| npm proxy (via Catalog) | <http://127.0.0.1:18080/repository/skills-npm/> |

### Use the CLI against the local stack

Set the Catalog and npm authentication contexts separately:

```bash
export SKILLS_REGISTRY_URL=http://127.0.0.1:18080
export SKILLS_REGISTRY_TOKEN=reader-secret

skills find pdf
skills add @company.skill/your-skill -y
```

Publishing requires a publisher-capable Catalog token:

```bash
export SKILLS_REGISTRY_TOKEN=publisher-secret
skills publish /absolute/path/to/your-skill
```

The CLI also supports importing local and Git sources, updating registry-installed skills, and synchronizing source-installed skills. Run `skills --help` for the complete command surface.

## Common Development Commands

```bash
make help
make test-go
make test-web
make test-cli
make typecheck-cli
make build-web
make dev
make down
```

The root `Makefile` owns the top-level environment workflow and verification entry points. Interactive skill operations remain commands of the standalone `skills` CLI.

## Documentation

- [Local kind and Artifact Keeper E2E runbook](docs/local-nexus-e2e.zh-CN.md) — Chinese
- [Artifact Keeper npm integration](docs/artifact-keeper-npm.zh-CN.md) — Chinese
- [Skills CLI npx execution and npm auth guide](docs/skills-cli-npx-auth.zh-CN.md) — Chinese
- [Batch skill import](docs/batch-import-skills.zh-CN.md) — Chinese
- [Local Web and Catalog E2E](docs/local-web-catalog-e2e.zh-CN.md) — Chinese
- [Architecture decision records](docs/adr/)
- [Catalog API contract](docs/superpowers/specs/2026-05-28-agent-skill-registry-catalog-api.openapi.yaml)
- [Product introduction deck](docs/skill-registry-product-intro/)

The root README is the default English introduction. Operational runbooks are currently maintained in Chinese; contributions that add English counterparts are welcome.

## Configuration

A deployment commonly configures the following environment variables:

| Variable | Purpose |
| --- | --- |
| `SKILL_REGISTRY_ADDR` | Catalog listen address. |
| `SKILL_REGISTRY_DATABASE_URL` | SQLite or PostgreSQL database URL. |
| `SKILL_REGISTRY_NPM_REGISTRY_URL` | Artifact repository used by the Catalog for metadata and bundle access. |
| `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` | Artifact repository URL reachable by CLI and browser clients. |
| `SKILL_REGISTRY_READER_TOKEN` | Token for catalog read operations. |
| `SKILL_REGISTRY_AUTHORING_TOKEN` | Token for authoring and packaging workflows. |
| `SKILL_REGISTRY_PUBLISHER_TOKEN` | Token for publishing and governance operations. |
| `SKILL_REGISTRY_NPM_READ_AUTHORIZATION` | Authorization header for private artifact-repository reads. |
| `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION` | Authorization header used for server-side publication. |

See the local runbooks for complete examples, private registry authentication, and deployment-specific settings.

## Contributing

Contributions are welcome. Before submitting a change:

1. Read the relevant architecture decision records and package contracts.
2. Keep the Catalog, CLI, Web UI, and artifact-repository boundaries explicit.
3. Add or update tests for behavioral changes.
4. Run `make verify` and include any environment limitations in the pull request description.
5. Keep documentation and examples in sync with the command or API being changed.

For changes to the public package model, install resolution, publishing workflow, or governance semantics, document the decision in `docs/adr/` or the relevant design document.

## License

This project is licensed under the [MIT License](LICENSE).
