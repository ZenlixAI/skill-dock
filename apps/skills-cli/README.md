# skills CLI

The `skills` CLI installs registry packages and local or git sources, manages local installations, and invokes catalog publication workflows. The catalog server owns hosted releases, governance, and registry install resolution.

## Configuration

```bash
export SKILLS_REGISTRY_URL=https://catalog.example.com
export SKILLS_REGISTRY_TOKEN=<token>
```

`SKILLS_REGISTRY_ROLE` sends the development `X-Role` header when the server accepts header authentication. For direct npm traffic, use `NPM_CONFIG_USERCONFIG` or a local `.npmrc`.

For user accounts, configure Logto device flow and run `skills login`:

```bash
export SKILLS_LOGTO_ENDPOINT=https://your-tenant.logto.app
export SKILLS_LOGTO_CLIENT_ID=<device-flow-app-id>
export SKILLS_LOGTO_API_RESOURCE=https://catalog.example.com/api
npx skills login
```

The refresh token is stored in `${XDG_CONFIG_HOME:-~/.config}/skill-dock/credentials.json` with `0600` permissions. Use `skills whoami`, `skills org list`, and `skills org use <slug>` to inspect and switch the active tenant. `SKILLS_REGISTRY_TOKEN` remains available for CI and service accounts.

For automatic imports after `add`, the CLI merges `${XDG_CONFIG_HOME:-~/.config}/skills/config.json` with the nearest project `skills.config.json`. Project values override global values.

```json
{
  "syncOnAdd": true,
  "packageNamePrefix": "@acme.skill/",
  "defaultVersion": "0.1.0",
  "registry": "https://catalog.example.com"
}
```

Set `syncOnAdd` to `false` to disable automatic import. The operation is best-effort and does not change a source installation into a registry installation.

## Commands

| Command | Purpose |
| --- | --- |
| `npx skills find [query]` | Search the catalog and optional external results |
| `npx skills add <source>` | Install a registry package or a source-discovered skill |
| `npx skills list` | List local installations |
| `npx skills remove [skills]` | Remove local installations |
| `npx skills update [skills]` | Refresh registry-installed skills |
| `npx skills sync [skills]` | Refresh local or git source-installed skills |
| `npx skills init [name]` | Create a skill scaffold |
| `npx skills inspect [path]` | Validate a local skill directory |
| `npx skills publish [path]` | Package and publish a local skill through Catalog |
| `npx skills import-local [path]` | Publish a local source through Catalog |
| `npx skills import-git <source>` | Discover and publish a git-source skill through Catalog |
| `npx skills import-npm <package>` | Download an npm package and publish its extracted source |
| `npx skills login` | Sign in with the Logto device flow |
| `npx skills logout` | Remove the local Logto credential |
| `npx skills whoami` | Show the current account |
| `npx skills org list` | List organizations |
| `npx skills org use <slug>` | Select the active tenant |

## Examples

```bash
npx skills find pdf --include-external
npx skills add @platform.skill/pdf-skill --version 1.2.0
npx skills add owner/repo --skill pdf
npx skills add gitlab:group/repo --skill pdf
npx skills add https://gitlab.com/group/repo/-/tree/main/skills/pdf
npx skills add ./local-skills --list
npx skills update @platform.skill/pdf-skill
npx skills sync legacy-skill
npx skills publish ./my-skill
npx skills import-local ./legacy-skill --package-name @company.skill/legacy-skill --version 0.1.0
npx skills import-git owner/repo --skill pdf
npx skills import-npm @scope/skill-name --registry https://registry.npmjs.org
```

Use `skills --help` or a command's `--help` flag for the complete option list.

## Boundaries

- Registry and source installations are separate workflows.
- GitLab HTTP(S) sources are recognized when the hostname starts with `gitlab.` and are normalized to SSH clone URLs. HTTPS GitLab URLs become `git@host:group/repo.git`; use an `ssh://` URL directly for non-standard SSH ports.
- `update` resolves registry releases through the server; `sync` replays recorded local or git sources.
- The CLI does not publish directly to the artifact repository.
- Only a server-resolvable hosted release can be installed as a registry package.
