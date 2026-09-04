# skills

> CLI for the Skill Registry platform.

This package is the command surface for registry installs and source-backed installs. The registry contract lives in the server; the CLI resolves that contract and performs local workflows.

## Current model

- `SKILLS_REGISTRY_URL` points to the Catalog entry point.
- Registry installs resolve through the server.
- Source installs can come from a local path, GitHub shorthand, or a git URL.
- `publish`, `import-local`, and `import-git` go through the server.
- `update` applies to registry-installed skills.
- `sync` applies to source-installed skills.

## Install behavior

Registry-backed installs can be resolved in two access modes:

- `proxy` — the server rewrites tarball URLs to its own npm proxy mount
- `direct` — the server returns upstream npm-compatible registry URLs unchanged, so the CLI can use the user's own npm configuration and credentials

This is a transport choice for install resolution, not a separate skill identity model. When the server returns `direct`, the CLI may read npm auth from the user's npm config file (`NPM_CONFIG_USERCONFIG` or the local `.npmrc`) to download the tarball. It does not use npm config for `proxy` targets.

## Commands

| Command | Purpose |
| --- | --- |
| `npx skills add <source>` | Install a registry skill or a source-discovered skill |
| `npx skills list` | List installed skills |
| `npx skills find [query]` | Search for skills |
| `npx skills remove [skills]` | Remove installed skills |
| `npx skills update [skills]` | Refresh registry-installed skills |
| `npx skills sync [skills]` | Refresh source-installed skills |
| `npx skills init [name]` | Create a skill scaffold |
| `npx skills inspect [path]` | Validate a local skill directory |
| `npx skills publish [path]` | Validate, pack, and publish a local skill |
| `npx skills import-local [path]` | Publish a local source tree through the server |
| `npx skills import-git <source>` | Clone a git source and publish a discovered skill through the server |

## Core options

### `add`

```bash
npx skills add @platform.skill/pdf-skill
npx skills add @platform.skill/pdf-skill --version 1.2.0
npx skills add owner/repo --skill pdf
npx skills add ./local-skills --list
```

Useful flags:

- `-g, --global`
- `-a, --agent <agents...>`
- `-s, --skill <names>`
- `-l, --list`
- `--all`
- `--full-depth`
- `-y, --yes`
- `--version <version>`
- `--registry <url>`
- `--copy`
- `--verbose`
- `--output-dir <path>`
- `--json`

### `list`

Useful flags:

- `-g, --global`
- `-p, --project`
- `--all-scopes`
- `-a, --agent <agents>`
- `--json`

### `find`

Useful flags:

- `--registry <url>`
- `--json`

### `update` and `sync`

```bash
npx skills update
npx skills update @platform.skill/pdf-skill
npx skills sync
npx skills sync legacy-skill
```

Useful flags:

- `-g, --global`
- `-p, --project`
- `-y, --yes`

### `publish` and import

```bash
npx skills publish ./my-skill
npx skills publish ./my-skill --strict
npx skills import-local ./legacy-skill --package-name @company.skill/legacy-skill --version 0.1.0
npx skills import-git owner/repo --skill pdf
```

Useful flags:

- `--registry <url>`
- `--strict`
- `--package-name <name>`
- `--version <version>`
- `--publish-registry <url>`
- `--skill <name>`
- `--full-depth`

## Authentication

For private server deployments, set:

- `SKILLS_REGISTRY_TOKEN` for `Authorization: Bearer <token>`
- `SKILLS_REGISTRY_ROLE` for `X-Role: <role>` when role-based local auth is enabled

## Notes

- `skills --help` is the authoritative command reference.
- Keep docs and help output aligned.
- Keep registry installs and source installs distinct.
