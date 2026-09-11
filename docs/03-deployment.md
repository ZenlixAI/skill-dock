# Production deployment

This guide deploys the Catalog server with Docker Compose. It is intended for a single host or a small internal installation. Put a TLS reverse proxy and access-control layer in front of Catalog before exposing it to the public internet.

## What is deployed

The production Compose stack contains:

- `catalog`: the Skill Registry API, packaging workspace, and npm proxy;
- `postgres`: persistent catalog, release, governance, audit, and idempotency state.

The website is not deployed by this project. Host the static site separately, or use the repository installation link. The artifact repository is also external; use an existing npm-compatible repository such as Verdaccio, Nexus, or Artifactory.

## Prerequisites

- Docker Engine 24 or later
- Docker Compose v2 (`docker compose`)
- An npm-compatible artifact repository reachable from the catalog container
- A DNS name and TLS termination for a public deployment

Build the supplied production images from the repository root. Do not use `make dev` for production; it is a kind-based development environment.

## 1. Create the deployment configuration

Create `zarf/compose/.env` with the following values. Do not commit the file. Generate each secret, and limit the file to the deployment operator (`chmod 600 zarf/compose/.env`).

```dotenv
# Database
POSTGRES_DB=skill_registry
POSTGRES_USER=skill_registry
POSTGRES_PASSWORD=<random-database-password>
SKILL_REGISTRY_DATABASE_URL=postgres://skill_registry:<url-encoded-database-password>@postgres:5432/skill_registry?sslmode=disable

# Catalog API roles
SKILL_REGISTRY_READER_TOKEN=<random-reader-token>
SKILL_REGISTRY_AUTHORING_TOKEN=<random-authoring-token>
SKILL_REGISTRY_PUBLISHER_TOKEN=<random-publisher-token>

# Artifact repository URL and server-side credentials
SKILL_REGISTRY_ARTIFACT_READ_URL=https://artifacts.example.com/npm/skills/
SKILL_REGISTRY_ARTIFACT_READ_AUTHORIZATION=Bearer <artifact-read-token>
SKILL_REGISTRY_ARTIFACT_PUBLISH_AUTHORIZATION=Bearer <artifact-publish-token>

# The public Catalog URL returned in CLI install targets.
SKILL_REGISTRY_PUBLIC_INSTALL_URL=https://registry.example.com/repository/skills-npm/

# Do not publish this value to browsers or CLI clients.
```

When a password contains reserved URL characters, percent-encode it in `SKILL_REGISTRY_DATABASE_URL`. The Docker network resolves the database host as `postgres`.

The Compose definition binds Catalog to `127.0.0.1:8081`. Put a TLS reverse proxy or another public gateway in front of Catalog when it must be reachable by CLI clients.

## 2. Build and start

Run from the repository root:

```bash
chmod 600 zarf/compose/.env

docker compose --env-file zarf/compose/.env \
  -f zarf/compose/production.yml build

docker compose --env-file zarf/compose/.env \
  -f zarf/compose/production.yml up -d
```

Verify container state and the health endpoints:

```bash
docker compose --env-file zarf/compose/.env \
  -f zarf/compose/production.yml ps

curl -fsS http://127.0.0.1:8080/healthz
curl -fsS http://127.0.0.1:8081/readyz
```

`/healthz` should return `ok`. `/readyz` verifies the catalog repository, audit store, and idempotency store. If readiness fails, inspect logs before retrying:

```bash
docker compose --env-file zarf/compose/.env \
  -f zarf/compose/production.yml logs --tail=100 catalog postgres
```

## 3. Configure the external reverse proxy

Point the public hostname, for example `registry.example.com`, to the Catalog service at `http://127.0.0.1:8081`. Catalog serves both `/api/v1` and `/repository/skills-npm/` directly.

The reverse proxy must:

- terminate TLS;
- forward all paths to Catalog;
- preserve `Host` and set `X-Forwarded-Proto` to the external scheme;
- permit request bodies and upstream timeouts suitable for package uploads;
- expose neither PostgreSQL nor the private artifact repository;
- protect or restrict write operations if publication is not open to all users.

Do not expose artifact repository credentials in the browser, CLI output, or public install target. After DNS and TLS are configured, verify the public endpoint:

```bash
curl -fsS https://registry.example.com/healthz
curl -fsS https://registry.example.com/api/v1/skills \
  -H "Authorization: Bearer <reader-token>"
```

## 4. Configure the CLI

Install or discover skills with a reader token:

```bash
export SKILLS_REGISTRY_URL=https://registry.example.com
export SKILLS_REGISTRY_TOKEN=<reader-token>

npx skills find
npx skills add @scope/skill-name
```

Use the token appropriate to the operation for packaging or publication:

```bash
export SKILLS_REGISTRY_TOKEN=<publisher-token>
npx skills publish ./my-skill
```

The catalog server remains the authority for publication. Do not publish bundles directly to the artifact repository.

## Backup, updates, and shutdown

The Compose stack creates two persistent named volumes:

- `postgres_data`: authoritative PostgreSQL state;
- `catalog_workspace`: the server-side packaging workspace.

Back up PostgreSQL through the normal database-backup process and test restores. The workspace is operational storage, not a replacement for artifact repository retention. Back up the artifact repository independently.

To update the application:

```bash
docker compose --env-file zarf/compose/.env \
  -f zarf/compose/production.yml build catalog

docker compose --env-file zarf/compose/.env \
  -f zarf/compose/production.yml up -d catalog

curl -fsS http://127.0.0.1:8081/readyz
```

To stop without deleting data:

```bash
docker compose --env-file zarf/compose/.env \
  -f zarf/compose/production.yml stop
```

To remove containers and the network while keeping volumes:

```bash
docker compose --env-file zarf/compose/.env \
  -f zarf/compose/production.yml down
```

Do not use `down --volumes` unless you intentionally want to delete the database and workspace.

## Configuration reference

- [Server architecture and configuration](01-server-architecture.md)
- [Auth and access](04-auth-and-access.md)
- [Artifact resolution](06-artifact-resolution.md)
- [CLI command reference](12-cli-command-spec.md)
- [`zarf/compose/production.yml`](../zarf/compose/production.yml)
