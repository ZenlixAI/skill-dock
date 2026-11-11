# Production deployment

This guide deploys the Catalog server with Docker Compose. It is intended for a single host or a small internal installation. The production stack does not deploy the static website. Put a TLS reverse proxy and access-control layer in front of Catalog before exposing it to the public internet.

## What is deployed

The production Compose stack contains:

- `catalog`: the Skill Dock API, packaging workspace, and npm proxy;
- `postgres`: persistent catalog, release, governance, audit, and idempotency state.

The website is maintained in this repository but is not part of the production Compose stack. Host the static site separately if you need it. The artifact repository is also external; use an existing npm-compatible repository such as Verdaccio, Nexus, or Artifactory.

## Prerequisites

- Docker Engine 24 or later
- Docker Compose v2 (`docker compose`)
- Pull access to the base images used by `zarf/docker/Dockerfile` (or a locally retagged equivalent)
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

# Artifact repository URL
SKILL_REGISTRY_ARTIFACT_READ_URL=https://artifacts.example.com/npm/skills/

# Keep these server-side credentials private. Never expose them to browsers or CLI clients.
SKILL_REGISTRY_ARTIFACT_READ_AUTHORIZATION=Bearer <artifact-read-token>
SKILL_REGISTRY_ARTIFACT_PUBLISH_AUTHORIZATION=Bearer <artifact-publish-token>

# The public Catalog URL returned in CLI install targets.
SKILL_REGISTRY_PUBLIC_INSTALL_URL=https://registry.example.com/repository/skills-npm/
```

When a password contains reserved URL characters, percent-encode it in `SKILL_REGISTRY_DATABASE_URL`. The Docker network resolves the database host as `postgres`.

The Compose definition binds Catalog to `127.0.0.1:8081` by default (`CATALOG_PORT` can override this). Put a TLS reverse proxy or another public gateway in front of Catalog when it must be reachable by CLI clients.

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

curl -fsS http://127.0.0.1:8081/healthz
curl -fsS http://127.0.0.1:8081/readyz
```

`/healthz` should return `ok`. `/readyz` verifies the catalog repository, audit store, and idempotency store. If readiness fails, inspect logs before retrying:

```bash
docker compose --env-file zarf/compose/.env \
  -f zarf/compose/production.yml logs --tail=100 catalog postgres
```

## 3. Configure the external reverse proxy

Point the public hostname, for example `registry.example.com`, to the Catalog service at `http://127.0.0.1:8081` by default. If you set `CATALOG_PORT`, use that host port instead. Catalog serves both `/api/v1` and `/repository/skills-npm/` directly.

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

Install the `skills` CLI from this repository or from the npm registry that hosts your catalog copy. The commands below assume `skills` is on `PATH`.

Discover and install with a reader token:

```bash
export SKILLS_REGISTRY_URL=https://registry.example.com
export SKILLS_REGISTRY_TOKEN=<reader-token>

skills find
skills add @scope/skill-name
```

Use the token appropriate to the operation for packaging or publication:

```bash
export SKILLS_REGISTRY_TOKEN=<publisher-token>
skills publish ./my-skill
```

The catalog server remains the authority for publication. Do not publish bundles directly to the artifact repository.

## Backup, updates, and shutdown

The Compose stack creates two persistent named volumes:

- `postgres_data`: authoritative PostgreSQL state;
- `catalog_workspace`: server-side packaging workspace. It is operational storage, not a substitute for artifact repository retention.

Back up PostgreSQL through the normal database-backup process and test restores. Back up the artifact repository independently.

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

## Website deployment

The public site lives in `apps/site` and deploys as a Cloudflare Worker with static assets. The Worker is defined in `apps/site/worker/index.ts`; `apps/site/wrangler.jsonc` points it at the Vite `dist` directory.

The `/registry` page is a public skills.sh directory. The Worker:

- serves the SPA and static assets through the `ASSETS` binding;
- reverse-proxies `/api/skills-sh/*` to `SKILLS_SH_BASE_URL` (default `https://skills.sh`);
- parses and sanitizes the skills.sh detail page and RSC payload on the edge;
- caches search/browse responses for 30 seconds and detail responses for 10 minutes;
- reports `SKILLS_SH_ENABLED` through `/api/skills-sh/capabilities`.

Deploy manually:

```bash
cd apps/site
npm ci
npx playwright install chromium
npm test
npm run deploy
```

`npm run deploy` runs the SEO build and then `wrangler deploy` using `wrangler.jsonc`. Set `SKILLS_SH_ENABLED=false` in `wrangler.jsonc`, the dashboard, or a deployment override to remove the directory data and render the disabled state.

Local development:

```bash
cd apps/site
npm run build          # build dist once
npm run dev:worker     # wrangler dev, serves the Worker and dist
```

Or run Vite for HMR and point its `/api/skills-sh` proxy at a running Worker:

```bash
VITE_SKILLS_SH_PROXY=http://127.0.0.1:8787 npm run dev
```

The self-hosted Docker site still works through the Go catalog: its nginx config proxies `/api/` to the catalog, and the catalog exposes the same `/api/skills-sh/capabilities`, `/search`, `/browse`, and `/skill` route names.

The generated canonical, Open Graph, robots, and sitemap URLs still use `https://skill-dock.pages.dev`; update `apps/site/index.html`, `apps/site/src/lib/content.ts`, and `apps/site/public/sitemap.xml` (or attach a custom domain) when the Worker gets a different hostname.

The SEO build renders the landing page to `dist/index.html`, the directory shell to `dist/registry.html`, and the legal pages to `dist/privacy.html` and `dist/terms.html`. `wrangler.jsonc` uses `not_found_handling: single-page-application`, so direct navigation to `/registry` works even when the prerendered file is absent.

### GitHub Actions

Two workflows run from the repository root:

- `.github/workflows/site.yml` runs the site tests, builds the static site, and deploys the Cloudflare Worker when `main` or `master` receives a push.
- `.github/workflows/catalog-image.yml` builds the catalog image from `zarf/docker/ci/Dockerfile` and publishes it to GHCR for `main`, `master`, and `v*` tags.

The site workflow needs these repository secrets:

| Secret | Value |
| --- | --- |
| `CLOUDFLARE_ACCOUNT_ID` | The Cloudflare account that owns the Worker |
| `CLOUDFLARE_API_TOKEN` | A token with `Account > Workers Scripts > Edit` |

The image workflow uses the repository `GITHUB_TOKEN` and needs no extra secret. The published image is `ghcr.io/<owner>/skill-dock-catalog`. The workflow lowercases the owner for the image name.

If `CLOUDFLARE_API_TOKEN` is not set, the site workflow builds and tests the site, then skips the deploy step with a warning. Add the secret to enable automatic deploys.
