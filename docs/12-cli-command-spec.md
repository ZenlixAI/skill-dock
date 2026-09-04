# CLI command and parameter specification

## Purpose

This document defines the canonical `skills` command surface, argument syntax, and flag behavior.

It is the command-level contract for discovery, installation, refresh, scaffold, publish, and import workflows.

## Scope

This document covers:

- command names and positional arguments
- flag names, accepted values, and precedence
- human and JSON output expectations
- provenance handling for external sources
- how the CLI consults registry configuration and npm configuration

This document does not define:

- internal implementation details
- database schema
- artifact storage internals
- UI behavior

## Command surface

| Command | Purpose |
| --- | --- |
| `find` | Search the registry catalog and optional external sources |
| `add` | Install a registry-backed or source-discovered skill |
| `list` | Show installed skills |
| `remove` | Uninstall local skills |
| `update` | Refresh registry-installed skills |
| `sync` | Refresh source-installed skills |
| `init` | Create a skill scaffold |
| `inspect` | Validate a local skill directory |
| `publish` | Validate, pack, and publish a local skill directory through the server |
| `import-local` | Publish a local source tree through the server |
| `import-git` | Clone a git source and publish a discovered skill through the server |
| `import-npm` | Import a package from an npm-compatible registry |

## Global conventions

### JSON output

Any command that supports `--json` should return stable machine-readable output.

JSON output must include provenance fields when the command resolves or imports an external source.

### Registry override

`--registry <url>` is an invocation-level override.

It does not persist as registry state and does not change the server contract.

### npm configuration

The CLI may read npm configuration only when the workflow uses a direct npm source or a direct tarball fetch.

It may read:

- `NPM_CONFIG_USERCONFIG`
- a local `.npmrc`
- scope-specific registry mappings
- npm authorization entries already present in the user's config

## `find`

### Synopsis

```bash
skills find [query]
```

### Purpose

Search the registry catalog first.

External npm registries, git sources, and local mirrors are opt-in search sources.

### Flags

| Flag | Purpose |
| --- | --- |
| `--include-external` | Merge external results into the output |
| `--source <catalog\|npm\|git\|local\|all>` | Select the search domain |
| `--registry <url>` | Search a specific npm-compatible registry for this invocation |
| `--scope <scope>` | Filter npm-style results by package scope |
| `--origin <url>` | Filter results by upstream registry or repository |
| `--json` | Return structured output |

### Behavior

- Default search is catalog-only.
- `--include-external` is shorthand for searching all supported sources.
- `--source` wins over `--include-external` when both are present.
- Search results should expose package name, version if known, source kind, origin, and installability.
- The CLI should keep provenance visible in human and JSON output.

## `add`

### Synopsis

```bash
skills add <source>
```

### Purpose

Install a registry skill or a source-discovered skill.

### Source forms

`<source>` may be one of:

- a package name
- a local path
- a GitHub shorthand reference
- a git URL

### Flags

| Flag | Purpose |
| --- | --- |
| `-g, --global` | Install for the global scope |
| `-a, --agent <agents...>` | Limit installation to specific agents |
| `-s, --skill <names>` | Select one or more skills from a source |
| `-l, --list` | List discovered skills instead of installing immediately |
| `--all` | Select all discovered skills |
| `--full-depth` | Clone or inspect source at full depth when needed |
| `-y, --yes` | Skip interactive confirmation |
| `--version <version>` | Pin the package version when applicable |
| `--registry <url>` | Resolve a package source against a specific registry |
| `--copy` | Copy files instead of linking when supported |
| `--verbose` | Emit verbose resolution details |
| `--output-dir <path>` | Write materialized output to a specific directory |
| `--json` | Return structured output |

### Behavior

- Registry-backed installs must resolve through the server first when the package exists in the catalog.
- Source installs may resolve directly from a local path or git source.
- When a direct npm fetch is required, the CLI may use the user's npm configuration.
- `--registry` only affects resolution for this invocation.
- `--version` only applies when versioned resolution is meaningful for the selected source.

## `list`

### Synopsis

```bash
skills list
```

### Flags

| Flag | Purpose |
| --- | --- |
| `-g, --global` | Show globally installed skills |
| `-p, --project` | Show project-scoped installs |
| `--all-scopes` | Show installs across all scopes |
| `-a, --agent <agents>` | Filter by agent |
| `--json` | Return structured output |

### Behavior

- `list` reports local installed state only.
- It does not query catalog truth.

## `remove`

### Synopsis

```bash
skills remove [skills]
```

### Behavior

- `remove` operates on local installed state only.
- It does not change catalog records.

## `update`

### Synopsis

```bash
skills update [skills]
```

### Flags

| Flag | Purpose |
| --- | --- |
| `-g, --global` | Update globally installed skills |
| `-p, --project` | Update project-scoped installs |
| `-y, --yes` | Skip interactive confirmation |

### Behavior

- `update` applies only to registry-installed skills.
- It follows the original install target mode returned by the server.
- `proxy` installs continue to download through the server proxy.
- `direct` installs continue to download from the upstream registry.

## `sync`

### Synopsis

```bash
skills sync [skills]
```

### Flags

| Flag | Purpose |
| --- | --- |
| `-g, --global` | Sync globally installed skills |
| `-p, --project` | Sync project-scoped installs |
| `-y, --yes` | Skip interactive confirmation |

### Behavior

- `sync` applies only to source-installed skills.
- It does not participate in registry access mode.

## `init`

### Synopsis

```bash
skills init [name]
```

### Behavior

- `init` creates a skill scaffold.
- It should not assume registry connectivity.

## `inspect`

### Synopsis

```bash
skills inspect [path]
```

### Behavior

- `inspect` validates a local skill directory.
- It should fail fast on malformed metadata or missing files.

## `publish`

### Synopsis

```bash
skills publish [path]
```

### Flags

| Flag | Purpose |
| --- | --- |
| `--registry <url>` | Use a specific server or registry endpoint |
| `--strict` | Treat validation warnings as errors |
| `--json` | Return structured output |

### Behavior

- `publish` validates, packs, and publishes a local skill directory through the server.
- It should not publish directly to the artifact repository.
- The server remains authoritative for the resulting release record.

## `import-local`

### Synopsis

```bash
skills import-local [path]
```

### Flags

| Flag | Purpose |
| --- | --- |
| `--package-name <name>` | Inject or override the package name |
| `--version <version>` | Inject or override the version |
| `--registry <url>` | Resolve against a specific server or registry endpoint |
| `--publish-registry <url>` | Set the target publish registry when needed |
| `--json` | Return structured output |

### Behavior

- `import-local` publishes a local source tree through the server.
- It is the path for turning a local directory into a server-recognized release.
- The import should preserve package identity once the server has accepted it.

## `import-git`

### Synopsis

```bash
skills import-git <source>
```

### Flags

| Flag | Purpose |
| --- | --- |
| `--skill <name>` | Select a discovered skill from the repository |
| `--full-depth` | Clone the repository with full history when needed |
| `--registry <url>` | Resolve the import through a specific server or registry endpoint |
| `--json` | Return structured output |

### Behavior

- `import-git` clones a git source and publishes a discovered skill through the server.
- It should preserve the upstream repository and revision in provenance data.

## `import-npm`

### Synopsis

```bash
skills import-npm <package>
```

### Flags

| Flag | Purpose |
| --- | --- |
| `--registry <url>` | Read package metadata from a specific npm-compatible registry |
| `--version <version>` | Import a specific package version |
| `--mirror` | Keep the imported release tied to the upstream source |
| `--hosted` | Promote the import into the registry's hosted release model |
| `--json` | Return structured output |

### Behavior

- `import-npm` resolves package metadata from an npm-compatible registry.
- The server validates the source and decides whether the result is mirrored or hosted.
- The imported release must preserve the original package name and version.
- `--mirror` is the default import posture for external npm sources.
- `--hosted` promotes the imported release into the registry's hosted ownership model.
- The command must not silently rewrite package identity.

## Output requirements

### Human output

Human output should be optimized for scanning.

It should show:

- package name
- version
- source kind
- origin
- visibility or installability when relevant

### JSON output

JSON output should be stable and explicit.

For imported or discovered external sources, it should include:

- `package_name`
- `version`
- `source_kind`
- `source_locator`
- `origin_registry_url` or `origin_repository_url`
- `import_mode` when relevant
- `visibility`
- `installable`
- `install_target_mode` when relevant

## Precedence rules

1. Explicit command flags win.
2. Server-returned install targets win over local assumptions.
3. User npm configuration is consulted only for direct npm discovery or direct tarball fetches.
4. Deployment defaults apply only when nothing more specific is provided.

## Contract rule

The CLI must not invent a second registry model.

It should use server responses for registry-backed workflows and standard npm configuration for direct external npm workflows.
