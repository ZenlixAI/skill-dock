import { readFile } from 'fs/promises';
import { join } from 'path';
import { existsSync, readdirSync } from 'fs';
import * as p from '@clack/prompts';

import { agents, detectInstalledAgents, getUniversalAgents } from './agents.ts';
import { hasSkillPackageScope, isExactVersion } from './authoring.ts';
import { readSkillLock, type SkillLockEntry } from './skill-lock.ts';
import { readLocalLock, type LocalSkillLockEntry } from './local-lock.ts';
import { runRegistryInstallWorkflow } from './registry-install-workflow.ts';
import { listInstalledSkills } from './installer.ts';
import { sanitizeMetadata } from './sanitize.ts';
import { track } from './telemetry.ts';
import { resolveInstallTarget } from './catalog.ts';
import type { AgentType } from './types.ts';

const RESET = '\x1b[0m';
const BOLD = '\x1b[1m';
const DIM = '\x1b[38;5;102m';
const TEXT = '\x1b[38;5;145m';

export type UpdateScope = 'project' | 'global' | 'both';

export interface UpdateCheckOptions {
  global?: boolean;
  project?: boolean;
  yes?: boolean;
  skills?: string[];
}

type RegistryUpdateEntry = {
  name: string;
  entry: SkillLockEntry | LocalSkillLockEntry;
};

interface InstalledPackageJson {
  name?: unknown;
  version?: unknown;
}

function ensureUniversalAgents(targetAgents: AgentType[]): AgentType[] {
  const universalAgents = getUniversalAgents();
  const result = [...targetAgents];

  for (const ua of universalAgents) {
    if (!result.includes(ua)) {
      result.push(ua);
    }
  }

  return result;
}

async function resolveTargetAgentsForUpdate(): Promise<AgentType[]> {
  const installedAgents = await detectInstalledAgents();
  if (installedAgents.length === 0) {
    return Object.keys(agents) as AgentType[];
  }
  return ensureUniversalAgents(installedAgents);
}

function resolveInstallModeForUpdate(targetAgents: AgentType[]): 'copy' | 'symlink' {
  const uniqueDirs = new Set(targetAgents.map((a) => agents[a].skillsDir));
  return uniqueDirs.size <= 1 ? 'copy' : 'symlink';
}

export function parseUpdateOptions(args: string[]): UpdateCheckOptions {
  const options: UpdateCheckOptions = {};
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

export function hasProjectSkills(cwd?: string): boolean {
  const dir = cwd || process.cwd();

  if (existsSync(join(dir, 'skills-lock.json'))) {
    return true;
  }

  const skillsDir = join(dir, '.agents', 'skills');
  try {
    const entries = readdirSync(skillsDir, { withFileTypes: true });
    for (const entry of entries) {
      if (entry.isDirectory() && existsSync(join(skillsDir, entry.name, 'SKILL.md'))) {
        return true;
      }
    }
  } catch {
    // Directory does not exist.
  }

  return false;
}

export async function resolveUpdateScope(options: UpdateCheckOptions): Promise<UpdateScope> {
  if (options.skills && options.skills.length > 0) {
    if (options.global) return 'global';
    if (options.project) return 'project';
    return 'both';
  }

  if (options.global && options.project) return 'both';
  if (options.global) return 'global';
  if (options.project) return 'project';

  if (options.yes || !process.stdin.isTTY) {
    return hasProjectSkills() ? 'project' : 'global';
  }

  const scope = await p.select({
    message: 'Update scope',
    options: [
      {
        value: 'project' as UpdateScope,
        label: 'Project',
        hint: 'Update skills in current directory',
      },
      {
        value: 'global' as UpdateScope,
        label: 'Global',
        hint: 'Update skills in home directory',
      },
      {
        value: 'both' as UpdateScope,
        label: 'Both',
        hint: 'Update all skills',
      },
    ],
  });

  if (p.isCancel(scope)) {
    p.cancel('Cancelled');
    process.exit(0);
  }

  return scope as UpdateScope;
}

export function matchesSkillFilter(name: string, filter?: string[]): boolean {
  if (!filter || filter.length === 0) return true;
  const lower = name.toLowerCase();
  return filter.some((value) => value.toLowerCase() === lower);
}

async function inferUnlockedGlobalRegistryEntries(
  lockedSkills: Record<string, SkillLockEntry>,
  skillFilter?: string[]
): Promise<RegistryUpdateEntry[]> {
  const installedSkills = await listInstalledSkills({ global: true });
  const lockedPackageNames = new Set(
    Object.entries(lockedSkills)
      .filter(([, entry]) => entry.sourceType === 'registry')
      .map(([name, entry]) => (entry.packageName || name).toLowerCase())
  );
  const lockedInstallNames = new Set(
    Object.entries(lockedSkills)
      .filter(([, entry]) => entry.sourceType === 'registry')
      .map(([name, entry]) => (entry.installName || name).toLowerCase())
  );
  const inferred: RegistryUpdateEntry[] = [];

  for (const installed of installedSkills) {
    const installName = installed.name.trim();
    if (lockedInstallNames.has(installName.toLowerCase())) {
      continue;
    }

    try {
      const raw = await readFile(join(installed.canonicalPath, 'package.json'), 'utf-8');
      const pkg = JSON.parse(raw) as InstalledPackageJson;
      if (typeof pkg.name !== 'string' || !hasSkillPackageScope(pkg.name)) {
        continue;
      }
      if (typeof pkg.version !== 'string' || !isExactVersion(pkg.version)) {
        continue;
      }
      if (
        !matchesSkillFilter(pkg.name, skillFilter) &&
        !matchesSkillFilter(installName, skillFilter)
      ) {
        continue;
      }
      if (lockedPackageNames.has(pkg.name.toLowerCase())) {
        continue;
      }

      inferred.push({
        name: pkg.name,
        entry: {
          sourceType: 'registry',
          skillFolderHash: '',
          installedAt: '',
          updatedAt: '',
          packageName: pkg.name,
          installName,
          version: pkg.version,
          registry: process.env.SKILLS_REGISTRY_URL?.trim() || undefined,
        },
      });
      lockedPackageNames.add(pkg.name.toLowerCase());
      lockedInstallNames.add(installName.toLowerCase());
    } catch {
      continue;
    }
  }

  return inferred;
}

async function updateRegistryEntries(
  entries: RegistryUpdateEntry[],
  global: boolean
): Promise<{ successCount: number; failCount: number; alreadyUpToDateCount: number }> {
  let successCount = 0;
  let failCount = 0;
  let alreadyUpToDateCount = 0;

  if (entries.length === 0) {
    return { successCount, failCount, alreadyUpToDateCount };
  }

  for (const skill of entries) {
    const packageName = skill.entry.packageName || skill.name;
    const safeName = sanitizeMetadata(packageName);

    console.log(`${TEXT}Updating ${safeName}...${RESET}`);

    try {
      const target = await resolveInstallTarget(packageName, {
        registry: skill.entry.registry,
        version: undefined,
      });

      if (skill.entry.version && skill.entry.version === target.version) {
        alreadyUpToDateCount++;
        console.log(`  ${DIM}Already up to date${RESET} ${safeName} ${DIM}@ ${target.version}${RESET}`);
        continue;
      }

      const targetAgents = await resolveTargetAgentsForUpdate();
      await runRegistryInstallWorkflow({
        packageName,
        version: target.version,
        registry: skill.entry.registry,
        targetAgents,
        installGlobally: global,
        installMode: resolveInstallModeForUpdate(targetAgents),
      });
      successCount++;
      console.log(`  ${TEXT}✓${RESET} Updated ${safeName}`);
    } catch {
      failCount++;
      console.log(`  ${DIM}✗ Failed to update ${safeName}${RESET}`);
    }
  }

  return { successCount, failCount, alreadyUpToDateCount };
}

export async function getProjectSkillsForUpdate(
  skillFilter?: string[]
): Promise<Array<{ name: string; entry: LocalSkillLockEntry }>> {
  const localLock = await readLocalLock();
  const skills: Array<{ name: string; entry: LocalSkillLockEntry }> = [];

  for (const [name, entry] of Object.entries(localLock.skills)) {
    if (!matchesSkillFilter(name, skillFilter)) continue;
    if (entry.sourceType !== 'registry') continue;
    skills.push({ name, entry });
  }

  return skills;
}

export async function updateGlobalSkills(
  options: UpdateCheckOptions = {}
): Promise<{ successCount: number; failCount: number; checkedCount: number }> {
  const lock = await readSkillLock();
  const matchedEntries = Object.entries(lock.skills)
    .filter(([name]) => matchesSkillFilter(name, options.skills))
    .map(([name, entry]) => ({ name, entry }));
  const inferredEntries = await inferUnlockedGlobalRegistryEntries(lock.skills, options.skills);
  const combinedEntries = [...matchedEntries, ...inferredEntries];

  if (combinedEntries.length === 0) {
    if (!options.skills) {
      console.log(`${DIM}No global skills tracked in lock file.${RESET}`);
      console.log(`${DIM}Install skills with${RESET} ${TEXT}skills add <package_name> -g${RESET}`);
    }
    return { successCount: 0, failCount: 0, checkedCount: 0 };
  }

  const registryEntries = combinedEntries.filter((item) => item.entry.sourceType === 'registry');
  const result = await updateRegistryEntries(registryEntries, true);

  if (
    registryEntries.length > 0 &&
    result.failCount === 0 &&
    result.successCount === 0 &&
    result.alreadyUpToDateCount > 0
  ) {
    console.log(`${TEXT}✓ All registry-installed global skills are up to date${RESET}`);
  }

  return {
    successCount: result.successCount,
    failCount: result.failCount,
    checkedCount: combinedEntries.length,
  };
}

export async function updateProjectSkills(
  options: UpdateCheckOptions = {}
): Promise<{ successCount: number; failCount: number; foundCount: number }> {
  const projectSkills = await getProjectSkillsForUpdate(options.skills);

  if (projectSkills.length === 0) {
    if (!options.skills) {
      console.log(`${DIM}No project skills to update.${RESET}`);
      console.log(`${DIM}Install project skills with${RESET} ${TEXT}skills add <package_name>${RESET}`);
    }
    return { successCount: 0, failCount: 0, foundCount: 0 };
  }

  const registryEntries = projectSkills.filter((item) => item.entry.sourceType === 'registry');
  const result = await updateRegistryEntries(registryEntries, false);

  if (
    registryEntries.length > 0 &&
    result.failCount === 0 &&
    result.successCount === 0 &&
    result.alreadyUpToDateCount > 0
  ) {
    console.log(`${TEXT}✓ All registry-installed project skills are up to date${RESET}`);
  }

  return {
    successCount: result.successCount,
    failCount: result.failCount,
    foundCount: projectSkills.length,
  };
}

export async function runUpdate(args: string[] = []): Promise<void> {
  const options = parseUpdateOptions(args);
  const scope = await resolveUpdateScope(options);

  if (options.skills) {
    console.log(`${TEXT}Updating ${options.skills.join(', ')}...${RESET}`);
  } else {
    console.log(`${TEXT}Checking for skill updates...${RESET}`);
  }
  console.log();

  let totalSuccess = 0;
  let totalFail = 0;
  let totalFound = 0;

  if (scope === 'global' || scope === 'both') {
    if (scope === 'both' && !options.skills) {
      console.log(`${BOLD}Global Skills${RESET}`);
    }
    const { successCount, failCount, checkedCount } = await updateGlobalSkills(options);
    totalSuccess += successCount;
    totalFail += failCount;
    totalFound += checkedCount;
    if (scope === 'both' && !options.skills) {
      console.log();
    }
  }

  if (scope === 'project' || scope === 'both') {
    if (scope === 'both' && !options.skills) {
      console.log(`${BOLD}Project Skills${RESET}`);
    }
    const { successCount, failCount, foundCount } = await updateProjectSkills(options);
    totalSuccess += successCount;
    totalFail += failCount;
    totalFound += foundCount;
  }

  if (options.skills && totalFound === 0) {
    console.log(`${DIM}No installed skills found matching: ${options.skills.join(', ')}${RESET}`);
  }

  console.log();
  if (totalSuccess > 0) {
    console.log(`${TEXT}✓ Updated ${totalSuccess} skill(s)${RESET}`);
  }
  if (totalFail > 0) {
    console.log(`${DIM}Failed to update ${totalFail} skill(s)${RESET}`);
  }
  if (totalFound > 0 && totalSuccess === 0 && totalFail === 0) {
    console.log(`${DIM}No updates were needed.${RESET}`);
  }

  track({
    event: 'update',
    scope,
    skillCount: String(totalSuccess + totalFail),
    successCount: String(totalSuccess),
    failCount: String(totalFail),
  });
}
