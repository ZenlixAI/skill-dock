import { join } from 'path';
import { existsSync, readdirSync } from 'fs';
import * as p from '@clack/prompts';

import { agents, detectInstalledAgents, getUniversalAgents } from './agents.ts';
import { runRegistryInstallWorkflow } from './registry-install-workflow.ts';
import { listInstalledView, type InstalledSkillView } from './installed-view.ts';
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

function recordMatchesSkillFilter(record: InstalledSkillView, filter?: string[]): boolean {
  if (!filter || filter.length === 0) {
    return true;
  }

  const requested = filter.map((value) => value.toLowerCase());
  if (record.packageName && requested.includes(record.packageName.toLowerCase())) {
    return true;
  }

  return record.aliases.some((alias) => requested.includes(alias.toLowerCase()));
}

function getRegistryRecordsForUpdate(
  records: InstalledSkillView[],
  skillFilter?: string[]
): InstalledSkillView[] {
  return records.filter((record) => {
    if (!record.packageName) return false;
    if (record.sourceType !== 'registry') return false;
    return recordMatchesSkillFilter(record, skillFilter);
  });
}

function getUpdatableRegistryRecords(
  records: InstalledSkillView[],
  skillFilter?: string[]
): InstalledSkillView[] {
  return getRegistryRecordsForUpdate(records, skillFilter).filter((record) => !record.orphaned);
}

async function updateRegistryEntries(
  entries: InstalledSkillView[],
  global: boolean
): Promise<{ successCount: number; failCount: number; alreadyUpToDateCount: number }> {
  let successCount = 0;
  let failCount = 0;
  let alreadyUpToDateCount = 0;

  if (entries.length === 0) {
    return { successCount, failCount, alreadyUpToDateCount };
  }

  for (const skill of entries) {
    const packageName = skill.packageName!;
    const safeName = sanitizeMetadata(packageName);

    console.log(`${TEXT}Updating ${safeName}...${RESET}`);

    try {
      const target = await resolveInstallTarget(packageName, {
        registry: skill.registry,
        version: undefined,
      });

      if (skill.version && skill.version === target.version) {
        alreadyUpToDateCount++;
        console.log(`  ${DIM}Already up to date${RESET} ${safeName} ${DIM}@ ${target.version}${RESET}`);
        continue;
      }

      const targetAgents = await resolveTargetAgentsForUpdate();
      await runRegistryInstallWorkflow({
        packageName,
        version: target.version,
        registry: skill.registry,
        target: target,
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

export async function updateGlobalSkills(
  options: UpdateCheckOptions = {}
): Promise<{
  successCount: number;
  failCount: number;
  matchedCount: number;
  updatableCount: number;
  skippedCount: number;
}> {
  const records = await listInstalledView({ scope: 'global' });
  const registryRecords = getRegistryRecordsForUpdate(records, options.skills);
  const updatableRecords = getUpdatableRegistryRecords(records, options.skills);
  const orphanedRecords = registryRecords.filter((record) => record.orphaned);

  if (registryRecords.length === 0) {
    if (!options.skills) {
      console.log(`${DIM}No global skills to update.${RESET}`);
      console.log(`${DIM}Install global skills with${RESET} ${TEXT}skills add <package_name> -g${RESET}`);
    }
    return { successCount: 0, failCount: 0, matchedCount: 0, updatableCount: 0, skippedCount: 0 };
  }

  for (const record of orphanedRecords) {
    console.log(
      `${DIM}Skipping ${sanitizeMetadata(record.packageName || record.displayName)}:${RESET} repair local installed state before update`
    );
  }

  const result = await updateRegistryEntries(updatableRecords, true);

  if (
    updatableRecords.length > 0 &&
    result.failCount === 0 &&
    result.successCount === 0 &&
    result.alreadyUpToDateCount > 0
  ) {
    console.log(`${TEXT}✓ All registry-installed global skills are up to date${RESET}`);
  }

  return {
    successCount: result.successCount,
    failCount: result.failCount,
    matchedCount: registryRecords.length,
    updatableCount: updatableRecords.length,
    skippedCount: orphanedRecords.length,
  };
}

export async function updateProjectSkills(
  options: UpdateCheckOptions = {}
): Promise<{
  successCount: number;
  failCount: number;
  matchedCount: number;
  updatableCount: number;
  skippedCount: number;
}> {
  const records = await listInstalledView({ scope: 'project' });
  const registryRecords = getRegistryRecordsForUpdate(records, options.skills);
  const updatableRecords = getUpdatableRegistryRecords(records, options.skills);
  const orphanedRecords = registryRecords.filter((record) => record.orphaned);

  if (registryRecords.length === 0) {
    if (!options.skills) {
      console.log(`${DIM}No project skills to update.${RESET}`);
      console.log(`${DIM}Install project skills with${RESET} ${TEXT}skills add <package_name>${RESET}`);
    }
    return { successCount: 0, failCount: 0, matchedCount: 0, updatableCount: 0, skippedCount: 0 };
  }

  for (const record of orphanedRecords) {
    console.log(
      `${DIM}Skipping ${sanitizeMetadata(record.packageName || record.displayName)}:${RESET} repair local installed state before update`
    );
  }

  const result = await updateRegistryEntries(updatableRecords, false);

  if (
    updatableRecords.length > 0 &&
    result.failCount === 0 &&
    result.successCount === 0 &&
    result.alreadyUpToDateCount > 0
  ) {
    console.log(`${TEXT}✓ All registry-installed project skills are up to date${RESET}`);
  }

  return {
    successCount: result.successCount,
    failCount: result.failCount,
    matchedCount: registryRecords.length,
    updatableCount: updatableRecords.length,
    skippedCount: orphanedRecords.length,
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
  let totalMatched = 0;
  let totalUpdatable = 0;
  let totalSkipped = 0;

  if (scope === 'global' || scope === 'both') {
    if (scope === 'both' && !options.skills) {
      console.log(`${BOLD}Global Skills${RESET}`);
    }
    const { successCount, failCount, matchedCount, updatableCount, skippedCount } =
      await updateGlobalSkills(options);
    totalSuccess += successCount;
    totalFail += failCount;
    totalMatched += matchedCount;
    totalUpdatable += updatableCount;
    totalSkipped += skippedCount;
    if (scope === 'both' && !options.skills) {
      console.log();
    }
  }

  if (scope === 'project' || scope === 'both') {
    if (scope === 'both' && !options.skills) {
      console.log(`${BOLD}Project Skills${RESET}`);
    }
    const { successCount, failCount, matchedCount, updatableCount, skippedCount } =
      await updateProjectSkills(options);
    totalSuccess += successCount;
    totalFail += failCount;
    totalMatched += matchedCount;
    totalUpdatable += updatableCount;
    totalSkipped += skippedCount;
  }

  if (options.skills && totalMatched === 0) {
    console.log(`${DIM}No installed skills found matching: ${options.skills.join(', ')}${RESET}`);
  }

  console.log();
  if (totalSuccess > 0) {
    console.log(`${TEXT}✓ Updated ${totalSuccess} skill(s)${RESET}`);
  }
  if (totalFail > 0) {
    console.log(`${DIM}Failed to update ${totalFail} skill(s)${RESET}`);
  }
  if (totalSkipped > 0) {
    console.log(
      `${DIM}Skipped ${totalSkipped} skill(s) that need local state repair before update.${RESET}`
    );
  }
  if (totalUpdatable > 0 && totalSuccess === 0 && totalFail === 0) {
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
