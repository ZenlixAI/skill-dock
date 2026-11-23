# CLI command reference

This document describes the implemented `skills` command surface. The CLI's `--help` output is the final source for option wording.

## Common configuration

| Input | Meaning |
| --- | --- |
| `SKILLS_REGISTRY_URL` | Catalog server base URL |
| `SKILLS_REGISTRY_TOKEN` | Bearer token for catalog requests (service account) |
| `SKILLS_REGISTRY_ROLE` | Development role header |
| `SKILLS_LOGTO_ENDPOINT` | Logto base URL for `skills login` |
| `SKILLS_LOGTO_CLIENT_ID` | Logto device-flow app ID |
| `SKILLS_LOGTO_API_RESOURCE` | Logto API resource for catalog tokens |
| `NPM_CONFIG_USERCONFIG` | npm credentials for direct npm traffic |
| `skills.config.json` | Project configuration for automatic import after `add` |

The CLI merges the optional `${XDG_CONFIG_HOME:-~/.config}/skills/config.json` with the nearest project `skills.config.json`; project values override global values.

`--registry <url>` selects the catalog server for `add`, `find`, `publish`, `import-local`, and `import-git`. For `import-npm`, it selects the source npm registry.

## Commands

| Command | Purpose |
| --- | --- |
| `find [query]` | Search the catalog and optional external results |
| `add <source>` | Install a registry package or local/git source |
| `list` | List local installed state |
| `remove [skills]` | Remove local installed skills |
| `update [skills]` | Refresh registry-installed skills |
| `sync [skills]` | Refresh local/git source-installed skills |
| `init [name]` | Create a local skill scaffold |
| `inspect [path]` | Validate a local skill directory |
| `publish [path]` | Package and publish a local skill |
| `import-local [path]` | Publish a local source tree |
| `import-git <source>` | Discover and publish a skill from git |
| `import-npm <package>` | Download an npm package and publish its extracted source |
| `login` | Sign in with the Logto device flow |
| `logout` | Remove the local Logto credential |
| `whoami` | Show the current account and organizations |
| `org list` | List organizations for the current account |
| `org use <slug>` | Select the active tenant |

## Account and tenant

```bash
skills login [--logto-endpoint <url>] [--client-id <id>] [--api-resource <uri>]
skills logout
skills whoami [--registry <url>]
skills org list [--registry <url>]
skills org use <slug> [--registry <url>]
```

`login` uses the OAuth 2.0 device authorization grant. The refresh token is stored under `${XDG_CONFIG_HOME:-~/.config}/skill-dock/credentials.json` with `0600` permissions. When an active organization is selected, token refresh requests an organization token; the catalog validates the `organization_id` claim. `SKILLS_REGISTRY_TOKEN` remains the service-account path and is not silently converted into a user session.

Private organization/private installs use the catalog proxy: the CLI sends the Skill Dock bearer token to the proxy, and the proxy checks the active tenant and install permission before forwarding to npm.

## Discovery and installation

```bash
skills find [query] [--registry <url>] [--include-external] [--source <catalog|skills.sh|all>] [--scope <scope>] [--origin <url>] [--json]
skills add <source> [options]
```

`add` accepts a package name, local path, GitHub shorthand, GitLab shorthand/URL, or git URL. GitLab HTTP(S) URLs are recognized when the hostname starts with `gitlab.` and are normalized to SSH clone URLs. Its source-install options include `--skill`, `--list`, `--all`, and `--full-depth`; registry installs also support `--version`, `--output-dir`, and `--json`.

Successful local and git installs attempt a server import by default. Set `syncOnAdd` to `false` to disable it. `packageNamePrefix`, `defaultVersion`, and `registry` can supply missing source metadata and select the import endpoint. Import failure is reported as a warning and does not roll back the local installation.

## Refresh

```bash
skills update [skills...] [-g|--global] [-p|--project] [-y|--yes]
skills sync [skills...] [-g|--global] [-p|--project] [-y|--yes]
```

`update` applies only to registry entries. `sync` applies only to `local`, `github`, `gitlab`, and `git` entries with recorded source and skill-path metadata. With no explicit scope, non-interactive execution chooses project scope when project skills exist, otherwise global scope.

## Authoring and publication

```bash
skills init [name]
skills inspect [path]
skills publish [path] [--registry <url>] [--strict]
skills import-local [path] [--registry <url>] [--package-name <name>] [--version <version>] [--publish-registry <url>] [--strict]
skills import-git <source> [--registry <url>] [--package-name <name>] [--version <version>] [--publish-registry <url>] [--skill <name>] [--full-depth] [--strict]
skills import-npm <package> [--registry <npm-url>] [--version <version>] [--hosted] [--strict]
```

`publish`, `import-local`, and `import-git` use the catalog server. `import-npm` downloads from npm using standard npm configuration, then sends the extracted source to the catalog URL in `SKILLS_REGISTRY_URL`. Import commands currently produce human-readable output only.

`--strict` disables server-side source normalization. `import-npm --hosted` changes only the mode reported by the CLI; it does not change the server request or release ownership.
