# Auth and access

The server authenticates callers and authorizes each operation. The CLI only forwards credentials.

## Catalog credentials

| Variable | Role |
| --- | --- |
| `SKILL_REGISTRY_READER_TOKEN` | Read catalog and audit data |
| `SKILL_REGISTRY_AUTHORING_TOKEN` | Review and package source input |
| `SKILL_REGISTRY_PUBLISHER_TOKEN` | Publish releases and apply governance changes |
| `SKILL_REGISTRY_AUTH_TOKENS_JSON` | Define additional token, role, actor, and package-scope mappings |

When no catalog token is configured, the server accepts `X-Role` as development header authentication. This mode is not enabled once a catalog token is configured.

Configured principals can be limited to package names or package-name prefixes through `allowedPackages` in `SKILL_REGISTRY_AUTH_TOKENS_JSON`.

## CLI credentials

| Variable | Purpose |
| --- | --- |
| `SKILLS_REGISTRY_TOKEN` | Sent as `Authorization: Bearer <token>` |
| `SKILLS_REGISTRY_ROLE` | Development `X-Role` header when the server accepts header authentication |

`--registry` changes the catalog server only for the current command. It does not persist credentials or registry state.

## Artifact repository credentials

The catalog service, not the CLI, uses artifact-repository credentials:

- `SKILL_REGISTRY_ARTIFACT_READ_AUTHORIZATION`
- `SKILL_REGISTRY_ARTIFACT_PUBLISH_AUTHORIZATION`
- `SKILL_REGISTRY_NPM_READ_AUTHORIZATION`
- `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION`
- `SKILL_REGISTRY_NPM_PUBLISH_TOKEN`

Do not expose these credentials in install targets or CLI output.
