# Auth and access

## Authority

Authentication determines who can call the server. The server's authorization policy determines which catalog, packaging, publication, and governance operations that principal may perform. The CLI sends credentials; it does not implement policy.

## Server credentials

| Variable | Purpose |
| --- | --- |
| `SKILL_REGISTRY_READER_TOKEN` | Read operations |
| `SKILL_REGISTRY_AUTHORING_TOKEN` | Authoring and packaging operations |
| `SKILL_REGISTRY_PUBLISHER_TOKEN` | Publication and governance operations |
| `SKILL_REGISTRY_AUTH_TOKENS_JSON` | Additional token-to-role mappings |
| `SKILL_REGISTRY_SESSION_SECRET` | Session authentication secret |
| `SKILL_REGISTRY_LOGIN_USERNAME` / `SKILL_REGISTRY_LOGIN_PASSWORD` | Bootstrap login credentials |
| `SKILL_REGISTRY_SESSION_USERS_JSON` | Session user map |

The exact role mapping is deployment configuration. Do not assume a reader token can publish or govern releases.

## CLI credentials

| Variable | Purpose |
| --- | --- |
| `SKILLS_REGISTRY_TOKEN` | Sent as `Authorization: Bearer <token>` |
| `SKILLS_REGISTRY_ROLE` | Optional `X-Role` header for role-based local auth |

`--registry` changes the catalog server for the current command only. It does not persist credentials or registry state.

## Artifact credentials

The server may need separate credentials for the npm-compatible artifact repository:

- `SKILL_REGISTRY_ARTIFACT_READ_AUTHORIZATION`
- `SKILL_REGISTRY_ARTIFACT_PUBLISH_AUTHORIZATION`
- `SKILL_REGISTRY_NPM_READ_AUTHORIZATION`
- `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION`
- `SKILL_REGISTRY_NPM_PUBLISH_TOKEN`

These credentials stay server-side. They must not be copied into public install targets or CLI output.

## Access principles

- read access is explicit;
- write access is narrower than read access;
- publication and governance require stronger authorization than installation;
- role policy remains deployment-driven;
- authentication configuration does not alter package identity or release semantics.
