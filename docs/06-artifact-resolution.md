# Artifact resolution

## Purpose

The server needs a predictable way to resolve bundle access across local development and private deployments.

## Resolution order

### Artifact source

The server resolves artifact access in this order:

1. `SKILL_REGISTRY_ARTIFACTS_PATH`
2. `SKILL_REGISTRY_ARTIFACT_READ_URL`
3. `SKILL_REGISTRY_NPM_REGISTRY_URL`
4. static fallback when no artifact source is configured

### Read authorization

The server resolves read authorization in this order:

1. `SKILL_REGISTRY_ARTIFACT_READ_AUTHORIZATION`
2. `SKILL_REGISTRY_NPM_READ_AUTHORIZATION`
3. `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION`
4. `SKILL_REGISTRY_NPM_PUBLISH_TOKEN` as a Bearer token fallback

### Publish authorization

The server resolves publish authorization in this order:

1. `SKILL_REGISTRY_ARTIFACT_PUBLISH_AUTHORIZATION`
2. `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION`
3. `SKILL_REGISTRY_NPM_PUBLISH_TOKEN` as a Bearer token fallback

### Public install URL

The server resolves the public install URL in this order:

1. `SKILL_REGISTRY_PUBLIC_INSTALL_URL`
2. `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL`

## Practical meaning

- local development can point at an external Artifact Keeper or a JSON artifact source
- private deployments can use explicit read and publish authorization headers
- CLI/browser clients should only depend on the public install URL, not on private backend credentials

## Design rule

The registry should not hard-code a single artifact backend.

When the artifact source changes, the deployment should update configuration rather than the server contract.
