# skills CLI

> Command-line entry point for Skill Registry.

The CLI handles registry installs, local and git source installs, refreshes, inspection, packaging, and imports. The server owns catalog truth, releases, governance, and registry install resolution.

## Current model

- `SKILLS_REGISTRY_URL` points to the Catalog server.
- Registry package installs resolve an install target through the server first.
- Source installs support local paths, GitHub shorthand, and git URLs.
- `publish`, `import-local`, and `import-git` run through the server.
- `import-npm` downloads and extracts an npm package in the CLI, then submits it through the server import flow.
- `update` refreshes registry-installed skills only.
- `sync` refreshes local or git source-installed skills only.

## Automatic import after `add`

After a successful local or git source install, the CLI calls the server's local import workflow by default. The merged global `${XDG_CONFIG_HOME:-~/.config}/skills/config.json` and nearest project `skills.config.json` can control it: project values override global values.

```json
{
  "syncOnAdd": true,
  "packageNamePrefix": "@acme.skill/",
  "defaultVersion": "0.1.0",
  "registry": "http://127.0.0.1:8080"
}
```

- `syncOnAdd: false` disables automatic import.
- `packageNamePrefix` supplies a scoped package prefix when the source has no package name.
- `defaultVersion` supplies an exact semver when the source has no version.
- Configuration is merged from the optional global `${XDG_CONFIG_HOME:-~/.config}/skills/config.json` and the nearest `skills.config.json` found by walking up from the current working directory. Project values override global values, and command-line options override both. Package metadata still takes precedence over `packageNamePrefix` and `defaultVersion`.

Automatic import is best-effort. A server failure reports a warning but does not roll back the completed local install. It does not apply to registry packages and does not replace `skills sync`.

## Install target modes

The server resolves registry installs as either:

- `proxy`: download through the server proxy;
- `direct`: download from the upstream npm-compatible repository, using matching credentials from `NPM_CONFIG_USERCONFIG` or a `.npmrc` in the current working directory when needed.

The CLI does not choose a fallback release.

## Commands

| Command | Purpose |
| --- | --- |
| `npx skills add <source>` | Install a registry package or a skill from a source |
| `npx skills list` | List locally installed skills |
| `npx skills find [query]` | Search the catalog and optional external results |
| `npx skills remove [skills]` | Remove local installations |
| `npx skills update [skills]` | Refresh registry-installed skills |
| `npx skills sync [skills]` | Refresh local/git source-installed skills |
| `npx skills init [name]` | Create a skill scaffold |
| `npx skills inspect [path]` | Validate a local skill directory |
| `npx skills publish [path]` | Package and publish a local skill through the server |
| `npx skills import-local [path]` | Import a local source through the server |
| `npx skills import-git <source>` | Discover and import a skill from a git source |
| `npx skills import-npm <package>` | Download an npm package and submit it to the server import flow |

## Common examples

```bash
npx skills add @platform.skill/pdf-skill
npx skills add @platform.skill/pdf-skill --version 1.2.0
npx skills add owner/repo --skill pdf
npx skills add ./local-skills --list
npx skills find pdf --include-external
npx skills update @platform.skill/pdf-skill
npx skills sync legacy-skill
npx skills publish ./my-skill
npx skills import-local ./legacy-skill --package-name @company.skill/legacy-skill --version 0.1.0
npx skills import-git owner/repo --skill pdf
npx skills import-npm @scope/skill-name --registry https://registry.npmjs.org
```

## `add` options

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

## `find` options

- `--registry <url>`: Catalog server URL.
- `--include-external`: Include results from the configured `skills.sh` provider.
- `--source <catalog|skills.sh|all>`: Select the search domain.
- `--scope <scope>`: Filter by npm scope.
- `--origin <url>`: Filter by origin.
- `--json`: Print machine-readable output.

External results include provenance and currently return `installable: false`; only a server-resolvable hosted release can be installed as a registry package.

## `update` and `sync`

```bash
npx skills update [skills...] [-g|--global] [-p|--project] [-y|--yes]
npx skills sync [skills...] [-g|--global] [-p|--project] [-y|--yes]
```

`update` processes entries with `sourceType: registry`. `sync` processes `local`, `github`, and `git` entries using their recorded source, ref, and skill path. It rediscovers the source, reinstalls it for the original agents, and updates the content hash.

## Publication and import options

```bash
npx skills publish [path] [--registry <url>] [--strict]
npx skills import-local [path] [--registry <url>] [--package-name <name>] [--version <version>] [--publish-registry <url>] [--strict]
npx skills import-git <source> [--registry <url>] [--package-name <name>] [--version <version>] [--publish-registry <url>] [--skill <name>] [--full-depth] [--strict]
npx skills import-npm <package> [--registry <npm-url>] [--version <version>] [--hosted] [--strict]
```

`--strict` disables server-side normalization of imported source. `import-npm --hosted` currently changes only the import mode reported by the CLI; it does not change the server request or release ownership. Import commands do not currently support `--json`.

## Authentication

For a private Catalog server:

- `SKILLS_REGISTRY_TOKEN` sends `Authorization: Bearer <token>`.
- `SKILLS_REGISTRY_ROLE` sends `X-Role: <role>` when role-based authentication is enabled.

## Constraints

- `skills --help` is authoritative for the installed CLI's supported flags and examples.
- The CLI does not own catalog truth, release governance, or artifact storage.
- Registry and source installs remain distinct.
- The CLI must not bypass the server to publish directly to the artifact repository.
