# AGENTS.md

This file provides guidance to AI coding agents working on the `skills` CLI codebase.

## Project overview

`skills` is the CLI for the open agent skills ecosystem.

## Current command model

- `add` installs registry packages or source-discovered skills.
- `find` searches skills.
- `list` shows installed skills.
- `remove` uninstalls local skills.
- `update` refreshes registry-installed skills.
- `sync` refreshes source-installed skills.
- `init` creates a skill scaffold.
- `inspect` validates a local skill directory.
- `publish`, `import-local`, and `import-git` publish through Catalog.

## Current design rules

- `SKILLS_REGISTRY_URL` points to the Catalog entry point.
- Registry installs resolve through Catalog.
- `update` only applies to registry-installed skills.
- `sync` only applies to source-installed skills.
- `publish` does not use a separate npm publish step in this repository.
- `import-local` and `import-git` are publish flows, not install-refresh flows.

## Useful commands

| Command | Purpose |
| --- | --- |
| `skills add <source>` | Install a registry skill or a source-discovered skill |
| `skills find [query]` | Search for skills interactively or by keyword |
| `skills list` | List installed skills |
| `skills remove [skills]` | Remove installed skills |
| `skills update [skills...]` | Update registry-installed skills |
| `skills sync [skills...]` | Refresh skills from source installs |
| `skills init [name]` | Create a new `SKILL.md` scaffold |
| `skills inspect [path]` | Validate a local skill directory |
| `skills publish [path]` | Validate, pack, and publish a local skill directory |
| `skills import-local [path]` | Publish a local source tree through Catalog |
| `skills import-git <source>` | Clone a git source and publish a selected skill through Catalog |

Aliases: `skills a` for `add`, `skills ls` for `list`, and `skills rm` for `remove`.

## Workflow notes

- Keep doc examples aligned with the current CLI help output.
- When changing command behavior, update the README files and command help text together.
- Prefer removing stale sections over appending new caveats.
