# skills

> This directory is the `skills` CLI app inside the `skill_registry` monorepo. It is subtree-managed from `XmchxUp/skills`; keep monorepo-only changes minimal and documented.

This README covers the CLI surface that matches the current monorepo design.

## Current design

- `SKILLS_REGISTRY_URL` points to the Catalog entry point.
- Registry installs resolve through Catalog.
- Source installs come from a local path, GitHub shorthand, or a git URL.
- `publish`, `import-local`, and `import-git` all publish through Catalog.
- `update` is for registry-installed skills.
- `sync` is for source-installed skills.

For exact flags and live help, run `npx skills --help`.

## Install a skill

```bash
npx skills add @platform.skill/pdf-skill
npx skills add @platform.skill/pdf-skill --version 1.2.0
npx skills add owner/repo --skill pdf
npx skills add ./local-skills --list
```

`skills add` accepts:

- a registry package name
- a local path
- a GitHub shorthand like `owner/repo`
- a git URL

Common options:

- `-g, --global` — install to the global scope instead of the project scope
- `-a, --agent <agents...>` — target specific agents
- `-s, --skill <names>` — select specific skills from a source tree
- `-l, --list` — list discovered skills without installing
- `--all` — install all discovered skills
- `--full-depth` — search more deeply in source trees
- `-y, --yes` — skip confirmation prompts
- `--version <version>` — install an exact published version
- `--registry <url>` — override the Catalog entry point

## Core commands

| Command | Purpose |
| --- | --- |
| `npx skills add <source>` | Install a registry package or a source-discovered skill |
| `npx skills list` | List installed skills |
| `npx skills find [query]` | Search for skills |
| `npx skills remove [skills]` | Remove installed skills |
| `npx skills update [skills]` | Update registry-installed skills |
| `npx skills sync [skills]` | Refresh source-installed skills |
| `npx skills init [name]` | Create a new skill scaffold |
| `npx skills inspect [path]` | Validate a local skill directory |
| `npx skills publish [path]` | Validate, pack, and publish a local skill |
| `npx skills import-local [path]` | Publish a local source tree through Catalog |
| `npx skills import-git <source>` | Clone a git source and publish a selected skill through Catalog |

## Update and sync

Use `update` when the skill came from the registry:

```bash
npx skills update
npx skills update @platform.skill/pdf-skill
npx skills update -g
npx skills update -p
```

Use `sync` when the skill came from a local path or git source:

```bash
npx skills sync
npx skills sync legacy-skill
npx skills sync -g
npx skills sync -p
```

Short version:

- `update` = registry install refresh
- `sync` = source install replay
- `remove` = local uninstall only

## Publish and import

### `skills publish`

Publishes a local skill directory through Catalog.

```bash
npx skills publish
npx skills publish ./my-skill
npx skills publish ./my-skill --strict
npx skills publish ./my-skill --registry https://catalog.example.com
```

### `skills import-local`

Publishes a local source tree through Catalog.

```bash
npx skills import-local ./legacy-skill \
  --package-name @company.skill/legacy-skill \
  --version 0.1.0
```

### `skills import-git`

Clones a git source, discovers skills, and publishes the selected skill through Catalog.

```bash
npx skills import-git owner/repo
npx skills import-git owner/repo --skill pdf
npx skills import-git https://github.com/owner/repo?path=skills/pdf
```

## Local authoring

### `skills init`

Creates a new skill scaffold.

```bash
npx skills init
npx skills init my-skill
```

Creates:

- `SKILL.md`
- `package.json`
- `commands/`

### `skills inspect`

Validates a local skill directory before publish.

```bash
npx skills inspect
npx skills inspect ./my-skill
```

It checks:

- `SKILL.md` frontmatter
- `package.json` identity and version
- local packaging constraints such as nested `SKILL.md` files and unsafe paths

## Authentication

For private Catalog deployments, set:

- `SKILLS_REGISTRY_TOKEN` for `Authorization: Bearer <token>`
- `SKILLS_REGISTRY_ROLE` for `X-Role: <role>` when the deployment uses role-based local auth

## Notes

- `skills add` can install to project or global scope.
- `skills find` supports interactive search and query search.
- `skills --help` is the authoritative source for the current flag set.

## Related docs

- Root repo overview: [`../../README.md`](../../README.md)
- Current usage guide: [`../../docs/README.md`](../../docs/README.md)
