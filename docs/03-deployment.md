# Deployment and local runtime

## Requirements

- Docker
- kind
- kubectl
- Go 1.26 or later
- Node.js 18 or later
- uv

## Local development

```bash
make dev
make dev-images
make down
```

`make dev` creates the local stack and initializes sample data. Use `make dev-images` when the kind cluster already exists and only rebuilt service images need to be reloaded. `make down` tears down the local stack.

## Local services and endpoints

The default stack contains:

- catalog server;
- PostgreSQL;
- external Artifact Keeper npm repository.

| Service | URL |
| --- | --- |
| Catalog server | <http://127.0.0.1:18080> |
| npm proxy | <http://127.0.0.1:18080/repository/skills-npm/> |

## Artifact repository overrides

When the artifact repository is not available at its default local address:

```bash
ARTIFACTKEEPER_INTERNAL_NPM_URL=http://host.docker.internal:30080/ \
ARTIFACTKEEPER_PUBLIC_NPM_URL=http://127.0.0.1:18080/repository/skills-npm/ \
make dev
```

The internal URL is used by server-side reads and publication. The public URL is used in client-facing install targets.

## CLI configuration

```bash
export SKILLS_REGISTRY_URL=http://127.0.0.1:18080
export SKILLS_REGISTRY_TOKEN=<reader-token>
```

Use a publisher token for packaging and publication:

```bash
export SKILLS_REGISTRY_TOKEN=<publisher-token>
skills publish /absolute/path/to/your-skill
```

For `direct` install targets and `import-npm`, the CLI reads npm credentials from `NPM_CONFIG_USERCONFIG` or a `.npmrc` in the current working directory. `proxy` targets use the URL returned by the server and do not use npm config for the tarball request.

## Production posture

Replace every sample or placeholder secret with a unique secret before deployment. Keep the database, packaging workspace, and private artifact repository writable only by the services that need them. Expose only the server's public endpoint and the configured public artifact URL to CLI clients.

See [Server architecture and configuration](01-server-architecture.md) and [Auth and access](04-auth-and-access.md) for the complete configuration and authorization model.
