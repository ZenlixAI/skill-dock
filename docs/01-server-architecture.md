# Server architecture and configuration

## Responsibilities

The catalog server is the authority for:

- catalog listing and search
- skill and release metadata
- install-target resolution
- packaging review and publish workflows
- visibility and lifecycle governance
- audit and idempotency records

It uses an external npm-compatible artifact repository for package metadata and bundle bytes, and a database for catalog and operational state.

## HTTP surfaces

Read routes are served under `/api/v1`:

- `GET /skills`
- `GET /skills/search`
- `GET /skills/curated`
- `GET /skills/:package`
- `GET /skills/:package/versions`
- `GET /skills/:package/versions/:version`
- `GET /skills/:package/install-target`

Packaging routes are:

- `POST /skill-packaging/import-review`
- `POST /skill-packaging/publish`

Governance routes create release and lifecycle records. See the [API endpoint reference](13-api-endpoint-spec.md).

## Required runtime configuration

| Variable | Purpose |
| --- | --- |
| `SKILL_REGISTRY_ADDR` | Server listen address; default `:8080` |
| `SKILL_REGISTRY_DATABASE_URL` | SQLite or PostgreSQL URL |
| `SKILL_REGISTRY_NPM_REGISTRY_URL` | Internal npm-compatible artifact repository URL |
| `SKILL_REGISTRY_READER_TOKEN` | Read access token |
| `SKILL_REGISTRY_AUTHORING_TOKEN` | Authoring and packaging token |
| `SKILL_REGISTRY_PUBLISHER_TOKEN` | Publishing and governance token |

The public artifact URL and private repository authorization are deployment-dependent. Do not expose private backend credentials in install targets.

## Optional runtime configuration

| Variable | Purpose |
| --- | --- |
| `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` | Artifact URL returned to clients |
| `SKILL_REGISTRY_PUBLIC_INSTALL_URL` | Explicit public install URL override |
| `SKILL_REGISTRY_ARTIFACTS_PATH` | JSON artifact source for tests or local setups |
| `SKILL_REGISTRY_ARTIFACT_READ_URL` | Explicit artifact read URL |
| `SKILL_REGISTRY_ARTIFACT_READ_AUTHORIZATION` | Explicit artifact read authorization |
| `SKILL_REGISTRY_ARTIFACT_PUBLISH_AUTHORIZATION` | Explicit artifact publish authorization |
| `SKILL_REGISTRY_NPM_READ_AUTHORIZATION` | npm read authorization fallback |
| `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION` | npm publish authorization |
| `SKILL_REGISTRY_NPM_PUBLISH_TOKEN` | Bearer-token publish fallback |
| `SKILL_REGISTRY_SHARED_WORKSPACE_ROOT` | Writable packaging workspace root |
| `SKILL_REGISTRY_DISABLE_SEED` | Disable sample catalog seed data |
| `SKILL_REGISTRY_SKILLS_SH_URL` | External search provider URL; default `https://skills.sh` |
| `SKILL_REGISTRY_INSTALL_TARGET_MODE` | `proxy` or `direct`; default `proxy` |
| `SKILL_REGISTRY_AUTH_TOKENS_JSON` | Additional token-to-role mappings |
| `SKILL_REGISTRY_SESSION_SECRET` | Session auth secret |
| `SKILL_REGISTRY_LOGIN_USERNAME` / `SKILL_REGISTRY_LOGIN_PASSWORD` | Bootstrap session credentials |
| `SKILL_REGISTRY_SESSION_USERS_JSON` | Session user map |

Read and publish authorization use explicit artifact variables first, then the corresponding npm variables. The artifact source uses `SKILL_REGISTRY_ARTIFACTS_PATH`, then an explicit read URL, then `SKILL_REGISTRY_NPM_REGISTRY_URL`.

## External search

The runtime configures one external search adapter for `skills.sh`. External results are opt-in through `include_external=true` or `source=skills.sh|all`; catalog results remain available without the adapter. The server does not currently aggregate arbitrary npm registries, git repositories, or local directories.

## Install target modes

- `proxy` rewrites the tarball URL to the server's npm proxy mount;
- `direct` returns the upstream npm-compatible URL unchanged.

The mode affects only tarball retrieval. It is not a release or identity mode.

## Design rule

Keep the server authoritative for hosted catalog truth and keep storage replaceable through configuration.
