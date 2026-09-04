# AGENTS.md

This file provides guidance to AI coding agents working on the `skills` CLI codebase.

## Project overview

`skills` is the CLI for the Skill Registry platform.

## Command model

- `add` installs registry packages or source-discovered skills.
- `find` searches skills.
- `list` shows installed skills.
- `remove` uninstalls local skills.
- `update` refreshes registry-installed skills.
- `sync` refreshes source-installed skills.
- `init` creates a skill scaffold.
- `inspect` validates a local skill directory.
- `publish`, `import-local`, and `import-git` publish through the server.

## Design rules

- `SKILLS_REGISTRY_URL` points to the Catalog entry point.
- Registry installs resolve through the server.
- `update` only applies to registry-installed skills.
- `sync` only applies to source-installed skills.
- `publish` does not use a separate npm publish step in this repository.
- `import-local` and `import-git` are publish flows, not refresh flows.
- `skills --help` is the command-level source of truth for flags and examples.

## Workflow notes

- Keep command help and README examples aligned.
- Prefer removing stale sections over adding more caveats.
- Keep the CLI contract distinct from the server contract.
- Use `docs/02-cli-contract.md` as the doc anchor for command semantics.
