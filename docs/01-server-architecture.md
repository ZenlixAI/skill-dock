# Server architecture and configuration

The catalog server owns catalog reads, release records, install-target resolution, packaging, governance, and audit records. It stores catalog state in a database and uses an npm-compatible artifact repository for package metadata and bundle bytes.

## HTTP surfaces

Catalog routes are served under `/api/v1`:

- `GET /skills`, `/skills/search`, and `/skills/curated`
- `GET /skills/:package`, `/skills/:package/versions`, and `/skills/:package/versions/:version`
- `GET /skills/:package/install-target`
- `GET /governance-changes` and `/audit-events`
- `POST /skill-packaging/import-review` and `/skill-packaging/publish`
- `PUT /skills/:package/versions/:version/release-record`
- `POST /skills/:package/governance-changes` and `/skills/:package/versions/:version/governance-changes`

The process also exposes unauthenticated `GET /healthz`, `GET /readyz`, and `GET /metrics` endpoints. The API endpoint reference describes request and response contracts.

## Runtime configuration

| Variable | Purpose | Default |
| --- | --- | --- |
| `SKILL_REGISTRY_ADDR` | HTTP listen address | `:8080` |
| `SKILL_REGISTRY_DATABASE_URL` | SQLite or PostgreSQL URL | `file:data/skill-registry.db` |
| `SKILL_REGISTRY_SHARED_WORKSPACE_ROOT` | Writable packaging workspace root | `data/packaging-workspaces` |
| `SKILL_REGISTRY_DISABLE_SEED` | Disable sample catalog seed data | `true` |
| `SKILL_REGISTRY_SERVER_MODE` | Route-serving mode | all routes |
| `SKILL_REGISTRY_INSTALL_TARGET_MODE` | Tarball access mode: `proxy` or `direct` | `proxy` |
| `SKILL_REGISTRY_SKILLS_SH_URL` | External search provider URL | `https://skills.sh` |
| `SKILL_REGISTRY_AUTH_TOKENS_JSON` | Additional token-to-role mappings | unset |

Database-pool, HTTP-timeout, read-cache, and request-limit settings use the `SKILL_REGISTRY_DB_*`, `SKILL_REGISTRY_HTTP_*`, `SKILL_REGISTRY_READ_*`, and `SKILL_REGISTRY_WRITE_MAX_IN_FLIGHT` environment variables. Their defaults are defined in `cmd/skill-registry/main.go`.

## Artifact repository configuration

| Variable | Purpose |
| --- | --- |
| `SKILL_REGISTRY_ARTIFACTS_PATH` | JSON artifact fixture source for tests or local setups |
| `SKILL_REGISTRY_ARTIFACT_READ_URL` | Artifact repository URL for server reads |
| `SKILL_REGISTRY_NPM_REGISTRY_URL` | Fallback artifact repository URL |
| `SKILL_REGISTRY_ARTIFACT_READ_AUTHORIZATION` | Read authorization header |
| `SKILL_REGISTRY_NPM_READ_AUTHORIZATION` | Fallback read authorization header |
| `SKILL_REGISTRY_ARTIFACT_PUBLISH_AUTHORIZATION` | Publish authorization header |
| `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION` | Fallback publish authorization header |
| `SKILL_REGISTRY_NPM_PUBLISH_TOKEN` | Bearer-token fallback for repository access |
| `SKILL_REGISTRY_PUBLIC_INSTALL_URL` | Public artifact URL returned to clients |
| `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` | Fallback public artifact URL |

Artifact source precedence is `ARTIFACTS_PATH`, then `ARTIFACT_READ_URL`, then `NPM_REGISTRY_URL`. Explicit artifact authorization takes precedence over its npm fallback. Private repository URLs and credentials are never returned in install targets.

## External search and install transport

Catalog results are always available. External results from the configured `skills.sh` adapter are included only when a request sets `include_external=true` or selects `source=skills.sh` or `source=all`.

An install target has one transport mode:

- `proxy` fetches the tarball through the server's npm proxy mount;
- `direct` returns the upstream npm-compatible URL.

Transport does not change package identity, release selection, visibility, or governance.
