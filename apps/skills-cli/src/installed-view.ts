import { existsSync, lstatSync } from 'fs';
import { readdir, stat } from 'fs/promises';
import { join } from 'path';
import { agents, detectInstalledAgents, isUniversalAgent } from './agents.ts';
import {
  deriveInstallName,
  getAgentBaseDir,
  getCanonicalSkillsDir,
  getInstalledDirectoryNames,
  sanitizeName,
} from './installer.ts';
import { readLocalLock, type LocalSkillLockEntry } from './local-lock.ts';
import { parseSkillMd } from './skills.ts';
import { getAllLockedSkills, type SkillLockEntry } from './skill-lock.ts';
import type { AgentType } from './types.ts';

export type InstalledViewScope = 'project' | 'global';
export type AgentLinkKind = 'canonical' | 'symlink' | 'copy' | 'missing';

export interface InstalledSkillView {
  lockKey?: string;
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

interface LockFact {
  key: string;
  entry: SkillLockEntry | LocalSkillLockEntry;
}

interface CanonicalFact {
  dirName: string;
  path: string;
  displayName: string;
  pluginName?: string;
}

interface AgentFact {
  agent: AgentType;
  dirName: string;
  path: string;
  displayName?: string;
  pluginName?: string;
}

interface CollectedScopeFacts {
  scope: InstalledViewScope;
  allowUniversalFallback: boolean;
  canonicalBase: string;
  canonicalFacts: CanonicalFact[];
  agentFacts: AgentFact[];
  agentBases: Array<{ agent: AgentType; baseDir: string }>;
  lockFacts: LockFact[];
  detectedVisibleAgents: AgentType[];
}

interface DraftInstalledView {
  lockKey?: string;
  packageName?: string;
  displayName: string;
  installName: string;
  aliases: Set<string>;
  identityAliases: Set<string>;
  scope: InstalledViewScope;
  canonicalPath: string;
  agentLinks: Map<string, { agent: AgentType; path: string; kind: AgentLinkKind }>;
  version?: string;
  sourceType: string;
  registry?: string;
  pluginName?: string;
  orphanReasons: Set<string>;
  lockEntry?: SkillLockEntry | LocalSkillLockEntry;
}

export async function listInstalledView(
  options: ListInstalledViewOptions = {}
): Promise<InstalledSkillView[]> {
  const cwd = options.cwd || process.cwd();
  const scopes = options.scope ? [options.scope] : (['project', 'global'] as InstalledViewScope[]);
  const allowUniversalFallback =
    scopes.length === 1 &&
    scopes[0] === 'global' &&
    (options.agentFilter === undefined || options.agentFilter.some((agent) => isUniversalAgent(agent)));
  const detectedAgents = await detectInstalledAgents();
  const detectedVisibleAgents = options.agentFilter
    ? detectedAgents.filter((agent) => options.agentFilter!.includes(agent))
    : detectedAgents;
  const inspectedAgents = options.agentFilter ?? (Object.keys(agents) as AgentType[]);

  const views = await Promise.all(
    scopes.map((scope) =>
      collectInstalledFacts(scope, cwd, inspectedAgents, detectedVisibleAgents, allowUniversalFallback)
    )
  );

  return views
    .flatMap((facts) => projectInstalledFacts(facts))
    .sort((left, right) =>
      `${left.scope}:${left.displayName}:${left.canonicalPath}`.localeCompare(
        `${right.scope}:${right.displayName}:${right.canonicalPath}`
      )
    );
}

async function collectInstalledFacts(
  scope: InstalledViewScope,
  cwd: string,
  inspectedAgents: AgentType[],
  detectedVisibleAgents: AgentType[],
  allowUniversalFallback: boolean
): Promise<CollectedScopeFacts> {
  const global = scope === 'global';
  const canonicalBase = getCanonicalSkillsDir(global, cwd);
  const canonicalFacts = await scanSkillDirs(canonicalBase);
  const lockFacts = await readScopeLockFacts(scope, cwd);
  const agentFacts: AgentFact[] = [];
  const agentBases: Array<{ agent: AgentType; baseDir: string }> = [];

  for (const agent of inspectedAgents) {
    if (isUniversalAgent(agent)) {
      continue;
    }

    if (global && agents[agent].globalSkillsDir === undefined) {
      continue;
    }

    const baseDir = getAgentBaseDir(agent, global, cwd);
    if (!existsSync(baseDir)) {
      continue;
    }

    agentBases.push({ agent, baseDir });

    const entries = await scanAgentDirs(baseDir);
    for (const entry of entries) {
      agentFacts.push({
        agent,
        dirName: entry.dirName,
        path: entry.path,
        displayName: entry.displayName,
        pluginName: entry.pluginName,
      });
    }
  }

  return {
    scope,
    allowUniversalFallback,
    canonicalBase,
    canonicalFacts,
    agentFacts,
    agentBases,
    lockFacts,
    detectedVisibleAgents,
  };
}

async function readScopeLockFacts(scope: InstalledViewScope, cwd: string): Promise<LockFact[]> {
  const skills =
    scope === 'global' ? await getAllLockedSkills() : (await readLocalLock(cwd)).skills;

  return Object.entries(skills).map(([key, entry]) => ({ key, entry }));
}

async function scanSkillDirs(
  baseDir: string
): Promise<Array<{ dirName: string; path: string; displayName: string; pluginName?: string }>> {
  try {
    const entries = await readdir(baseDir, { withFileTypes: true });
    const skills = await Promise.all(
      entries.map(async (entry) => {
        const skillPath = join(baseDir, entry.name);
        if (!(await isDirEntryOrSymlinkToDir(entry, skillPath))) {
          return null;
        }

        const skill = await parseSkillMd(join(skillPath, 'SKILL.md'));
        if (!skill) {
          return null;
        }

        return {
          dirName: entry.name,
          path: skillPath,
          displayName: skill.name,
          pluginName: skill.pluginName,
        };
      })
    );

    return skills
      .filter((skill): skill is NonNullable<typeof skill> => skill !== null)
      .sort((left, right) => left.path.localeCompare(right.path));
  } catch {
    return [];
  }
}

async function scanAgentDirs(
  baseDir: string
): Promise<Array<{ dirName: string; path: string; displayName?: string; pluginName?: string }>> {
  try {
    const entries = await readdir(baseDir, { withFileTypes: true });
    const skills = await Promise.all(
      entries.map(async (entry) => {
        const skillPath = join(baseDir, entry.name);
        if (!(await isDirEntryOrSymlinkToDir(entry, skillPath))) {
          return null;
        }

        const parsedSkill = await parseSkillMd(join(skillPath, 'SKILL.md'));
        if (!parsedSkill) {
          return null;
        }

        return {
          dirName: entry.name,
          path: skillPath,
          displayName: parsedSkill.name,
          pluginName: parsedSkill.pluginName,
        };
      })
    );

    return skills
      .filter((skill): skill is NonNullable<typeof skill> => skill !== null)
      .sort((left, right) => left.path.localeCompare(right.path));
  } catch {
    return [];
  }
}

async function isDirEntryOrSymlinkToDir(
  entry: { isDirectory(): boolean; isSymbolicLink(): boolean },
  entryPath: string
): Promise<boolean> {
  if (entry.isDirectory()) {
    return true;
  }

  if (!entry.isSymbolicLink()) {
    return false;
  }

  try {
    return (await stat(entryPath)).isDirectory();
  } catch {
    return false;
  }
}

function projectInstalledFacts(facts: CollectedScopeFacts): InstalledSkillView[] {
  const drafts: DraftInstalledView[] = [];
  const unmatchedLocks = new Set(facts.lockFacts.map((fact) => fact.key));
  const matchedAgentPaths = new Set<string>();
  const agentFactsByPath = new Map(facts.agentFacts.map((fact) => [fact.path, fact]));

  for (const fact of facts.canonicalFacts) {
    const lockFact = findMatchingLockFact(fact, facts.lockFacts, unmatchedLocks);
    if (lockFact) {
      unmatchedLocks.delete(lockFact.key);
    }

    const installName = lockFact?.entry.installName || fact.dirName;
    const draft = createDraft({
      scope: facts.scope,
      lockKey: lockFact?.key,
      displayName: fact.displayName,
      installName,
      canonicalPath: fact.path,
      packageName: lockFact?.entry.packageName,
      version: lockFact?.entry.version,
      sourceType: lockFact?.entry.sourceType || 'local',
      registry: lockFact?.entry.registry,
      pluginName: fact.pluginName,
      lockEntry: lockFact?.entry,
      aliases: buildAliases({
        packageName: lockFact?.entry.packageName,
        installName,
        displayName: fact.displayName,
        dirName: fact.dirName,
        lockKey: lockFact?.key,
      }),
      identityAliases: buildIdentityAliases({
        packageName: lockFact?.entry.packageName,
        installName,
        dirName: fact.dirName,
        lockKey: lockFact?.key,
      }),
      orphanReasons: lockFact ? [] : ['missing_lock_entry'],
    });
    drafts.push(draft);
  }

  for (const draft of drafts) {
    for (const agentBase of facts.agentBases) {
      const linkedPath = findExistingAgentLinkPath(agentBase.baseDir, draft, agentFactsByPath);
      if (!linkedPath || matchedAgentPaths.has(linkedPath)) {
        continue;
      }

      draft.agentLinks.set(agentBase.agent, {
        agent: agentBase.agent,
        path: linkedPath,
        kind: classifyAgentLinkKind(linkedPath, draft.canonicalPath),
      });
      matchedAgentPaths.add(linkedPath);
    }

    for (const agent of facts.detectedVisibleAgents) {
      if (!isUniversalAgent(agent)) {
        continue;
      }

      draft.agentLinks.set(agent, {
        agent,
        path: draft.canonicalPath,
        kind: 'canonical',
      });
    }

    if (facts.allowUniversalFallback && draft.agentLinks.size === 0 && existsSync(draft.canonicalPath)) {
      draft.agentLinks.set('universal', {
        agent: 'universal',
        path: draft.canonicalPath,
        kind: 'canonical',
      });
    }
  }

  for (const agentFact of facts.agentFacts) {
    if (matchedAgentPaths.has(agentFact.path)) {
      continue;
    }

    drafts.push(
      createDraft({
        scope: facts.scope,
        displayName: agentFact.displayName || agentFact.dirName,
        installName: agentFact.dirName,
        canonicalPath: join(facts.canonicalBase, sanitizeName(agentFact.dirName)),
        sourceType: 'local',
        pluginName: agentFact.pluginName,
        aliases: buildAliases({
          installName: agentFact.dirName,
          displayName: agentFact.displayName || agentFact.dirName,
          dirName: agentFact.dirName,
        }),
        identityAliases: buildIdentityAliases({
          installName: agentFact.dirName,
          dirName: agentFact.dirName,
        }),
        orphanReasons: ['missing_canonical_path'],
        agentLinks: [
          {
            agent: agentFact.agent,
            path: agentFact.path,
            kind: classifyAgentLinkKind(agentFact.path, join(facts.canonicalBase, sanitizeName(agentFact.dirName))),
          },
        ],
      })
    );
  }

  for (const lockFact of facts.lockFacts) {
    if (!unmatchedLocks.has(lockFact.key)) {
      continue;
    }

    const installName =
      lockFact.entry.installName ||
      (lockFact.entry.packageName ? deriveInstallName(lockFact.entry.packageName) : lockFact.key);

    drafts.push(
      createDraft({
        scope: facts.scope,
        lockKey: lockFact.key,
        displayName: lockFact.entry.installName || lockFact.entry.packageName || lockFact.key,
        installName,
        canonicalPath: join(facts.canonicalBase, sanitizeName(installName)),
        packageName: lockFact.entry.packageName,
        version: lockFact.entry.version,
        sourceType: lockFact.entry.sourceType,
        registry: lockFact.entry.registry,
        lockEntry: lockFact.entry,
        aliases: buildAliases({
          packageName: lockFact.entry.packageName,
          installName,
          displayName: lockFact.entry.installName || lockFact.entry.packageName || lockFact.key,
          lockKey: lockFact.key,
        }),
        identityAliases: buildIdentityAliases({
          packageName: lockFact.entry.packageName,
          installName,
          lockKey: lockFact.key,
        }),
        orphanReasons: ['missing_canonical_path'],
      })
    );
  }

  markAliasConflicts(drafts);

  return drafts.map((draft) => ({
    lockKey: draft.lockKey,
    packageName: draft.packageName,
    displayName: draft.displayName,
    installName: draft.installName,
    aliases: Array.from(draft.aliases).sort(),
    scope: draft.scope,
    canonicalPath: draft.canonicalPath,
    agentLinks: Array.from(draft.agentLinks.values()).sort((left, right) =>
      `${left.agent}:${left.path}`.localeCompare(`${right.agent}:${right.path}`)
    ),
    version: draft.version,
    sourceType: draft.sourceType,
    registry: draft.registry,
    pluginName: draft.pluginName,
    orphaned: draft.orphanReasons.size > 0,
    orphanReasons: Array.from(draft.orphanReasons).sort(),
    lockEntry: draft.lockEntry,
  }));
}

function findMatchingLockFact(
  fact: CanonicalFact,
  lockFacts: LockFact[],
  unmatchedLocks: Set<string>
): LockFact | undefined {
  const canonicalAliases = buildIdentityAliases({
    installName: fact.dirName,
    dirName: fact.dirName,
  });

  return lockFacts.find((lockFact) => {
    if (!unmatchedLocks.has(lockFact.key)) {
      return false;
    }

    return aliasesOverlap(
      canonicalAliases,
      buildIdentityAliases({
        packageName: lockFact.entry.packageName,
        installName: lockFact.entry.installName || lockFact.key,
        lockKey: lockFact.key,
      })
    );
  });
}

function createDraft(input: {
  scope: InstalledViewScope;
  lockKey?: string;
  displayName: string;
  installName: string;
  canonicalPath: string;
  sourceType: string;
  packageName?: string;
  version?: string;
  registry?: string;
  pluginName?: string;
  lockEntry?: SkillLockEntry | LocalSkillLockEntry;
  aliases: Set<string>;
  identityAliases: Set<string>;
  orphanReasons?: string[];
  agentLinks?: Array<{ agent: AgentType; path: string; kind: AgentLinkKind }>;
}): DraftInstalledView {
  const agentLinks = new Map<string, { agent: AgentType; path: string; kind: AgentLinkKind }>();
  for (const link of input.agentLinks || []) {
    agentLinks.set(link.agent, link);
  }

  return {
    lockKey: input.lockKey,
    packageName: input.packageName,
    displayName: input.displayName,
    installName: input.installName,
    aliases: input.aliases,
    identityAliases: input.identityAliases,
    scope: input.scope,
    canonicalPath: input.canonicalPath,
    agentLinks,
    version: input.version,
    sourceType: input.sourceType,
    registry: input.registry,
    pluginName: input.pluginName,
    orphanReasons: new Set(input.orphanReasons || []),
    lockEntry: input.lockEntry,
  };
}

function buildAliases(input: {
  packageName?: string;
  installName?: string;
  displayName?: string;
  dirName?: string;
  lockKey?: string;
}): Set<string> {
  const aliases = new Set<string>();
  const candidates = [
    input.packageName,
    input.installName,
    input.displayName,
    input.dirName,
    input.lockKey,
  ].filter((value): value is string => typeof value === 'string' && value.trim().length > 0);

  for (const candidate of candidates) {
    aliases.add(candidate);
  }

  for (const candidate of candidates) {
    for (const alias of getInstalledDirectoryNames(candidate, input.installName)) {
      aliases.add(alias);
    }
    aliases.add(sanitizeName(candidate));
  }

  return aliases;
}

function buildIdentityAliases(input: {
  packageName?: string;
  installName?: string;
  dirName?: string;
  lockKey?: string;
}): Set<string> {
  const aliases = new Set<string>();
  const installCandidates = [
    input.installName,
    input.dirName,
    input.lockKey,
  ].filter((value): value is string => typeof value === 'string' && value.trim().length > 0);

  if (input.packageName) {
    aliases.add(input.packageName);
  }

  for (const candidate of installCandidates) {
    aliases.add(candidate);
    aliases.add(sanitizeName(candidate));
  }

  return aliases;
}

function aliasesOverlap(left: Set<string>, right: Set<string>): boolean {
  for (const alias of left) {
    if (right.has(alias)) {
      return true;
    }
  }
  return false;
}

function findExistingAgentLinkPath(
  baseDir: string,
  draft: DraftInstalledView,
  agentFactsByPath: Map<string, AgentFact>
): string | null {
  for (const candidate of buildDirectoryCandidates(draft)) {
    const candidatePath = join(baseDir, candidate);
    if (!existsSync(candidatePath)) {
      continue;
    }

    const agentFact = agentFactsByPath.get(candidatePath);
    if (!agentFact || agentSkillMatchesDraft(agentFact, draft)) {
      return candidatePath;
    }
  }

  return null;
}

function buildDirectoryCandidates(draft: DraftInstalledView): string[] {
  const candidates = new Set<string>();
  for (const alias of draft.identityAliases) {
    candidates.add(sanitizeName(alias));
  }
  candidates.add(sanitizeName(draft.installName));

  return Array.from(candidates).filter((candidate) => candidate.length > 0);
}

function agentSkillMatchesDraft(agentFact: AgentFact, draft: DraftInstalledView): boolean {
  if (!agentFact.displayName) {
    return true;
  }

  return aliasesOverlap(draft.aliases, buildParsedSkillAliases(agentFact.displayName));
}

function buildParsedSkillAliases(displayName: string): Set<string> {
  const aliases = new Set<string>();
  aliases.add(displayName);
  aliases.add(sanitizeName(displayName));

  for (const alias of getInstalledDirectoryNames(displayName)) {
    aliases.add(alias);
  }

  return aliases;
}

function classifyAgentLinkKind(path: string, canonicalPath: string): AgentLinkKind {
  if (!existsSync(path)) {
    return 'missing';
  }

  if (path === canonicalPath) {
    return 'canonical';
  }

  try {
    return lstatSync(path).isSymbolicLink() ? 'symlink' : 'copy';
  } catch {
    return 'copy';
  }
}

function markAliasConflicts(drafts: DraftInstalledView[]): void {
  const aliasMap = new Map<string, DraftInstalledView[]>();

  for (const draft of drafts) {
    for (const alias of draft.identityAliases) {
      const records = aliasMap.get(alias) || [];
      records.push(draft);
      aliasMap.set(alias, records);
    }
  }

  for (const [_, records] of aliasMap) {
    if (records.length < 2) {
      continue;
    }

    for (const record of records) {
      record.orphanReasons.add('alias_conflict');
    }
  }
}
