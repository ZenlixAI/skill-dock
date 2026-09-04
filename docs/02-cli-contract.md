# CLI contract and command model

## Role of the CLI

`skills` is the operational entry point for developers and automation.

It talks to the server for registry-backed workflows and performs local install or refresh work for source-backed workflows.

## Command model

- `find` searches the registry
- `add` installs a registry package or source-discovered skill
- `list` shows installed skills
- `remove` uninstalls local skills
- `update` refreshes registry-installed skills
- `sync` refreshes source-installed skills
- `init` creates a skill scaffold
- `inspect` validates a local skill directory
- `publish` validates, packs, and publishes a local skill directory through the server
- `import-local` publishes a local source tree through the server
- `import-git` clones a git source and publishes a discovered skill through the server

## Core rules

- `SKILLS_REGISTRY_URL` points to the catalog entry point
- registry installs resolve through the server
- source installs can come from a local path, GitHub shorthand, or a git URL
- `publish`, `import-local`, and `import-git` are publish flows
- `update` only applies to registry-installed skills
- `sync` only applies to source-installed skills
- `list` and `remove` work on local installed state only
- `skills --help` is the authoritative command reference for flags and examples

## Registry install behavior

Registry installs always resolve the package through the server first. The resulting install target includes the registry access mode:

- `proxy` means the CLI uses the server-rewritten tarball URL
- `direct` means the CLI uses the upstream npm-compatible registry URL and may attach credentials from the user's npm configuration

That mode is a server decision, not a CLI override. When the server returns `direct`, the CLI may read npm auth from `NPM_CONFIG_USERCONFIG` or the local `.npmrc` for the tarball request. It does not use npm config for `proxy` targets.

## Update and refresh behavior

`update` still only applies to registry-installed skills, but it now follows the same install target mode as the original install:

- `proxy` targets continue to download through the server proxy
- `direct` targets continue to download directly from the user's chosen registry

`sync` remains the source-install refresh path and does not participate in registry access mode.

## Short workflow summary

```bash
skills find pdf
skills add @company.skill/your-skill -y
skills update @company.skill/pdf-skill
skills sync legacy-skill
skills publish /absolute/path/to/your-skill
skills import-local ./legacy-skill --package-name @company.skill/legacy-skill --version 0.1.0
skills import-git owner/repo --skill pdf
```

## What the CLI should not do

- it should not own registry truth
- it should not assume storage details
- it should not blur registry installs and source installs
- it should not invent server behavior outside the documented contract
- it should not treat the help output and the docs as separate sources of truth
