# External Skill Sync Design

## Summary

`skills-cli` needs a first-class way to resynchronize skills that were originally installed from external sources such as a local directory, a GitHub repo, or a generic git URL.

The recommended design is to add a new `skills sync` command for source-based reinstall/update, while keeping `import-local` and `import-git` as registry-publish flows.

This preserves a clean command model:

- `skills add <source>`: initial install into agent skill directories
- `skills sync [skills...]`: refresh already-installed external-source skills
- `skills update [skills...]`: refresh registry-installed skills
- `skills import-local` / `skills import-git`: publish imported sources to the registry

## Problem

The current CLI supports two distinct source families:

1. Registry packages
2. External sources (`local`, `github`, `git`)

Registry packages already have a steady-state lifecycle:

- install with `skills add <package_name>`
- refresh with `skills update`

External sources do not.

Today, `skills add <source>` installs skills from external sources and records compatibility metadata in the local/global lock files, but there is no command that uses that metadata to resynchronize the installed copy later.

This creates a lifecycle gap:

- initial install exists
- remove exists
- refresh does not exist

## Existing Constraints

### `add`

`skills add` already:

- parses `local`, `github`, and `git` sources
- clones remote sources when needed
- discovers one or more skills from the source tree
- installs them into project/global agent directories
- writes legacy compatibility metadata such as `source`, `ref`, and `skillPath`

That means the core source-acquisition and source-install machinery already exists.

### `update`

`skills update` only operates on installed-view records whose `sourceType` is `registry` and that have a stable `packageName`.

That is correct for registry version resolution, but it means external-source installs are currently invisible to the update flow.

### `import-local` / `import-git`

These commands are not install commands.

They are publish commands that:

- inspect or clone a source tree
- package it for Catalog
- invoke Catalog one-shot publish

They belong to the registry publishing lifecycle, not the installed-skill lifecycle.

## Goals

- Add a supported way to resynchronize already-installed external-source skills.
- Reuse the source metadata already written into lock files where possible.
- Keep the command model understandable and avoid mixing install-sync with publish-to-registry.
- Align scope flags and filtering behavior with existing `update` / `remove` patterns.

## Non-Goals

- Do not change registry-installed skill updates; they stay on `skills update`.
- Do not merge publish flows into install-sync flows.
- Do not redesign Catalog packaging or release-record behavior.
- Do not add automatic background syncing; this is an explicit CLI action.

## Options Considered

### Option 1: Add `skills sync` for external-source installs

Behavior:

- `skills sync` refreshes installed skills whose lock entry points at a replayable external source.
- Supported source types: `local`, `github`, `git`
- Unsupported source types: `registry`, malformed legacy records, or records without replayable source metadata

Pros:

- Clear lifecycle complement to `skills add <source>`
- Keeps registry updates and source sync conceptually separate
- Fits current lock-file data model with minimal schema churn

Cons:

- Adds a new top-level command
- Some update-like behavior now exists in both `update` and `sync`

### Option 2: Extend `skills update` to also handle external sources

Behavior:

- `skills update` would branch on source type:
  - registry records resolve latest version
  - external-source records replay source acquisition and reinstall

Pros:

- Fewer commands
- “refresh what I installed” becomes a single entry point

Cons:

- One command would now mean two different refresh models
- Error handling and user messaging become harder to reason about
- The current `update` mental model is specifically registry-oriented

### Option 3: Merge `import-local` / `import-git` into `sync`

Behavior:

- `sync` would both reinstall external-source skills and optionally publish them to the registry

Pros:

- Reduces command count on paper

Cons:

- Bad semantic fit
- “sync local installation state” and “publish to registry” are different operations with different risk profiles
- Makes confirmation, output, and error handling significantly more confusing

## Recommendation

Choose Option 1.

Add `skills sync` as the source-based refresh command and keep `import-local` / `import-git` separate.

Longer term, if command count needs to shrink, the more coherent consolidation is to eventually fold `import-local` / `import-git` into a single publish-oriented entry point such as `skills publish --from <source>`. That is independent of `skills sync` and should not block it.

## Command Design

### Basic usage

```bash
skills sync
skills sync foo bar
skills sync --project
skills sync --global
skills sync --yes
```

### Scope behavior

- `--project`: sync project-installed source skills only
- `--global`: sync global-installed source skills only
- default behavior should mirror `update`:
  - when non-interactive or `--yes`, prefer project if project skills exist, otherwise global
  - when interactive, prompt for scope if needed

### Filtering behavior

- Positional skill arguments filter by installed aliases, matching current installed-view style behavior
- Registry-installed skills are ignored by `sync`
- If the filter matches only registry installs, report that no source-installed skills matched

## Data Model

`skills sync` should read from the installed view plus the existing lock-file compatibility metadata:

- `sourceType`
- `legacy.source`
- `legacy.ref`
- `legacy.skillPath`

No new schema is required for the first version.

The existing compatibility metadata is sufficient because:

- `legacy.source` identifies the local path or remote source string
- `legacy.ref` preserves branch/tag selection when present
- `legacy.skillPath` identifies the repo-local `SKILL.md` location used during the original install

## Sync Algorithm

For each targeted installed skill:

1. Validate that the record is source-installed and replayable.
2. Parse `legacy.source` back through the existing source parser.
3. Re-acquire the source tree:
   - local path: use the path directly
   - github/git: clone into a temp directory
4. Rediscover skills under the same search root behavior used by `add`.
5. Re-identify the original skill using `legacy.skillPath`.
6. Reinstall the skill into the same scope and agent layout using the existing installer path.
7. Recompute the stored hash and refresh lock metadata.

Important rule:

- Sync must not “guess” when the original skill cannot be unambiguously identified.

If the repo layout changed and the recorded `skillPath` no longer resolves to exactly one discovered skill, sync should fail with an actionable error.

## Failure Modes

`skills sync` should fail clearly for these cases:

- missing `legacy.source`
- unsupported `sourceType`
- local source path no longer exists
- remote source cannot be cloned
- no skills found in the replayed source
- recorded `skillPath` no longer resolves
- multiple candidate skills match an ambiguous record
- installed record is orphaned and local state must be repaired first

The command should continue processing other requested skills where practical and summarize successes, failures, and skips at the end, similar to `update`.

## Testing Strategy

Use TDD.

Minimum behavior coverage:

1. project scope sync for a local-source install refreshes files and rewrites the local lock entry
2. global scope sync for a git/github-source install replays the source and rewrites the global lock entry
3. registry-installed records are ignored by sync
4. orphaned source-installed records are skipped with a repair message
5. missing `legacy.source` produces a replayability error
6. changed source layout causing unresolved `skillPath` produces a deterministic failure
7. positional skill filters only target matching source-installed skills
8. CLI help and usage text explain the install-sync semantics clearly

## Open Follow-Up

After `skills sync` ships, evaluate whether publish flows should be cleaned up separately:

- keep `import-local` / `import-git` as-is
- or replace them with a unified publish entry point later

That cleanup is intentionally out of scope for this change.
