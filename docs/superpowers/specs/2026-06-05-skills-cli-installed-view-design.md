# skills-cli Installed View Module Design

## Summary

This design deepens the `skills-cli` architecture around one domain concept: the **Installed View**.

Today, `list`, `remove`, and `update` each reconstruct their own partial understanding of an **Installed Skill** by mixing:

- filesystem scans
- alias derivation
- lock-file lookups
- agent-link detection
- ad-hoc fallback rules

That makes the current seam shallow. Callers must know too much about how installation state is physically represented.

This design introduces a dedicated **Installed View module** that projects one authoritative read model for installed skills across each **Installation Scope**. The first refactor only deepens the read side and the state-projection side. It does not yet absorb install or publish lifecycle behavior.

## Goals

- Create a deep module for the **Installed View** concept.
- Make `list`, `remove`, and `update` consume one projection seam instead of re-scanning state independently.
- Promote **Package Name** to the primary identity in command behavior, consistent with ADR-0001.
- Surface broken or partially tracked install state explicitly instead of silently guessing through it.
- Improve locality for testing installed-skill state transitions and edge cases.

## Non-Goals

- Do not redesign the install workflow itself.
- Do not merge install, remove, and update into one lifecycle module in this iteration.
- Do not change Catalog resolution or packaging workflows.
- Do not remove compatibility support for historical source installs yet.

## Context

The project language defines:

- **Agent Skill** as the reusable capability unit
- **Package Name** as the only public identity in V1
- **Installation Scope** as the local target boundary
- **Installed View** as the CLI listing of locally installed skills across scopes

ADR-0001 requires the CLI to center **Package Name** as the primary public identity.
ADR-0002 keeps discovery and install resolution governed by the Catalog while npm stores artifacts.

This refactor strengthens both decisions by ensuring local installed-state projection does not depend on ad-hoc caller heuristics.

## Current Friction

### Repeated installed-state reconstruction

The current architecture spreads installed-state logic across:

- `src/installer.ts`
- `src/list.ts`
- `src/remove.ts`
- `src/update.ts`

Examples of duplicated or leaking logic:

- `list` rebuilds display metadata from local/global lock files.
- `remove` re-scans directories and reconstructs aliases from scratch.
- `update` infers registry installs from disk and `package.json` when lock state is incomplete.
- multiple callers know how install names, canonical paths, aliases, and agent-specific directories relate.

### Shallow seam

The current seam exposes directory-scan results rather than an actual **Installed View**. As a result, callers must know:

- where canonical installs live
- where agent-specific links or copies live
- how to recover aliases
- how to map lock entries back to disk
- when to guess based on `package.json`

Deleting this logic from the existing modules would not remove complexity. It would simply reappear in each caller. That is the signal that a deeper module should own it.

## Approaches Considered

### Approach A: Thin projection wrapper

Add a small wrapper around `listInstalledSkills` and preserve most caller-side heuristics.

Pros:

- lowest change surface
- minimal migration effort

Cons:

- keeps alias, canonical-path, and orphan handling outside the seam
- does not materially improve locality
- preserves the shallow-interface problem

### Approach B: Installed View module

Introduce a dedicated module that collects raw installed-state facts and projects them into one read model used by `list`, `remove`, and `update`.

Pros:

- strongest locality improvement without rewriting all lifecycle behavior
- isolates filesystem and lock correlation behind one seam
- supports cleaner command output and cleaner internal types

Cons:

- changes internal types and some user-visible output
- requires migration across several callers

### Approach C: Full installed lifecycle module

Absorb list, remove, and update behavior into one higher-level module.

Pros:

- deepest eventual leverage

Cons:

- too wide for the first refactor
- couples read-model changes with destructive behavior changes
- raises verification risk substantially

## Decision

Use **Approach B**.

This iteration introduces an **Installed View module** with a deep projection interface, then migrates `list`, `remove`, and `update` onto that seam. Remove and update remain separate command modules, but they will stop reconstructing installed state themselves.

## Module Design

### New module

Add a new module:

- `apps/skills-cli/src/installed-view.ts`

This module owns the seam for projecting installed skill state.

### Interface

The module should expose types along these lines:

```ts
type InstalledViewScope = 'project' | 'global';

type AgentLinkKind = 'canonical' | 'symlink' | 'copy' | 'missing';

interface InstalledSkillView {
  packageName?: string;
  displayName: string;
  installName: string;
  aliases: string[];
  scope: InstalledViewScope;
  canonicalPath: string;
  agentLinks: Array<{
    agent: AgentType;
    path: string;
    kind: AgentLinkKind;
  }>;
  version?: string;
  sourceType: string;
  registry?: string;
  pluginName?: string;
  orphaned: boolean;
  orphanReasons: string[];
  lockEntry?: SkillLockEntry | LocalSkillLockEntry;
}

interface ListInstalledViewOptions {
  scope?: InstalledViewScope;
  agentFilter?: AgentType[];
  cwd?: string;
}
```

The exact type names may change during implementation, but the interface must preserve these responsibilities:

- one projected record per installed skill per scope
- one place to obtain aliases
- one place to obtain canonical path
- one place to obtain agent links
- one place to obtain lock metadata
- one place to determine orphan state

## Projection Model

The module must be implemented in two phases.

### Phase 1: Collect

Collect raw installed-state facts without making command-specific decisions.

Collected facts include:

- canonical skill directories for each **Installation Scope**
- agent-specific skill directories for detected and existing agents
- parsed `SKILL.md` metadata
- lock-file entries for project and global scope
- path presence for canonical locations and agent-specific locations

This phase is allowed to know filesystem layout and lock-file shape.
Callers are not.

### Phase 2: Project

Project collected facts into `InstalledSkillView`.

Projection responsibilities:

- derive one `displayName`
- derive one `installName`
- derive alias set
- attach the matching lock entry if present
- attach agent links
- compute orphan state

## Identity Resolution Rules

Projection must use a stable merge precedence so callers do not depend on incidental scan order.

Suggested precedence:

1. `lock packageName`
2. `lock installName`
3. canonical directory install name
4. parsed `SKILL.md` display name aliases

This ordering keeps the interface aligned with ADR-0001:

- **Package Name** is primary when known
- install name is secondary
- display-name-derived matching is compatibility-only

## Alias Rules

`aliases` should be produced once by the module and include:

- `packageName` when present
- `installName`
- sanitized install directory names
- compatibility aliases derived from historical install-name rules
- display-name compatibility alias where needed for historical source installs

The alias list must be deduplicated and normalized for matching use.

Callers should no longer rebuild aliases themselves.

## Orphan Rules

The module must compute orphan state explicitly.

`orphaned = false` only when:

- projected identity is coherent
- canonical install location exists
- lock state and disk state can be correlated well enough to trust command behavior

`orphaned = true` in these cases:

- disk skill exists but no corresponding lock entry exists
- lock entry exists but canonical path does not exist
- only agent-specific path exists and canonical path is missing
- alias or install-name conflict maps multiple incompatible candidates to one projected identity
- projection can only be recovered through ambiguous fallback

The module should also retain `orphanReasons` so callers can produce clearer output without recomputing logic.

## Command Migration

### `list`

`list.ts` will consume `InstalledSkillView[]` directly.

It should stop:

- rebuilding display metadata maps
- reconstructing install-name fallbacks
- independently correlating skills to lock entries

Expected output changes:

- orphaned skills are explicitly labeled
- output distinguishes tracked vs untracked state more clearly
- all-scope output is based on projected scope records rather than ad-hoc metadata stitching

### `remove`

`remove.ts` will stop scanning directories directly to discover removable skills.

Instead it will:

- select candidate records from Installed View
- match requested skill references against `packageName` first
- then match against `aliases`
- then match against display name as compatibility fallback
- use `canonicalPath` and `agentLinks` to determine cleanup targets

This keeps removal behavior centered on **Package Name** while preserving compatibility for historical installs.

### `update`

`update.ts` will stop inferring registry entries by re-reading installed `package.json` files as a primary recovery strategy.

It will update only records that satisfy:

- `packageName` exists
- `sourceType === 'registry'`
- `orphaned === false`

For orphaned registry installs, `update` should report that local installed state must be repaired before update resolution can proceed.

This is a deliberate tightening of behavior. It removes ambiguous guessing and keeps update resolution inside a trustworthy seam.

## Relationship To Existing Modules

### `installer.ts`

`installer.ts` currently mixes:

- installation mechanics
- path helpers
- installed-state listing

After this refactor:

- path and installation helpers remain in `installer.ts`
- installed-state listing logic moves to `installed-view.ts`
- `listInstalledSkills` should either be deleted or reduced to a thin compatibility wrapper over the new seam

### `skill-lock.ts` and `local-lock.ts`

These stay as persistence adapters in this iteration.

The new module may read them both, but callers should stop branching on global vs project lock shape whenever possible.

This refactor does not yet unify the lock adapters. That remains a later deepening opportunity.

## User-Visible Behavior Changes

This design intentionally allows output cleanup to gain a better interface.

The CLI should become more explicit about four local states:

- tracked registry install
- tracked source/local install
- untracked disk skill
- orphaned tracked install

That clarity is preferable to the current behavior where commands silently reinterpret partial state differently.

## Testing Strategy

New tests should target the **Installed View module** interface, not just the commands.

Minimum coverage:

1. project canonical install with matching lock entry
2. global canonical install with matching lock entry
3. only agent-specific directory exists and canonical path is missing
4. lock entry exists but disk skill is missing
5. `installName` differs from display name and aliases still resolve correctly
6. project and global installs for the same package both project correctly by scope
7. registry install and source install coexist without identity confusion
8. alias collision produces orphaned projection

Command-level regression tests should then verify:

- `list` prints orphan markers
- `remove` prefers `Package Name` matching
- `update` skips orphaned registry installs and explains why

## Implementation Plan Shape

Implementation should proceed in this order:

1. add `installed-view.ts` and its tests
2. migrate `list.ts` to the new seam
3. migrate `remove.ts` to the new seam
4. migrate `update.ts` to the new seam
5. retire or thin `installer.ts:listInstalledSkills`

This sequence keeps the new seam stable before destructive behaviors depend on it.

## Risks

- historical installs may have surprising alias patterns not captured by current tests
- explicit orphan labeling may change snapshots and user expectations
- `remove` behavior may expose latent inconsistencies that were previously hidden by directory rescans

These are acceptable risks because the refactor is intended to make installed state more truthful, not more forgiving through hidden heuristics.

## Success Criteria

This refactor is successful when:

- one module owns the **Installed View** seam
- `list`, `remove`, and `update` no longer reconstruct installed state independently
- `Package Name` becomes the primary identity in remove and update behavior
- orphaned local state is surfaced explicitly
- tests for installed-state projection concentrate around one interface
