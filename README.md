# Skill Registry

[English](README.md) | [简体中文](README.zh-CN.md)

> A self-hosted control plane for discovering, publishing, installing, and governing reusable agent skills.

Skill Registry has two operational surfaces:

- the **catalog server**, which owns catalog data, hosted releases, install resolution, packaging, visibility, governance, and audit records;
- the **`skills` CLI**, which discovers sources, installs skills locally, refreshes local state, and invokes catalog workflows.

Bundles live in an external npm-compatible artifact repository. A bundle becomes a published release only when the catalog server accepts its release record.

## Workflows

| Workflow | Command | Authority |
| --- | --- | --- |
| Catalog discovery | `skills find` | Catalog server; optional `skills.sh` results |
| Installation | `skills add` | Server for registry packages; CLI for local and git sources |
| Local inventory | `skills list`, `skills remove` | Local files and lock files |
| Registry refresh | `skills update` | Server release resolution |
| Source refresh | `skills sync` | Recorded local or git source |
| Authoring | `skills init`, `skills inspect` | Local filesystem |
| Publication | `skills publish`, `skills import-*` | Server packaging and release workflow |

After installing a local or git source, the CLI attempts a server import by default. This best-effort operation does not change the installation into a registry install. Set `syncOnAdd` to `false` in the merged global or project `skills.config.json` to disable it.

## Docker Compose

For local development, use the root Compose file. It starts Catalog, PostgreSQL, and a local npm-compatible artifact repository; it does not start the static website:

```bash
cp .env.example .env
# Edit .env and replace every sample password and token.
docker compose up --build -d
```

The Catalog API is available at <http://localhost:8080/api/v1>; its npm proxy is available at <http://localhost:8080/repository/skills-npm/>. Verify the service and an authenticated catalog read with:

```bash
curl -fsS http://localhost:8080/readyz
curl -fsS http://localhost:8080/api/v1/skills \
  -H "Authorization: Bearer <reader-token>"
```

Stop it with `docker compose down`; add `-v` only when you also want to remove local database and artifact data.

For a production deployment, use the [production deployment guide](docs/03-deployment.md). It runs only Catalog and PostgreSQL; connect Catalog to an external npm-compatible artifact repository and put a TLS reverse proxy in front of it.

## Build and run Catalog with Dockerfile

Build and run the standalone Catalog image with SQLite for local evaluation. This command does not configure an external artifact repository, so it is not a complete registry-install or publication deployment. Configure PostgreSQL, an npm-compatible artifact repository, and the public install URL before using it for production.

```bash
docker build -f zarf/docker/local/Dockerfile -t skill-registry-catalog .
docker run --rm -p 8080:8080 -v skill-registry-data:/var/lib/skill-registry \
  -e SKILL_REGISTRY_READER_TOKEN=reader-secret \
  -e SKILL_REGISTRY_AUTHORING_TOKEN=authoring-secret \
  -e SKILL_REGISTRY_PUBLISHER_TOKEN=publisher-secret \
  skill-registry-catalog
```

Use real secrets and configure PostgreSQL plus an npm-compatible artifact repository before production. See [deployment](docs/03-deployment.md) and [server configuration](docs/01-server-architecture.md).


## Documentation

- [Documentation index](docs/README.md)
- [CLI package README](apps/skills-cli/README.md)

## Configuration

### Server

| Variable | Purpose |
| --- | --- |
| `SKILL_REGISTRY_DATABASE_URL` | SQLite or PostgreSQL URL |
| `SKILL_REGISTRY_SHARED_WORKSPACE_ROOT` | Writable packaging workspace root |
| `SKILL_REGISTRY_ARTIFACT_READ_URL` | Artifact repository URL for server reads |
| `SKILL_REGISTRY_PUBLIC_INSTALL_URL` | Artifact URL exposed to clients |
| `SKILL_REGISTRY_READER_TOKEN` | Reader token |
| `SKILL_REGISTRY_AUTHORING_TOKEN` | Authoring token |
| `SKILL_REGISTRY_PUBLISHER_TOKEN` | Publisher token |
| `SKILL_REGISTRY_INSTALL_TARGET_MODE` | `proxy` or `direct` |

### CLI

| Variable | Purpose |
| --- | --- |
| `SKILLS_REGISTRY_URL` | Catalog server base URL |
| `SKILLS_REGISTRY_TOKEN` | Bearer token for catalog requests |
| `SKILLS_REGISTRY_ROLE` | Development role header |
| `NPM_CONFIG_USERCONFIG` | npm credentials used for direct npm traffic |

## License

MIT
