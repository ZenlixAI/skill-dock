# skills-cli Installed View Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce a deep Installed View module for `skills-cli` and migrate `list`, `remove`, and `update` to consume it instead of reconstructing installed state independently.

**Architecture:** Add a new `installed-view.ts` module that performs a two-phase collect/project pass over canonical skill directories, agent-specific directories, and lock adapters. Keep install mechanics in `installer.ts`, but move installed-state projection behind a new seam and make command modules consume `InstalledSkillView` records.

**Tech Stack:** TypeScript, Node.js filesystem APIs, Vitest, existing `skills-cli` command/test surface

---

## File Map

- Create: `apps/skills-cli/src/installed-view.ts`
  - Own the Installed View seam
  - Collect raw filesystem and lock facts
  - Project `InstalledSkillView` records with aliases, agent links, and orphan state
- Create: `apps/skills-cli/src/installed-view.test.ts`
  - Add direct module tests for projection rules and orphan cases
- Modify: `apps/skills-cli/src/list.ts`
  - Consume Installed View instead of rebuilding metadata maps and alias fallbacks
- Modify: `apps/skills-cli/src/list.test.ts`
  - Assert orphan markers and updated list projection behavior
- Modify: `apps/skills-cli/src/remove.ts`
  - Stop scanning directories directly
  - Match selected skills through Installed View records
- Modify: `apps/skills-cli/src/remove.test.ts`
  - Assert `Package Name` priority and alias-based removal through the new seam
- Modify: `apps/skills-cli/src/update.ts`
  - Stop inferring registry records from disk `package.json` as the primary path
  - Update only non-orphaned registry Installed View records
- Modify: `apps/skills-cli/tests/update.test.ts`
  - Assert orphaned registry installs are skipped with an explicit message
- Modify: `apps/skills-cli/src/installer.ts`
  - Delete or thin `listInstalledSkills()` into a compatibility wrapper over the new module

### Task 1: Add Failing Installed View Projection Tests

**Files:**
- Create: `apps/skills-cli/src/installed-view.test.ts`
- Test: `apps/skills-cli/src/installed-view.test.ts`

- [ ] **Step 1: Write a failing test for a tracked project registry install**

```ts
it('projects a tracked project registry install with aliases and agent links', async () => {
  const root = mkdtempSync(join(tmpdir(), 'installed-view-project-'));
  mkdirSync(join(root, '.agents', 'skills', 'pdf-skill'), { recursive: true });
  writeFileSync(
    join(root, '.agents', 'skills', 'pdf-skill', 'SKILL.md'),
    `---\nname: PDF Skill\ndescription: Read PDFs\n---\n`
  );
  writeFileSync(
    join(root, 'skills-lock.json'),
    JSON.stringify(
      {
        version: 1,
        skills: {
          '@platform.skill/pdf-skill': {
            sourceType: 'registry',
            computedHash: 'abc123',
            packageName: '@platform.skill/pdf-skill',
            installName: 'pdf-skill',
            version: '1.2.3',
            registry: 'https://catalog.example.com',
          },
        },
      },
      null,
      2
    )
  );
  mkdirSync(join(root, '.cursor', 'rules', 'pdf-skill'), { recursive: true });

  const records = await listInstalledView({ scope: 'project', cwd: root });

  expect(records).toEqual(
    expect.arrayContaining([
      expect.objectContaining({
        packageName: '@platform.skill/pdf-skill',
        installName: 'pdf-skill',
        aliases: expect.arrayContaining(['@platform.skill/pdf-skill', 'pdf-skill']),
        scope: 'project',
        orphaned: false,
      }),
    ])
  );
});
```

- [ ] **Step 2: Add failing tests for orphaned states**

```ts
it('marks a skill orphaned when only the agent-specific directory exists', async () => {
  const root = mkdtempSync(join(tmpdir(), 'installed-view-agent-only-'));
  mkdirSync(join(root, '.cursor', 'rules', 'pdf-skill'), { recursive: true });
  writeFileSync(
    join(root, '.cursor', 'rules', 'pdf-skill', 'SKILL.md'),
    `---\nname: PDF Skill\ndescription: Read PDFs\n---\n`
  );

  const records = await listInstalledView({ scope: 'project', cwd: root });

  expect(records[0]).toMatchObject({
    displayName: 'PDF Skill',
    orphaned: true,
    orphanReasons: expect.arrayContaining(['missing_canonical_path']),
  });
});

it('marks a lock-only registry entry orphaned when the canonical path is missing', async () => {
  const root = mkdtempSync(join(tmpdir(), 'installed-view-lock-only-'));
  writeFileSync(
    join(root, 'skills-lock.json'),
    JSON.stringify(
      {
        version: 1,
        skills: {
          '@platform.skill/pdf-skill': {
            sourceType: 'registry',
            computedHash: 'abc123',
            packageName: '@platform.skill/pdf-skill',
            installName: 'pdf-skill',
            version: '1.2.3',
          },
        },
      },
      null,
      2
    )
  );

  const records = await listInstalledView({ scope: 'project', cwd: root });

  expect(records[0]).toMatchObject({
    packageName: '@platform.skill/pdf-skill',
    orphaned: true,
    orphanReasons: expect.arrayContaining(['missing_canonical_path']),
  });
});
```

- [ ] **Step 3: Add a failing alias-collision test**

```ts
it('marks conflicting install aliases as orphaned', async () => {
  const root = mkdtempSync(join(tmpdir(), 'installed-view-conflict-'));

  for (const dir of ['pdf-skill', 'pdf.skill']) {
    mkdirSync(join(root, '.agents', 'skills', dir), { recursive: true });
    writeFileSync(
      join(root, '.agents', 'skills', dir, 'SKILL.md'),
      `---\nname: ${dir}\ndescription: collision\n---\n`
    );
  }

  const records = await listInstalledView({ scope: 'project', cwd: root });

  expect(records.some((record) => record.orphanReasons.includes('alias_conflict'))).toBe(true);
});
```

- [ ] **Step 4: Run the new test file to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-cli && npm test -- src/installed-view.test.ts
```

Expected: FAIL because `installed-view.ts` and `listInstalledView()` do not exist yet.

- [ ] **Step 5: Commit the failing tests**

```bash
git add apps/skills-cli/src/installed-view.test.ts
git commit -m "test: add installed view projection coverage"
```

### Task 2: Implement the Installed View Module

**Files:**
- Create: `apps/skills-cli/src/installed-view.ts`
- Modify: `apps/skills-cli/src/installer.ts`
- Test: `apps/skills-cli/src/installed-view.test.ts`

- [ ] **Step 1: Add the core types and public seam**

Create `apps/skills-cli/src/installed-view.ts` with:

```ts
import { existsSync } from 'fs';
import { readdir, stat } from 'fs/promises';
import { join } from 'path';
import { agents, detectInstalledAgents, isUniversalAgent } from './agents.ts';
import { getCanonicalSkillsDir, getAgentBaseDir, getInstalledDirectoryNames, sanitizeName } from './installer.ts';
import { readLocalLock, type LocalSkillLockEntry } from './local-lock.ts';
import { parseSkillMd } from './skills.ts';
import { getAllLockedSkills, type SkillLockEntry } from './skill-lock.ts';
import type { AgentType } from './types.ts';

export type InstalledViewScope = 'project' | 'global';
export type AgentLinkKind = 'canonical' | 'symlink' | 'copy' | 'missing';

export interface InstalledSkillView {
  packageName?: string;
  displayName: string;
  installName: string;
  aliases: string[];
  scope: InstalledViewScope;
  canonicalPath: string;
  agentLinks: Array<{ agent: AgentType; path: string; kind: AgentLinkKind }>;
  version?: string;
  sourceType: string;
  registry?: string;
  pluginName?: string;
  orphaned: boolean;
  orphanReasons: string[];
  lockEntry?: SkillLockEntry | LocalSkillLockEntry;
}

export interface ListInstalledViewOptions {
  scope?: InstalledViewScope;
  agentFilter?: AgentType[];
  cwd?: string;
}
```

- [ ] **Step 2: Implement the collect phase**

Add helpers:

```ts
type RawInstalledFact = {
  scope: InstalledViewScope;
  canonicalDirName: string;
  canonicalPath: string;
  parsedName?: string;
  parsedDescription?: string;
  discoveredFrom: Array<{ agent?: AgentType; path: string; kind: AgentLinkKind }>;
};

async function collectInstalledFacts(
  scope: InstalledViewScope,
  cwd: string,
  agentFilter?: AgentType[]
): Promise<{
  facts: RawInstalledFact[];
  lockEntries: Record<string, SkillLockEntry | LocalSkillLockEntry>;
}> {
  const canonicalDir = getCanonicalSkillsDir(scope === 'global', cwd);
  const lockEntries =
    scope === 'global' ? await getAllLockedSkills() : (await readLocalLock(cwd)).skills;
  const facts = new Map<string, RawInstalledFact>();
  const installedAgents = await detectInstalledAgents();
  const candidateAgents = agentFilter
    ? installedAgents.filter((agent) => agentFilter.includes(agent))
    : installedAgents;

  async function collectFromDirectory(path: string, agent?: AgentType) {
    if (!existsSync(path)) return;
    const entries = await readdir(path, { withFileTypes: true });
    for (const entry of entries) {
      if (!entry.isDirectory()) continue;
      const skillPath = join(path, entry.name);
      const skillMdPath = join(skillPath, 'SKILL.md');
      try {
        await stat(skillMdPath);
      } catch {
        continue;
      }
      const parsed = await parseSkillMd(skillMdPath);
      const key = `${scope}:${entry.name}`;
      const current =
        facts.get(key) ??
        {
          scope,
          canonicalDirName: entry.name,
          canonicalPath: join(canonicalDir, entry.name),
          parsedName: parsed?.name,
          parsedDescription: parsed?.description,
          discoveredFrom: [],
        };
      current.discoveredFrom.push({
        agent,
        path: skillPath,
        kind: agent ? 'symlink' : 'canonical',
      });
      facts.set(key, current);
    }
  }

  await collectFromDirectory(canonicalDir);
  for (const agent of candidateAgents) {
    await collectFromDirectory(getAgentBaseDir(agent, scope === 'global', cwd), agent);
  }

  return { facts: Array.from(facts.values()), lockEntries };
}
```

Collection rules:

- canonical scope path is always scanned first
- agent-specific directories are scanned for detected agents and existing on-disk agent dirs
- directories without `SKILL.md` are ignored

- [ ] **Step 3: Implement projection, alias generation, and orphan detection**

Add helpers:

```ts
function buildInstalledAliases(input: {
  packageName?: string;
  installName: string;
  displayName: string;
}): string[] {
  return Array.from(
    new Set(
      [
        input.packageName,
        input.installName,
        ...getInstalledDirectoryNames(input.packageName || input.installName, input.installName),
        sanitizeName(input.displayName),
      ].filter((value): value is string => typeof value === 'string' && value.trim().length > 0)
    )
  );
}

function detectOrphanReasons(input: {
  hasCanonicalPath: boolean;
  lockEntry?: SkillLockEntry | LocalSkillLockEntry;
  agentLinks: Array<{ agent: AgentType; path: string; kind: AgentLinkKind }>;
  aliasConflict: boolean;
}): string[] {
  const reasons: string[] = [];
  if (!input.hasCanonicalPath) reasons.push('missing_canonical_path');
  if (!input.lockEntry) reasons.push('missing_lock_entry');
  if (input.aliasConflict) reasons.push('alias_conflict');
  if (
    input.agentLinks.some((link) => link.kind !== 'missing') &&
    !input.hasCanonicalPath
  ) {
    reasons.push('agent_only_install');
  }
  return reasons;
}
```

Then export:

```ts
export async function listInstalledView(
  options: ListInstalledViewOptions = {}
): Promise<InstalledSkillView[]> {
  const cwd = options.cwd || process.cwd();
  const scopes: InstalledViewScope[] = options.scope ? [options.scope] : ['project', 'global'];
  const results: InstalledSkillView[] = [];

  for (const scope of scopes) {
    const { facts, lockEntries } = await collectInstalledFacts(scope, cwd, options.agentFilter);
    for (const fact of facts) {
      const lockEntry =
        Object.values(lockEntries).find(
          (entry) =>
            entry.installName === fact.canonicalDirName ||
            entry.packageName === fact.canonicalDirName
        ) ?? undefined;
      const displayName = fact.parsedName || lockEntry?.packageName || fact.canonicalDirName;
      const installName = lockEntry?.installName || fact.canonicalDirName;
      const aliases = buildInstalledAliases({
        packageName: lockEntry?.packageName,
        installName,
        displayName,
      });
      const orphanReasons = detectOrphanReasons({
        hasCanonicalPath: existsSync(fact.canonicalPath),
        lockEntry,
        agentLinks: [],
        aliasConflict: false,
      });

      results.push({
        packageName: lockEntry?.packageName,
        displayName,
        installName,
        aliases,
        scope,
        canonicalPath: fact.canonicalPath,
        agentLinks: [],
        version: lockEntry?.version,
        sourceType: lockEntry?.sourceType || 'local',
        registry: lockEntry?.registry,
        pluginName: 'pluginName' in (lockEntry || {}) ? lockEntry?.pluginName : undefined,
        orphaned: orphanReasons.length > 0,
        orphanReasons,
        lockEntry,
      });
    }
  }

  return results.sort((a, b) =>
    `${a.scope}:${a.packageName || a.displayName}`.localeCompare(
      `${b.scope}:${b.packageName || b.displayName}`
    )
  );
}
```

- [ ] **Step 4: Replace `installer.ts:listInstalledSkills()` with a compatibility wrapper**

Change `installer.ts` to:

```ts
import { listInstalledView } from './installed-view.ts';

export async function listInstalledSkills(
  options: { global?: boolean; cwd?: string; agentFilter?: AgentType[] } = {}
): Promise<InstalledSkill[]> {
  const scope =
    options.global === undefined ? undefined : options.global ? 'global' : 'project';
  const records = await listInstalledView({
    scope,
    cwd: options.cwd,
    agentFilter: options.agentFilter,
  });

  return records.map((record) => ({
    name: record.displayName,
    description: '',
    path: record.canonicalPath,
    canonicalPath: record.canonicalPath,
    scope: record.scope,
    agents: record.agentLinks
      .filter((link) => link.kind !== 'missing')
      .map((link) => link.agent),
  }));
}
```

- [ ] **Step 5: Run the new test file to verify GREEN**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-cli && npm test -- src/installed-view.test.ts
```

Expected: PASS with the new Installed View projection tests green.

- [ ] **Step 6: Commit the Installed View module**

```bash
git add apps/skills-cli/src/installed-view.ts apps/skills-cli/src/installed-view.test.ts apps/skills-cli/src/installer.ts
git commit -m "feat: add installed view projection module"
```

### Task 3: Migrate `list` to the Installed View Seam

**Files:**
- Modify: `apps/skills-cli/src/list.ts`
- Modify: `apps/skills-cli/src/list.test.ts`
- Test: `apps/skills-cli/src/list.test.ts`

- [ ] **Step 1: Add a failing list test for orphan markers**

Add to `apps/skills-cli/src/list.test.ts`:

```ts
it('marks orphaned installed skills in human-readable output', async () => {
  vi.mock('./installed-view.ts', () => ({
    listInstalledView: vi.fn(() =>
      Promise.resolve([
        {
          displayName: 'PDF Skill',
          installName: 'pdf-skill',
          aliases: ['@platform.skill/pdf-skill', 'pdf-skill'],
          scope: 'project',
          canonicalPath: '/tmp/project/.agents/skills/pdf-skill',
          agentLinks: [],
          sourceType: 'registry',
          packageName: '@platform.skill/pdf-skill',
          version: '1.2.3',
          orphaned: true,
          orphanReasons: ['missing_lock_entry'],
        },
      ])
    ),
  }));

  await runList([]);

  expect(stdout).toContain('(orphaned)');
});
```

- [ ] **Step 2: Run the list test file to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-cli && npm test -- src/list.test.ts
```

Expected: FAIL because `runList()` still uses the old metadata stitching path.

- [ ] **Step 3: Replace `runList()` input source with Installed View**

Update `apps/skills-cli/src/list.ts` so that:

```ts
import { listInstalledView, type InstalledSkillView } from './installed-view.ts';
```

and:

```ts
const installedSkills = await listInstalledView({
  scope: allScopes ? undefined : scope ? 'global' : 'project',
  agentFilter,
});
```

Use one formatter:

```ts
function formatDisplayName(record: InstalledSkillView): string {
  const name = sanitizeMetadata(record.packageName || record.displayName);
  return record.orphaned ? `${name} ${DIM}(orphaned)${RESET}` : name;
}
```

Delete:

- `getDisplayMetadataByScope()`
- `getDisplayMetadataForSkill()`
- direct lock lookups for display metadata

- [ ] **Step 4: Keep grouped output but drive it from projection fields**

Use:

```ts
const version = record.version ? ` ${DIM}v${record.version}${RESET}` : '';
const agentNames = record.agentLinks
  .filter((link) => link.kind !== 'missing')
  .map((link) => agents[link.agent].displayName);
```

Group by:

```ts
const groupName = record.pluginName || 'General';
```

- [ ] **Step 5: Run the list tests to verify GREEN**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-cli && npm test -- src/list.test.ts
```

Expected: PASS with orphan marker coverage and no regressions in grouping/output behavior.

- [ ] **Step 6: Commit the list migration**

```bash
git add apps/skills-cli/src/list.ts apps/skills-cli/src/list.test.ts
git commit -m "refactor: drive list command from installed view"
```

### Task 4: Migrate `remove` to the Installed View Seam

**Files:**
- Modify: `apps/skills-cli/src/remove.ts`
- Modify: `apps/skills-cli/src/remove.test.ts`
- Test: `apps/skills-cli/src/remove.test.ts`

- [ ] **Step 1: Add a failing remove test for `Package Name` priority**

Add to `apps/skills-cli/src/remove.test.ts`:

```ts
it('matches requested skills by package name before display-name aliases', async () => {
  vi.mock('./installed-view.ts', () => ({
    listInstalledView: vi.fn(() =>
      Promise.resolve([
        {
          packageName: '@platform.skill/pdf-skill',
          displayName: 'PDF Skill',
          installName: 'pdf-skill',
          aliases: ['@platform.skill/pdf-skill', 'pdf-skill', 'pdf-skill-display'],
          scope: 'project',
          canonicalPath: '/tmp/project/.agents/skills/pdf-skill',
          agentLinks: [
            {
              agent: 'cursor',
              path: '/tmp/project/.cursor/rules/pdf-skill',
              kind: 'symlink',
            },
          ],
          sourceType: 'registry',
          version: '1.2.3',
          orphaned: false,
          orphanReasons: [],
        },
      ])
    ),
  }));

  await removeCommand(['@platform.skill/pdf-skill'], { yes: true });

  expect(rmMock).toHaveBeenCalledWith(
    '/tmp/project/.agents/skills/pdf-skill',
    expect.objectContaining({ recursive: true, force: true })
  );
});
```

- [ ] **Step 2: Run the remove tests to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-cli && npm test -- src/remove.test.ts
```

Expected: FAIL because `removeCommand()` still scans directories directly.

- [ ] **Step 3: Replace the scan-and-map logic with Installed View selection**

In `apps/skills-cli/src/remove.ts`, import:

```ts
import { listInstalledView, type InstalledSkillView } from './installed-view.ts';
```

Replace the scan section with:

```ts
const records = await listInstalledView({
  scope: isGlobal ? 'global' : 'project',
  cwd,
});
```

Select records with:

```ts
function matchesRequestedRecord(record: InstalledSkillView, requested: string): boolean {
  if (record.packageName?.toLowerCase() === requested.toLowerCase()) return true;
  if (record.aliases.some((alias) => alias.toLowerCase() === requested.toLowerCase())) return true;
  return record.displayName.toLowerCase() === requested.toLowerCase();
}
```

- [ ] **Step 4: Remove via `canonicalPath` and `agentLinks`**

Use:

```ts
const pathsToCleanup = new Set<string>([
  record.canonicalPath,
  ...record.agentLinks
    .filter((link) => targetAgents.includes(link.agent) && link.kind !== 'missing')
    .map((link) => link.path),
]);
```

When `record.orphaned` is true, remove the visible paths but keep the final log message explicit:

```ts
source: record.packageName || record.displayName,
sourceType: record.sourceType,
```

- [ ] **Step 5: Run the remove tests to verify GREEN**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-cli && npm test -- src/remove.test.ts
```

Expected: PASS with `Package Name`-first matching and no regression in interactive/non-interactive removal cases.

- [ ] **Step 6: Commit the remove migration**

```bash
git add apps/skills-cli/src/remove.ts apps/skills-cli/src/remove.test.ts
git commit -m "refactor: drive remove command from installed view"
```

### Task 5: Migrate `update` to the Installed View Seam

**Files:**
- Modify: `apps/skills-cli/src/update.ts`
- Modify: `apps/skills-cli/tests/update.test.ts`
- Test: `apps/skills-cli/tests/update.test.ts`

- [ ] **Step 1: Add a failing update test for orphaned registry installs**

Add to `apps/skills-cli/tests/update.test.ts`:

```ts
it('skips orphaned registry installs and tells the user to repair local state first', async () => {
  vi.mock('../src/installed-view.ts', () => ({
    listInstalledView: vi.fn(() =>
      Promise.resolve([
        {
          packageName: '@platform.skill/pdf-skill',
          displayName: 'PDF Skill',
          installName: 'pdf-skill',
          aliases: ['@platform.skill/pdf-skill', 'pdf-skill'],
          scope: 'global',
          canonicalPath: '/tmp/home/.agents/skills/pdf-skill',
          agentLinks: [],
          sourceType: 'registry',
          version: '1.2.3',
          registry: 'https://catalog.example.com',
          orphaned: true,
          orphanReasons: ['missing_lock_entry'],
        },
      ])
    ),
  }));

  await updateGlobalSkills();

  expect(stdout).toContain('repair local installed state');
  expect(resolveInstallTarget).not.toHaveBeenCalled();
});
```

- [ ] **Step 2: Run the update tests to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-cli && npm test -- tests/update.test.ts
```

Expected: FAIL because `update.ts` still infers registry entries through lock scans and disk reads.

- [ ] **Step 3: Replace `getProjectSkillsForUpdate()` and global inference with Installed View**

Import:

```ts
import { listInstalledView, type InstalledSkillView } from './installed-view.ts';
```

Add a selector:

```ts
function getUpdatableRegistryRecords(
  records: InstalledSkillView[],
  skillFilter?: string[]
): InstalledSkillView[] {
  return records.filter((record) => {
    if (!record.packageName) return false;
    if (record.sourceType !== 'registry') return false;
    if (record.orphaned) return false;
    if (!skillFilter || skillFilter.length === 0) return true;
    return (
      skillFilter.some((value) => value.toLowerCase() === record.packageName?.toLowerCase()) ||
      skillFilter.some((value) => record.aliases.some((alias) => alias.toLowerCase() === value.toLowerCase()))
    );
  });
}
```

Delete:

- `inferUnlockedGlobalRegistryEntries()`
- direct `package.json` inference path

- [ ] **Step 4: Emit explicit skip messages for orphaned registry installs**

When a record is orphaned and registry-backed, log:

```ts
console.log(
  `${DIM}Skipping ${sanitizeMetadata(record.packageName || record.displayName)}:${RESET} repair local installed state before update`
);
```

Use `record.packageName`, `record.version`, and `record.registry` directly when calling `resolveInstallTarget()` and `runRegistryInstallWorkflow()`.

- [ ] **Step 5: Run the update tests to verify GREEN**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-cli && npm test -- tests/update.test.ts
```

Expected: PASS with orphaned registry installs skipped and normal registry installs still updating.

- [ ] **Step 6: Commit the update migration**

```bash
git add apps/skills-cli/src/update.ts apps/skills-cli/tests/update.test.ts
git commit -m "refactor: drive update command from installed view"
```

### Task 6: Full CLI Verification and Cleanup

**Files:**
- Modify: `apps/skills-cli/src/installer.ts`
- Test: `apps/skills-cli/src/installed-view.test.ts`
- Test: `apps/skills-cli/src/list.test.ts`
- Test: `apps/skills-cli/src/remove.test.ts`
- Test: `apps/skills-cli/tests/update.test.ts`

- [ ] **Step 1: Remove dead helper code left behind by the migration**

Delete any now-unused helpers such as:

```ts
// remove from list.ts
getDisplayMetadataByScope
getDisplayMetadataForSkill

// remove from update.ts
inferUnlockedGlobalRegistryEntries

// remove from remove.ts
scanDir
displayNameBySanitized
```

If `installer.ts:listInstalledSkills()` is still unused after the command migrations, delete it entirely and remove the `InstalledSkill` compatibility type.

- [ ] **Step 2: Run the focused CLI test suite**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-cli && npm test -- src/installed-view.test.ts src/list.test.ts src/remove.test.ts tests/update.test.ts
```

Expected: PASS with `0 failed`.

- [ ] **Step 3: Run the broader fast CLI suite**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-cli && npm test
```

Expected: PASS with the full `skills-cli` test suite green.

- [ ] **Step 4: Run type-check**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-cli && npm run type-check
```

Expected: PASS with no TypeScript errors.

- [ ] **Step 5: Commit the cleanup and verified refactor**

```bash
git add apps/skills-cli/src/installed-view.ts apps/skills-cli/src/installed-view.test.ts apps/skills-cli/src/list.ts apps/skills-cli/src/list.test.ts apps/skills-cli/src/remove.ts apps/skills-cli/src/remove.test.ts apps/skills-cli/src/update.ts apps/skills-cli/tests/update.test.ts apps/skills-cli/src/installer.ts
git commit -m "refactor: centralize skills-cli installed state projection"
```

## Self-Review

- Spec coverage:
  - Installed View seam: covered by Tasks 1-2
  - `list` migration: covered by Task 3
  - `remove` migration: covered by Task 4
  - `update` migration and orphan skip behavior: covered by Task 5
  - cleanup and final verification: covered by Task 6
- Placeholder scan:
  - no deferred implementation placeholders remain
  - each code-changing step contains concrete code snippets or exact removals
- Type consistency:
  - `InstalledSkillView`, `InstalledViewScope`, `AgentLinkKind`, and `listInstalledView()` names are used consistently across tasks
