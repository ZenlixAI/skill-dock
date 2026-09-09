# Artifact resolution

## Responsibility

The server resolves artifact metadata and bundle access. The artifact repository remains an external integration and can be replaced through configuration.

## Artifact source precedence

The server selects the artifact source in this order:

1. `SKILL_REGISTRY_ARTIFACTS_PATH`;
2. `SKILL_REGISTRY_ARTIFACT_READ_URL`;
3. `SKILL_REGISTRY_NPM_REGISTRY_URL`;
4. the static source used when no external source is configured.

The JSON source is intended for tests and local fixtures. Normal deployments use an npm-compatible repository.

## Authorization precedence

Read authorization:

1. `SKILL_REGISTRY_ARTIFACT_READ_AUTHORIZATION`;
2. `SKILL_REGISTRY_NPM_READ_AUTHORIZATION`;
3. `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION`;
4. `SKILL_REGISTRY_NPM_PUBLISH_TOKEN` as a Bearer token.

Publish authorization:

1. `SKILL_REGISTRY_ARTIFACT_PUBLISH_AUTHORIZATION`;
2. `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION`;
3. `SKILL_REGISTRY_NPM_PUBLISH_TOKEN` as a Bearer token.

## Public install URL precedence

The server exposes:

1. `SKILL_REGISTRY_PUBLIC_INSTALL_URL`, when set;
2. otherwise `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL`.

Private repository URLs and credentials are not part of the client contract.

## Install target modes

- `proxy` rewrites the tarball URL to the server's npm proxy mount;
- `direct` returns the upstream npm-compatible URL unchanged.

The server selects the mode using `SKILL_REGISTRY_INSTALL_TARGET_MODE`; `proxy` is the default. The mode changes only tarball transport and does not change release identity, authority, or governance.
