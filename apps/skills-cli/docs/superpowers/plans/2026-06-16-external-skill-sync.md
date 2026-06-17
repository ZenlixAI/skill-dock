# External Skill Sync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `skills sync` command that replays external-source installs (`local`, `github`, `git`) and refreshes the installed skill content without changing the publish-oriented meaning of `import-local` / `import-git`.

**Architecture:** Keep `sync` separate from `update` and from publish flows. Add a thin CLI command in `src/sync.ts`, a focused source-replay helper in `src/sync-source.ts`, and a workflow orchestrator in `src/sync-workflow.ts` that selects installed records, re-acquires the source, re-identifies the original skill via `legacy.skillPath`, reinstalls it, and refreshes the lock entry.

**Tech Stack:** TypeScript, Vitest, existing installer/lock/installed-view helpers, `@clack/prompts`, existing git/source parser utilities.

---

## File Structure

- Create: `src/sync.ts`
  Command parsing, scope selection, user-facing output, workflow invocation.
- Create: `src/sync.test.ts`
  Command-surface tests: option parsing, help text, success/failure output.
- Create: `src/sync-source.ts`
  Replay helpers that turn lock metadata back into a source + discovered skill match.
- Create: `src/sync-source.test.ts`
  Deterministic unit tests for legacy source replay and `skillPath` matching.
- Create: `src/sync-workflow.ts`
  Installed-view filtering, per-record sync execution, lock refresh, cleanup, summary.
- Create: `src/sync-workflow.test.ts`
  Workflow tests covering success, skips, replayability errors, and git cleanup.
- Modify: `src/cli.ts`
  Register the `sync` command in global help, dedicated help, and command dispatch.
- Modify: `README.md`
  Document `skills sync` as the external-source refresh command and clarify that `import-*` remains publish-only.

## Task 1: Add the `skills sync` CLI Surface

**Files:**
- Create: `src/sync.ts`
- Test: `src/sync.test.ts`
- Modify: `src/cli.ts`

- [ ] **Step 1: Write the failing command tests**

```ts
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('./sync-workflow.ts', () => ({
  runSyncWorkflow: vi.fn(),
}));

import { parseSyncOptions, runSync } from './sync.ts';
import { runSyncWorkflow } from './sync-workflow.ts';
import { runCliOutput, stripLogo } from './test-utils.ts';

describe('sync command', () => {
  let logSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    vi.mocked(runSyncWorkflow).mockReset();
    process.exitCode = undefined;
  });

  afterEach(() => {
    logSpy.mockRestore();
    vi.restoreAllMocks();
    process.exitCode = undefined;
  });

  it('parses explicit sync options', () => {
    expect(parseSyncOptions(['foo', 'bar', '--global', '--yes'])).toEqual({
      skills: ['foo', 'bar'],
      global: true,
      yes: true,
    });
  });

  it('prints success output from the sync workflow result', async () => {
    vi.mocked(runSyncWorkflow).mockResolvedValueOnce({
      successCount: 1,
      failCount: 0,
      skippedCount: 1,
      failures: [],
      skipped: ['@acme.skill/registry-only'],
    });

    await runSync(['legacy-skill', '--project']);

    expect(runSyncWorkflow).toHaveBeenCalledWith({
      project: true,
      skills: ['legacy-skill'],
    });
    expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('Checking source-installed skills'));
    expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('✓ Synced 1 skill(s)'));
  });

  it('prints failure output when sync workflow reports failures', async () => {
    vi.mocked(runSyncWorkflow).mockResolvedValueOnce({
      successCount: 0,
      failCount: 1,
      skippedCount: 0,
      failures: [{ skill: 'legacy-skill', error: 'legacy.source is required for sync' }],
      skipped: [],
    });

    await runSync(['legacy-skill']);

    expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('Failed to sync 1 skill(s)'));
    expect(process.exitCode).toBe(1);
  });

  it('shows dedicated sync help output', () => {
    const output = stripLogo(runCliOutput(['sync', '--help']));
    expect(output).toContain('Usage: skills sync [skills...] [options]');
    expect(output).toContain('Refresh installed skills that came from local paths or git sources.');
    expect(output).toContain('--global');
    expect(output).toContain('--project');
  });
});
```

- [ ] **Step 2: Run the new command tests and confirm they fail**

Run: `pnpm test src/sync.test.ts`

Expected: FAIL with module-not-found or missing export errors for `src/sync.ts` / `runSyncWorkflow`, plus missing help text in `src/cli.ts`.

- [ ] **Step 3: Implement the minimal command module and CLI wiring**

```ts
// src/sync.ts
import { runSyncWorkflow } from './sync-workflow.ts';

export interface SyncOptions {
  global?: boolean;
  project?: boolean;
  yes?: boolean;
  skills?: string[];
}

export function parseSyncOptions(args: string[]): SyncOptions {
  const options: SyncOptions = {};
  const positional: string[] = [];

  for (const arg of args) {
    if (arg === '-g' || arg === '--global') {
      options.global = true;
    } else if (arg === '-p' || arg === '--project') {
      options.project = true;
    } else if (arg === '-y' || arg === '--yes') {
      options.yes = true;
    } else if (!arg.startsWith('-')) {
      positional.push(arg);
    }
  }

  if (positional.length > 0) {
    options.skills = positional;
  }

  return options;
}

export async function runSync(args: string[] = []): Promise<void> {
  const options = parseSyncOptions(args);

  console.log('Checking source-installed skills...');
  console.log();

  const result = await runSyncWorkflow(options);

  if (result.successCount > 0) {
    console.log(`✓ Synced ${result.successCount} skill(s)`);
  }
  if (result.skippedCount > 0) {
    console.log(`Skipped ${result.skippedCount} skill(s)`);
  }
  if (result.failCount > 0) {
    console.log(`Failed to sync ${result.failCount} skill(s)`);
    for (const failure of result.failures) {
      console.log(`- ${failure.skill}: ${failure.error}`);
    }
    process.exitCode = 1;
  }
}
```

```ts
// src/cli.ts additions
import { runSync } from './sync.ts';

// in showHelp()
//   sync [skills...]     Refresh installed skills that came from local paths or git sources

function showSyncHelp(): void {
  console.log(`
${BOLD}Usage:${RESET} skills sync [skills...] [options]

${BOLD}Description:${RESET}
  Refresh installed skills that came from local paths or git sources.

${BOLD}Options:${RESET}
  -g, --global           Sync global skills only
  -p, --project          Sync project skills only
  -y, --yes              Skip scope prompt
`);
}

// in command dispatch
case 'sync':
  if (rest.includes('--help') || rest.includes('-h')) {
    showSyncHelp();
    return;
  }
  await runSync(rest);
  return;
```

- [ ] **Step 4: Run the command tests again**

Run: `pnpm test src/sync.test.ts`

Expected: PASS with the new parse/help/output behavior green.

- [ ] **Step 5: Commit the CLI surface**

```bash
git add src/sync.ts src/sync.test.ts src/cli.ts
git commit -m "feat: add sync command surface"
```

## Task 2: Add Source Replay Helpers for Legacy External Installs

**Files:**
- Create: `src/sync-source.ts`
- Test: `src/sync-source.test.ts`

- [ ] **Step 1: Write failing replay-helper tests**

```ts
import { describe, expect, it } from 'vitest';
import type { Skill } from './types.ts';
import { parseSourceInput } from './source-parser.ts';
import {
  getRecordedSkillPath,
  parseLegacyReplaySource,
  selectDiscoveredSkillForSync,
} from './sync-source.ts';

describe('sync source helpers', () => {
  it('replays a github legacy source with its ref', () => {
    const parsed = parseLegacyReplaySource({
      sourceType: 'github',
      legacy: {
        source: 'https://github.com/acme/skills.git',
        ref: 'main',
      },
    });

    expect(parsed).toMatchObject(parseSourceInput('https://github.com/acme/skills.git')!);
    expect(parsed?.ref).toBe('main');
  });

  it('returns the recorded SKILL.md path from legacy metadata', () => {
    expect(
      getRecordedSkillPath({
        legacy: { skillPath: 'skills/legacy-skill/SKILL.md' },
      })
    ).toBe('skills/legacy-skill/SKILL.md');
  });

  it('selects the discovered skill whose SKILL.md matches legacy.skillPath', () => {
    const discovered: Skill[] = [
      { name: 'first', description: 'First', path: '/repo/skills/first' },
      { name: 'second', description: 'Second', path: '/repo/skills/second' },
    ];

    expect(
      selectDiscoveredSkillForSync(discovered, '/repo', 'skills/second/SKILL.md')
    )?.toMatchObject({ name: 'second', path: '/repo/skills/second' });
  });

  it('throws when legacy.skillPath does not match any discovered skill', () => {
    expect(() =>
      selectDiscoveredSkillForSync(
        [{ name: 'first', description: 'First', path: '/repo/skills/first' }],
        '/repo',
        'skills/missing/SKILL.md'
      )
    ).toThrow('Recorded skill path no longer resolves to a discovered skill');
  });
});
```

- [ ] **Step 2: Run the helper tests and verify they fail**

Run: `pnpm test src/sync-source.test.ts`

Expected: FAIL because `src/sync-source.ts` does not exist yet.

- [ ] **Step 3: Implement the replay helper module**

```ts
// src/sync-source.ts
import { join, relative } from 'path';
import { parseSourceInput } from './source-parser.ts';
import type { Skill, ParsedSource } from './types.ts';

interface SyncLegacyShape {
  sourceType?: string;
  legacy?: {
    source?: string;
    ref?: string;
    skillPath?: string;
  };
}

export function getRecordedSkillPath(input: SyncLegacyShape): string | null {
  const raw = input.legacy?.skillPath?.trim();
  return raw ? raw.replace(/\\/g, '/') : null;
}

export function parseLegacyReplaySource(input: SyncLegacyShape): ParsedSource | null {
  const rawSource = input.legacy?.source?.trim();
  if (!rawSource) {
    return null;
  }

  const parsed = parseSourceInput(rawSource);
  if (!parsed) {
    return null;
  }

  return {
    ...parsed,
    ref: input.legacy?.ref?.trim() || parsed.ref,
  };
}

export function selectDiscoveredSkillForSync(
  discoveredSkills: Skill[],
  searchRoot: string,
  recordedSkillPath: string
): Skill {
  const normalizedTarget = recordedSkillPath.replace(/\\/g, '/');
  const matches = discoveredSkills.filter((skill) => {
    const skillMdPath = join(skill.path, 'SKILL.md');
    return relative(searchRoot, skillMdPath).replace(/\\/g, '/') === normalizedTarget;
  });

  if (matches.length === 1) {
    return matches[0]!;
  }
  if (matches.length === 0) {
    throw new Error('Recorded skill path no longer resolves to a discovered skill.');
  }
  throw new Error('Recorded skill path resolved to multiple discovered skills.');
}
```

- [ ] **Step 4: Run the helper tests again**

Run: `pnpm test src/sync-source.test.ts`

Expected: PASS with deterministic replay/path matching behavior.

- [ ] **Step 5: Commit the replay helper**

```bash
git add src/sync-source.ts src/sync-source.test.ts
git commit -m "feat: add sync source replay helpers"
```

## Task 3: Implement the Sync Workflow

**Files:**
- Create: `src/sync-workflow.ts`
- Test: `src/sync-workflow.test.ts`
- Modify: `src/sync.ts`

- [ ] **Step 1: Write the failing workflow tests**

```ts
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('./installed-view.ts', () => ({
  listInstalledView: vi.fn(),
}));
vi.mock('./git.ts', () => ({
  cloneRepo: vi.fn(),
  cleanupTempDir: vi.fn(() => Promise.resolve()),
}));
vi.mock('./skills.ts', () => ({
  discoverSkills: vi.fn(),
}));
vi.mock('./installer.ts', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./installer.ts')>();
  return {
    ...actual,
    installSkillForAgent: vi.fn(),
  };
});
vi.mock('./local-lock.ts', () => ({
  addSkillToLocalLock: vi.fn(),
  computeSkillFolderHash: vi.fn(),
}));
vi.mock('./skill-lock.ts', () => ({
  addSkillToLock: vi.fn(),
}));

import { runSyncWorkflow } from './sync-workflow.ts';
import { listInstalledView } from './installed-view.ts';
import { cloneRepo, cleanupTempDir } from './git.ts';
import { discoverSkills } from './skills.ts';
import { installSkillForAgent } from './installer.ts';
import { addSkillToLocalLock, computeSkillFolderHash } from './local-lock.ts';
import { addSkillToLock } from './skill-lock.ts';

describe('sync workflow', () => {
  beforeEach(() => {
    vi.mocked(listInstalledView).mockReset();
    vi.mocked(cloneRepo).mockReset();
    vi.mocked(cleanupTempDir).mockClear();
    vi.mocked(discoverSkills).mockReset();
    vi.mocked(installSkillForAgent).mockReset();
    vi.mocked(addSkillToLocalLock).mockReset();
    vi.mocked(computeSkillFolderHash).mockReset();
    vi.mocked(addSkillToLock).mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('replays a project local-source install and refreshes the local lock entry', async () => {
    vi.mocked(listInstalledView).mockResolvedValueOnce([
      {
        lockKey: 'legacy-skill',
        packageName: undefined,
        displayName: 'Legacy Skill',
        installName: 'legacy-skill',
        aliases: ['legacy-skill'],
        scope: 'project',
        canonicalPath: '/repo/.agents/skills/legacy-skill',
        agentLinks: [{ agent: 'codex', path: '/repo/.agents/skills/legacy-skill', kind: 'canonical' }],
        sourceType: 'local',
        registry: undefined,
        version: undefined,
        pluginName: undefined,
        orphaned: false,
        orphanReasons: [],
        lockEntry: {
          sourceType: 'local',
          computedHash: 'old-hash',
          legacy: {
            source: '/tmp/source-skill',
            skillPath: 'SKILL.md',
          },
        },
      },
    ]);
    vi.mocked(discoverSkills).mockResolvedValueOnce([
      {
        name: 'Legacy Skill',
        description: 'Updated',
        path: '/tmp/source-skill',
      },
    ]);
    vi.mocked(installSkillForAgent).mockResolvedValueOnce({
      success: true,
      path: '/repo/.agents/skills/legacy-skill',
      canonicalPath: '/repo/.agents/skills/legacy-skill',
      mode: 'symlink',
    });
    vi.mocked(computeSkillFolderHash).mockResolvedValueOnce('new-hash');

    const result = await runSyncWorkflow({ project: true, skills: ['legacy-skill'] });

    expect(discoverSkills).toHaveBeenCalledWith('/tmp/source-skill', undefined, { fullDepth: false });
    expect(installSkillForAgent).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'Legacy Skill', path: '/tmp/source-skill' }),
      'codex',
      expect.objectContaining({ global: false })
    );
    expect(addSkillToLocalLock).toHaveBeenCalledWith(
      'legacy-skill',
      expect.objectContaining({
        sourceType: 'local',
        computedHash: 'new-hash',
      }),
      expect.any(String)
    );
    expect(result.successCount).toBe(1);
  });

  it('skips orphaned or registry-installed records', async () => {
    vi.mocked(listInstalledView).mockResolvedValueOnce([
      {
        lockKey: '@acme.skill/pdf',
        packageName: '@acme.skill/pdf',
        displayName: '@acme.skill/pdf',
        installName: 'pdf',
        aliases: ['pdf', '@acme.skill/pdf'],
        scope: 'project',
        canonicalPath: '/repo/.agents/skills/pdf',
        agentLinks: [],
        sourceType: 'registry',
        registry: 'https://catalog.example.com',
        version: '1.0.0',
        pluginName: undefined,
        orphaned: false,
        orphanReasons: [],
        lockEntry: {
          sourceType: 'registry',
          computedHash: 'hash',
          packageName: '@acme.skill/pdf',
          version: '1.0.0',
        },
      },
      {
        lockKey: 'broken-skill',
        packageName: undefined,
        displayName: 'broken-skill',
        installName: 'broken-skill',
        aliases: ['broken-skill'],
        scope: 'project',
        canonicalPath: '/repo/.agents/skills/broken-skill',
        agentLinks: [],
        sourceType: 'git',
        registry: undefined,
        version: undefined,
        pluginName: undefined,
        orphaned: true,
        orphanReasons: ['missing_canonical_path'],
        lockEntry: {
          sourceType: 'git',
          computedHash: 'hash',
          legacy: {
            source: 'https://github.com/acme/skills.git',
            skillPath: 'skills/broken-skill/SKILL.md',
          },
        },
      },
    ]);

    const result = await runSyncWorkflow({ project: true });

    expect(result.successCount).toBe(0);
    expect(result.skippedCount).toBe(2);
    expect(result.failCount).toBe(0);
  });

  it('clones git sources and always cleans up temp directories', async () => {
    vi.mocked(listInstalledView).mockResolvedValueOnce([
      {
        lockKey: 'legacy-skill',
        packageName: undefined,
        displayName: 'Legacy Skill',
        installName: 'legacy-skill',
        aliases: ['legacy-skill'],
        scope: 'global',
        canonicalPath: '/Users/test/.agents/skills/legacy-skill',
        agentLinks: [{ agent: 'codex', path: '/Users/test/.agents/skills/legacy-skill', kind: 'canonical' }],
        sourceType: 'git',
        registry: undefined,
        version: undefined,
        pluginName: undefined,
        orphaned: false,
        orphanReasons: [],
        lockEntry: {
          sourceType: 'git',
          skillFolderHash: 'old-hash',
          legacy: {
            source: 'https://github.com/acme/skills.git',
            skillPath: 'skills/legacy-skill/SKILL.md',
          },
        },
      },
    ]);
    vi.mocked(cloneRepo).mockResolvedValueOnce('/tmp/cloned-skills');
    vi.mocked(discoverSkills).mockResolvedValueOnce([
      {
        name: 'Legacy Skill',
        description: 'Updated',
        path: '/tmp/cloned-skills/skills/legacy-skill',
      },
    ]);
    vi.mocked(installSkillForAgent).mockResolvedValueOnce({
      success: true,
      path: '/Users/test/.agents/skills/legacy-skill',
      canonicalPath: '/Users/test/.agents/skills/legacy-skill',
      mode: 'symlink',
    });
    vi.mocked(computeSkillFolderHash).mockResolvedValueOnce('global-hash');

    await runSyncWorkflow({ global: true, yes: true });

    expect(cloneRepo).toHaveBeenCalledWith('https://github.com/acme/skills.git', undefined);
    expect(addSkillToLock).toHaveBeenCalledWith(
      'legacy-skill',
      expect.objectContaining({
        sourceType: 'git',
        skillFolderHash: 'global-hash',
      })
    );
    expect(cleanupTempDir).toHaveBeenCalledWith('/tmp/cloned-skills');
  });
});
```

- [ ] **Step 2: Run the workflow tests and verify they fail**

Run: `pnpm test src/sync-workflow.test.ts`

Expected: FAIL because `src/sync-workflow.ts` does not exist yet and the command module still calls a stub.

- [ ] **Step 3: Implement the workflow orchestration**

```ts
// src/sync-workflow.ts
import { addSkillToLocalLock, computeSkillFolderHash } from './local-lock.ts';
import { cloneRepo, cleanupTempDir } from './git.ts';
import { listInstalledView, type InstalledSkillView } from './installed-view.ts';
import { installSkillForAgent } from './installer.ts';
import { addSkillToLock } from './skill-lock.ts';
import { discoverSkills } from './skills.ts';
import { parseLegacyReplaySource, getRecordedSkillPath, selectDiscoveredSkillForSync } from './sync-source.ts';
import type { SyncOptions } from './sync.ts';

export interface SyncWorkflowResult {
  successCount: number;
  failCount: number;
  skippedCount: number;
  failures: Array<{ skill: string; error: string }>;
  skipped: string[];
}

function recordMatchesFilter(record: InstalledSkillView, requested?: string[]): boolean {
  if (!requested || requested.length === 0) {
    return true;
  }
  const aliases = new Set(record.aliases.map((value) => value.toLowerCase()));
  if (record.packageName) {
    aliases.add(record.packageName.toLowerCase());
  }
  return requested.some((value) => aliases.has(value.toLowerCase()));
}

function syncableRecord(record: InstalledSkillView): boolean {
  return ['local', 'github', 'git'].includes(record.sourceType);
}

export async function runSyncWorkflow(options: SyncOptions): Promise<SyncWorkflowResult> {
  const scope = options.global ? 'global' : 'project';
  const records = await listInstalledView({ scope, cwd: process.cwd() });
  const selected = records.filter((record) => recordMatchesFilter(record, options.skills));

  const result: SyncWorkflowResult = {
    successCount: 0,
    failCount: 0,
    skippedCount: 0,
    failures: [],
    skipped: [],
  };

  for (const record of selected) {
    if (!syncableRecord(record) || record.orphaned) {
      result.skippedCount++;
      result.skipped.push(record.packageName || record.displayName);
      continue;
    }

    const parsedSource = parseLegacyReplaySource(record.lockEntry || {});
    const recordedSkillPath = getRecordedSkillPath(record.lockEntry || {});
    if (!parsedSource || !recordedSkillPath) {
      result.failCount++;
      result.failures.push({
        skill: record.packageName || record.displayName,
        error: 'legacy.source and legacy.skillPath are required for sync',
      });
      continue;
    }

    let cleanupRoot: string | null = null;

    try {
      const sourceRoot =
        parsedSource.type === 'local'
          ? parsedSource.localPath || parsedSource.url
          : await cloneRepo(parsedSource.url, parsedSource.ref);
      cleanupRoot = parsedSource.type === 'local' ? null : sourceRoot;
      const searchRoot = parsedSource.subpath ? `${sourceRoot}/${parsedSource.subpath}` : sourceRoot;
      const discoveredSkills = await discoverSkills(sourceRoot, parsedSource.subpath, { fullDepth: false });
      const skill = selectDiscoveredSkillForSync(discoveredSkills, searchRoot, recordedSkillPath);

      const targetAgents = record.agentLinks
        .filter((link) => link.kind !== 'missing')
        .map((link) => link.agent);
      const uniqueAgents = Array.from(new Set(targetAgents));

      if (uniqueAgents.length === 0) {
        throw new Error('Installed skill has no materialized agent targets to refresh.');
      }

      for (const agent of uniqueAgents) {
        const installResult = await installSkillForAgent(skill, agent, {
          global: scope === 'global',
          cwd: process.cwd(),
        });
        if (!installResult.success) {
          throw new Error(installResult.error || 'Failed to reinstall skill');
        }
      }

      if (scope === 'global') {
        const skillFolderHash = await computeSkillFolderHash(skill.path);
        await addSkillToLock(record.lockKey || record.installName, {
          ...(record.lockEntry as never),
          sourceType: record.sourceType,
          skillFolderHash,
        });
      } else {
        await addSkillToLocalLock(
          record.lockKey || record.installName,
          {
            ...(record.lockEntry as never),
            sourceType: record.sourceType,
            computedHash: await computeSkillFolderHash(skill.path),
          },
          process.cwd()
        );
      }

      result.successCount++;
    } catch (error) {
      result.failCount++;
      result.failures.push({
        skill: record.packageName || record.displayName,
        error: error instanceof Error ? error.message : String(error),
      });
    } finally {
      if (cleanupRoot) {
        await cleanupTempDir(cleanupRoot).catch(() => {});
      }
    }
  }

  return result;
}
```

- [ ] **Step 4: Run the workflow tests again**

Run: `pnpm test src/sync-workflow.test.ts src/sync.test.ts src/sync-source.test.ts`

Expected: PASS with project/local success, registry/orphan skips, and git cleanup all green.

- [ ] **Step 5: Commit the workflow**

```bash
git add src/sync-workflow.ts src/sync-workflow.test.ts src/sync.ts
git commit -m "feat: implement external skill sync workflow"
```

## Task 4: Document the New Lifecycle and Run Verification

**Files:**
- Modify: `README.md`
- Modify: `src/cli.ts`
- Test: `src/sync.test.ts`

- [ ] **Step 1: Write the README update**

```md
### `skills sync`

`skills sync` refreshes installed skills that originally came from a local path,
GitHub repository, or generic git URL. It replays the recorded external source,
re-discovers the original skill by its stored `skillPath`, and reinstalls it into
the current project or global scope.

Use `skills update` for registry-installed skills.
Use `skills import-local` / `skills import-git` to publish imported sources to the registry.
```

- [ ] **Step 2: Update the main CLI examples**

```ts
// showHelp() examples in src/cli.ts
${DIM}$${RESET} skills sync
${DIM}$${RESET} skills sync legacy-skill
${DIM}$${RESET} skills sync -g
```

- [ ] **Step 3: Run targeted tests**

Run: `pnpm test src/sync.test.ts src/sync-source.test.ts src/sync-workflow.test.ts`

Expected: PASS

- [ ] **Step 4: Run type-check and one broader regression slice**

Run: `pnpm type-check`

Expected: PASS

Run: `pnpm test src/import-source.test.ts src/import-source-workflow.test.ts tests/update.test.ts`

Expected: PASS so the new sync path does not break adjacent source-install and registry-update behavior.

- [ ] **Step 5: Commit docs and verification sweep**

```bash
git add README.md src/cli.ts docs/superpowers/plans/2026-06-16-external-skill-sync.md
git commit -m "docs: document external skill sync lifecycle"
```

## Self-Review

Spec coverage check:

- command boundary (`sync` separate from `import-*`): covered by Task 1 + Task 4
- replay existing external installs only: covered by Task 2 + Task 3
- reuse `legacy.source/ref/skillPath`: covered by Task 2 + Task 3
- skip registry installs: covered by Task 3
- fail clearly on missing replay metadata and changed paths: covered by Task 2 + Task 3
- document lifecycle split against `update` and `import-*`: covered by Task 4

Placeholder scan:

- No `TODO`, `TBD`, or “similar to Task N” placeholders remain.

Type consistency:

- `SyncOptions`, `SyncWorkflowResult`, `parseLegacyReplaySource`, `getRecordedSkillPath`, and `selectDiscoveredSkillForSync` are named consistently across tasks.

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-06-16-external-skill-sync.md`. Two execution options:

1. Subagent-Driven (recommended) - I dispatch a fresh subagent per task, review between tasks, fast iteration

2. Inline Execution - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
