# CLI command reference

This reference describes the current `skills` command surface. Run `skills --help` for the installed binary's final wording.

## Common configuration

| Input | Meaning |
| --- | --- |
| `SKILLS_REGISTRY_URL` | Catalog server base URL |
| `SKILLS_REGISTRY_TOKEN` | Bearer token for catalog requests |
| `SKILLS_REGISTRY_ROLE` | Optional role header |
| `NPM_CONFIG_USERCONFIG` | npm auth config for direct npm requests |
| `skills.config.json` | Project config for automatic source import after `add` |

`--registry <url>` means catalog server for `add`, `find`, `publish`, `import-local`, and `import-git`. For `import-npm`, it selects the source npm registry.

## Commands

| Command | Purpose |
| --- | --- |
| `find [query]` | Search the catalog and optional configured external results |
| `add <source>` | Install a registry package or a local/git source |
| `list` | List local installed state |
| `remove [skills]` | Remove local installed skills |
| `update [skills]` | Refresh registry-installed skills |
| `sync [skills]` | Refresh local/git source-installed skills |
| `init [name]` | Create a local skill scaffold |
| `inspect [path]` | Validate a local skill directory |
| `publish [path]` | Package and publish a local skill through the server |
| `import-local [path]` | Package and publish a local source tree |
| `import-git <source>` | Discover one git-source skill and publish it |
| `import-npm <package>` | Download an npm package and submit it to the local import flow |

## `find`

```bash
skills find [query] [--registry <url>] [--include-external] [--source <kind>] [--scope <scope>] [--origin <url>] [--json]
```

- Default search is catalog-only.
- `--include-external` enables the configured external provider.
- `--source` is forwarded to the server. Current meaningful values are `catalog`, `skills.sh`, and `all`.
- `--scope` and `--origin` are forwarded to the server and external provider.
- `--json` returns the search result fields, including available provenance.

## `add`

```bash
skills add <source> [options]
```

`<source>` is a package name, local path, GitHub shorthand, or git URL.

| Flag | Meaning |
| --- | --- |
| `-g`, `--global` | Install globally |
| `-a`, `--agent <agents...>` | Limit installation to agents |
| `-s`, `--skill <names>` | Select discovered skills |
| `-l`, `--list` | List discovered skills without installing |
| `--all` | Select all discovered skills |
| `--full-depth` | Discover deeper source layouts |
| `-y`, `--yes` | Skip confirmation |
| `--version <version>` | Exact version for registry package installs |
| `--registry <url>` | Catalog server override |
| `--copy` | Copy rather than symlink where supported |
| `--verbose` | Print per-agent results |
| `--output-dir <path>` | Materialize a registry skill to a directory; requires `--json` |
| `--json` | Machine-readable registry install output |

For successful local or git installs, automatic server import is enabled unless `skills.config.json` contains `"syncOnAdd": false`. Its optional `packageNamePrefix`, `defaultVersion`, and `registry` values fill missing metadata and choose the import endpoint. Failure is a warning; the local install remains successful.

## `update` and `sync`

```bash
skills update [skills...] [-g|--global] [-p|--project] [-y|--yes]
skills sync [skills...] [-g|--global] [-p|--project] [-y|--yes]
```

`update` applies to registry entries. `sync` applies to `local`, `github`, and `git` entries with recorded source and skill path metadata. With no explicit scope, non-interactive execution chooses project scope when project skills exist, otherwise global scope.

## Authoring and publication

```bash
skills init [name]
skills inspect [path]
skills publish [path] [--registry <url>] [--strict]
skills import-local [path] [--registry <url>] [--package-name <name>] [--version <version>] [--publish-registry <url>] [--strict]
skills import-git <source> [--registry <url>] [--package-name <name>] [--version <version>] [--publish-registry <url>] [--skill <name>] [--full-depth] [--strict]
skills import-npm <package> [--registry <npm-url>] [--version <version>] [--hosted] [--strict]
```

`publish`, `import-local`, and `import-git` use the catalog server. `import-npm` first reads and downloads from npm using standard npm config, then uses the catalog URL from `SKILLS_REGISTRY_URL` for the import workflow. Import commands print human-readable output; they do not currently support `--json`.

`--strict` disables server-side normalization of imported source. `import-npm --hosted` currently changes the reported mode only; it does not change the server request.
