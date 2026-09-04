# Deployment and local runtime

## Local development

The supported local path uses the repository's development scripts.

```bash
make dev
make dev-images
make down
```

Use `make dev` to create the local stack and initialize sample data.
Use `make dev-images` when the kind cluster already exists and you only need to reload rebuilt service images without re-running the data initialization step.

## Requirements

- Docker
- kind
- kubectl
- Go 1.26 or later
- Node.js 18 or later
- uv

## Default local services

- catalog server
- PostgreSQL
- external artifact repository

## Default local endpoints

| Service | URL |
| --- | --- |
| Catalog server | <http://127.0.0.1:18080> |
| npm proxy | <http://127.0.0.1:18080/repository/skills-npm/> |

## Artifact repository overrides

Use these variables when the artifact repository is not running at the default local address:

```bash
ARTIFACTKEEPER_INTERNAL_NPM_URL=http://host.docker.internal:30080/ \
ARTIFACTKEEPER_PUBLIC_NPM_URL=http://127.0.0.1:18080/repository/skills-npm/ \
make dev
```

## CLI runtime

Set the registry endpoint and token before using the CLI:

```bash
export SKILLS_REGISTRY_URL=http://127.0.0.1:18080
export SKILLS_REGISTRY_TOKEN=reader-secret
```

If your server returns `direct` install targets, also make the relevant npm credentials available to the CLI. By default it reads the npm config file pointed to by `NPM_CONFIG_USERCONFIG`, or a local `.npmrc` in the working directory.

Use a publisher token for publish flows:

```bash
export SKILLS_REGISTRY_TOKEN=publisher-secret
skills publish /absolute/path/to/your-skill
```

## Operational idea

Keep deployment packaging boring:

- server process
- database
- artifact repository
- explicit environment variables
- no hidden registry assumptions

If a deployment needs a different artifact backend, it should be configured, not hard-coded into the docs or the CLI contract.
