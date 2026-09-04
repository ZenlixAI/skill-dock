# Auth and access

## Purpose

The registry keeps authentication separate from catalog truth.

Authentication decides who can talk to the server. The server decides what is published, visible, and installable.

## Server-side auth inputs

The current server contract supports token-based and session-based access patterns.

### Token-based access

| Variable | Purpose |
| --- | --- |
| `SKILL_REGISTRY_READER_TOKEN` | Read operations |
| `SKILL_REGISTRY_AUTHORING_TOKEN` | Authoring and packaging workflows |
| `SKILL_REGISTRY_PUBLISHER_TOKEN` | Publishing and governance operations |
| `SKILL_REGISTRY_AUTH_TOKENS_JSON` | Additional token map used by the server |

### Session-based access

| Variable | Purpose |
| --- | --- |
| `SKILL_REGISTRY_SESSION_SECRET` | Session secret |
| `SKILL_REGISTRY_LOGIN_USERNAME` / `SKILL_REGISTRY_LOGIN_PASSWORD` | Bootstrap login credentials |
| `SKILL_REGISTRY_SESSION_USERS_JSON` | JSON user map for session-based access |

## CLI auth inputs

| Variable | Purpose |
| --- | --- |
| `SKILLS_REGISTRY_TOKEN` | Bearer token used by the CLI |
| `SKILLS_REGISTRY_ROLE` | Optional role header for deployments that use role-based local auth |

## Access principles

- read access should be explicit
- write access should be narrower than read access
- publish access should be separate from install access
- role-based access should stay deployment-driven, not hard-coded into the CLI
- auth configuration should not change the catalog contract

## Current contract

The docs and code should keep these ideas distinct:

- **server**: owns auth policy and catalog truth
- **CLI**: sends credentials and executes workflows
- **artifact repository**: may require its own authorization header

If a deployment needs a different auth mechanism, it should be added as a configuration or adapter choice rather than a new registry model.
