# External sources and discovery

## Current behavior

The server catalog is the default discovery domain. External discovery is available only through the configured `skills.sh` adapter:

- `skills find <query> --include-external` requests catalog and external results;
- `skills find <query> --source skills.sh` requests external results;
- `skills find <query> --source all` requests both.

The server forwards the query, page size, and optional `registry`, `scope`, and `origin` values to the configured external provider. External results are returned with `sourceType: external`, `sourceKind: skills.sh`, an origin URL, and `installable: false`.

The default external provider URL is `https://skills.sh` and can be changed with `SKILL_REGISTRY_SKILLS_SH_URL`.

## Provenance

Catalog list, detail, version, and install-target models have fields for source and provenance, including:

- `sourceKind` and `sourceLocator`;
- `originRegistryUrl`, `originRepositoryUrl`, and `originRevision`;
- `importMode`;
- `hosted`, `mirrored`, and `installable`.

These fields are descriptive. Hosted release identity remains the package name plus exact version, and only server-owned release records are installable through registry resolution.

## Direct source workflows

The CLI supports local and git source installation without catalog discovery:

- `skills add ./path`;
- `skills add owner/repo`;
- `skills add <git-url>`.

It supports server publication from local, git, and npm sources through `import-local`, `import-git`, and `import-npm`. Local and git `add` also trigger a best-effort automatic import unless disabled by global or project `skills.config.json`.

## npm configuration

The CLI uses standard npm configuration for direct npm traffic:

- explicit `--registry`;
- `npm_config_registry` or `NPM_CONFIG_REGISTRY`;
- `NPM_CONFIG_USERCONFIG` or `.npmrc` for matching auth tokens.

This applies to direct tarball fetches and `import-npm`. It does not make arbitrary npm registries server-managed search sources.

## Not implemented

The following are design possibilities, not current behavior:

- external npm, git, or local search aggregation by `find`;
- a server-side npm import source type;
- server-recorded mirror or hosted import mode;
- promotion of an external mirrored release to hosted storage;
- release-record provenance writes from import input.

Do not describe these as current product behavior until server and CLI contracts implement them.
