# Server architecture and configuration

## Responsibilities

The server is the source of truth for the registry.

It owns:

- skill discovery and metadata
- release records and visibility
- publish and import workflows
- install resolution
- governance and audit data
- private repository integration

Bundles remain in an external npm-compatible artifact repository.

## Runtime model

The server runs with a database and an artifact repository.

### Required server configuration

| Variable | Purpose |
| --- | --- |
| `SKILL_REGISTRY_ADDR` | Server listen address |
| `SKILL_REGISTRY_DATABASE_URL` | SQLite or PostgreSQL URL |
| `SKILL_REGISTRY_NPM_REGISTRY_URL` | Artifact repository used for metadata and bundle access |
| `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` | Artifact repository URL visible to clients |
| `SKILL_REGISTRY_READER_TOKEN` | Read token |
| `SKILL_REGISTRY_AUTHORING_TOKEN` | Authoring token |
| `SKILL_REGISTRY_PUBLISHER_TOKEN` | Publisher token |
| `SKILL_REGISTRY_NPM_READ_AUTHORIZATION` | Authorization header for private repository reads |
| `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION` | Authorization header for server-side publication |
| `SKILL_REGISTRY_INSTALL_TARGET_MODE` | Install target mode returned by the server (`proxy` or `direct`) |

### Optional server configuration

| Variable | Purpose |
| --- | --- |
| `SKILL_REGISTRY_ARTIFACTS_PATH` | JSON artifact source override used in tests or local setups |
| `SKILL_REGISTRY_ARTIFACT_READ_URL` | Explicit artifact read URL override |
| `SKILL_REGISTRY_ARTIFACT_READ_AUTHORIZATION` | Explicit read authorization override |
| `SKILL_REGISTRY_ARTIFACT_PUBLISH_AUTHORIZATION` | Explicit publish authorization override |
| `SKILL_REGISTRY_PUBLIC_INSTALL_URL` | Explicit public install URL override |
| `SKILL_REGISTRY_SHARED_WORKSPACE_ROOT` | Workspace root for packaging flows |
| `SKILL_REGISTRY_DISABLE_SEED` | Disable repository seed data |
| `SKILL_REGISTRY_SESSION_SECRET` | Session secret for local auth flows |
| `SKILL_REGISTRY_LOGIN_USERNAME` / `SKILL_REGISTRY_LOGIN_PASSWORD` | Bootstrap login credentials |
| `SKILL_REGISTRY_SESSION_USERS_JSON` | JSON session user map |
| `SKILL_REGISTRY_AUTH_TOKENS_JSON` | JSON token map for API auth |

## Install target modes

The server can expose install targets in either of two modes:

- `proxy`: rewrite tarball URLs to the server's npm proxy mount
- `direct`: return the upstream npm-compatible registry URLs unchanged

`proxy` is the default. `direct` is useful when the caller should connect to the registry configured in the catalog record or deployment metadata.

## Reading order

If you want the server model in smaller pieces, read these next:

1. [Auth and access](04-auth-and-access.md)
2. [Package and release model](05-package-and-release-model.md)
3. [Artifact resolution](06-artifact-resolution.md)
4. [Publish and import workflows](07-publish-and-import-workflows.md)
5. [Install resolution and refresh](08-install-resolution-and-refresh.md)
6. [Governance and visibility](09-governance-and-visibility.md)

## Design rule

Prefer explicit configuration over hidden behavior.

If a deployment needs a different artifact backend, it should change configuration, not the server contract.
