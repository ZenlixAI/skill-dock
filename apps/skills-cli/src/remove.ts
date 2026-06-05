import * as p from '@clack/prompts';
import pc from 'picocolors';
import { rm } from 'fs/promises';
import { agents } from './agents.ts';
import { track } from './telemetry.ts';
import { detectAgent } from './detect-agent.ts';
import { removeSkillFromLock } from './skill-lock.ts';
import { removeSkillFromLocalLock } from './local-lock.ts';
import type { AgentType } from './types.ts';
import { listInstalledView, type InstalledSkillView } from './installed-view.ts';

export interface RemoveOptions {
  global?: boolean;
  agent?: string[];
  yes?: boolean;
  all?: boolean;
}

function normalizeSkillReference(value: string): string {
  return value.trim().toLowerCase();
}

function getRecordLabel(record: InstalledSkillView): string {
  return record.packageName || record.displayName;
}

function getRecordSelectionKey(record: InstalledSkillView): string {
  return `${record.scope}:${record.canonicalPath}:${record.packageName || record.displayName}`;
}

function selectRequestedRecords(
  records: InstalledSkillView[],
  requestedSkill: string
): InstalledSkillView[] {
  const requestedNormalized = normalizeSkillReference(requestedSkill);
  const packageMatches = records.filter(
    (record) => record.packageName && normalizeSkillReference(record.packageName) === requestedNormalized
  );

  if (packageMatches.length > 0) {
    return packageMatches;
  }

  const aliasMatches = records.filter((record) =>
    record.aliases.some((alias) => normalizeSkillReference(alias) === requestedNormalized)
  );
  if (aliasMatches.length > 0) {
    return aliasMatches;
  }

  return records.filter(
    (record) => normalizeSkillReference(record.displayName) === requestedNormalized
  );
}

async function removeRecordFromLock(
  record: InstalledSkillView,
  isGlobal: boolean,
  cwd: string
): Promise<void> {
  const candidates = [
    record.packageName,
    record.displayName,
    record.installName,
    ...record.aliases,
  ].filter((value): value is string => typeof value === 'string' && value.trim().length > 0);

  const attempted = new Set<string>();
  for (const candidate of candidates) {
    if (attempted.has(candidate)) {
      continue;
    }
    attempted.add(candidate);

    const removed = isGlobal
      ? await removeSkillFromLock(candidate)
      : await removeSkillFromLocalLock(candidate, cwd);
    if (removed) {
      return;
    }
  }
}

export async function removeCommand(skillNames: string[], options: RemoveOptions) {
  // Auto-enable non-interactive mode when running inside an AI agent
  const agentResult = await detectAgent();
  if (agentResult.isAgent) {
    options.yes = true;
    p.log.info(
      pc.bgCyan(pc.black(pc.bold(` ${agentResult.agent.name} `))) +
        ' ' +
        'Agent detected — removing non-interactively'
    );
  }

  const isGlobal = options.global ?? false;
  const cwd = process.cwd();

  const spinner = p.spinner();

  spinner.start('Scanning for installed skills...');
  const records = await listInstalledView({
    scope: isGlobal ? 'global' : 'project',
    cwd,
  });
  spinner.stop(`Found ${records.length} unique installed skill(s)`);

  if (records.length === 0) {
    p.outro(pc.yellow('No skills found to remove.'));
    return;
  }

  // Validate agent options BEFORE prompting for skill selection
  if (options.agent && options.agent.length > 0) {
    const validAgents = Object.keys(agents);
    const invalidAgents = options.agent.filter((a) => !validAgents.includes(a));

    if (invalidAgents.length > 0) {
      p.log.error(`Invalid agents: ${invalidAgents.join(', ')}`);
      p.log.info(`Valid agents: ${validAgents.join(', ')}`);
      process.exit(1);
    }
  }

  let selectedRecords: InstalledSkillView[] = [];

  if (options.all) {
    selectedRecords = records;
  } else if (skillNames.length > 0) {
    const matchedRecords = new Map<string, InstalledSkillView>();
    for (const skillName of skillNames) {
      for (const record of selectRequestedRecords(records, skillName)) {
        matchedRecords.set(getRecordSelectionKey(record), record);
      }
    }
    selectedRecords = Array.from(matchedRecords.values());

    if (selectedRecords.length === 0) {
      p.log.error(`No matching skills found for: ${skillNames.join(', ')}`);
      return;
    }
  } else {
    const choices = records.map((record, index) => ({
      value: String(index),
      label: getRecordLabel(record),
    }));

    const selected = await p.multiselect({
      message: `Select skills to remove ${pc.dim('(space to toggle)')}`,
      options: choices,
      required: true,
    });

    if (p.isCancel(selected)) {
      p.cancel('Removal cancelled');
      process.exit(0);
    }

    selectedRecords = (selected as string[])
      .map((value) => records[Number(value)])
      .filter((record): record is InstalledSkillView => Boolean(record));
  }

  let targetAgents: AgentType[];
  if (options.agent && options.agent.length > 0) {
    targetAgents = options.agent as AgentType[];
  } else {
    // When removing, we should target all known agents to ensure
    // ghost symlinks are cleaned up, even if the agent is not detected.
    targetAgents = Object.keys(agents) as AgentType[];
    spinner.stop(`Targeting ${targetAgents.length} potential agent(s)`);
  }

  if (!options.yes) {
    console.log();
    p.log.info('Skills to remove:');
    for (const record of selectedRecords) {
      p.log.message(`  ${pc.red('•')} ${getRecordLabel(record)}`);
    }
    console.log();

    const confirmed = await p.confirm({
      message: `Are you sure you want to uninstall ${selectedRecords.length} skill(s)?`,
    });

    if (p.isCancel(confirmed) || !confirmed) {
      p.cancel('Removal cancelled');
      process.exit(0);
    }
  }

  spinner.start('Removing skills...');

  const results: {
    skill: string;
    success: boolean;
    source?: string;
    sourceType?: string;
    error?: string;
  }[] = [];

  for (const record of selectedRecords) {
    try {
      const targetLinks = record.agentLinks.filter(
        (link) => targetAgents.includes(link.agent) && link.kind !== 'missing'
      );
      const untargetedVisibleLinks = record.agentLinks.filter(
        (link) => !targetAgents.includes(link.agent) && link.kind !== 'missing'
      );
      const pathsToCleanup = new Set<string>([
        record.canonicalPath,
        ...targetLinks.map((link) => link.path),
      ]);
      if (untargetedVisibleLinks.length > 0) {
        pathsToCleanup.delete(record.canonicalPath);
      }

      for (const pathToCleanup of pathsToCleanup) {
        try {
          await rm(pathToCleanup, { recursive: true, force: true });
        } catch (err) {
          p.log.warn(
            `Could not remove ${getRecordLabel(record)} from ${pathToCleanup}: ${
              err instanceof Error ? err.message : String(err)
            }`
          );
        }
      }

      const keepsVisibleInstall = untargetedVisibleLinks.length > 0;
      if (!keepsVisibleInstall) {
        await removeRecordFromLock(record, isGlobal, cwd);
      }

      results.push({
        skill: getRecordLabel(record),
        success: true,
        source: record.packageName || record.displayName,
        sourceType: record.sourceType,
      });
    } catch (err) {
      results.push({
        skill: getRecordLabel(record),
        success: false,
        error: err instanceof Error ? err.message : String(err),
      });
    }
  }

  spinner.stop('Removal process complete');

  const successful = results.filter((r) => r.success);
  const failed = results.filter((r) => !r.success);

  // Track removal (grouped by source)
  if (successful.length > 0) {
    const bySource = new Map<string, { skills: string[]; sourceType?: string }>();

    for (const r of successful) {
      const source = r.source || 'local';
      const existing = bySource.get(source) || { skills: [] };
      existing.skills.push(r.skill);
      existing.sourceType = r.sourceType;
      bySource.set(source, existing);
    }

    for (const [source, data] of bySource) {
      track({
        event: 'remove',
        source,
        skills: data.skills.join(','),
        agents: targetAgents.join(','),
        ...(isGlobal && { global: '1' }),
        sourceType: data.sourceType,
      });
    }
  }

  if (successful.length > 0) {
    p.log.success(pc.green(`Successfully removed ${successful.length} skill(s)`));
  }

  if (failed.length > 0) {
    p.log.error(pc.red(`Failed to remove ${failed.length} skill(s)`));
    for (const r of failed) {
      p.log.message(`  ${pc.red('✗')} ${r.skill}: ${r.error}`);
    }
  }

  console.log();
  p.outro(pc.green('Done!'));
}

/**
 * Parse command line options for the remove command.
 * Separates skill names from options flags.
 */
export function parseRemoveOptions(args: string[]): { skills: string[]; options: RemoveOptions } {
  const options: RemoveOptions = {};
  const skills: string[] = [];

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];

    if (arg === '-g' || arg === '--global') {
      options.global = true;
    } else if (arg === '-y' || arg === '--yes') {
      options.yes = true;
    } else if (arg === '--all') {
      options.all = true;
    } else if (arg === '-a' || arg === '--agent') {
      options.agent = options.agent || [];
      i++;
      let nextArg = args[i];
      while (i < args.length && nextArg && !nextArg.startsWith('-')) {
        options.agent.push(nextArg);
        i++;
        nextArg = args[i];
      }
      i--; // Back up one since the loop will increment
    } else if (arg && !arg.startsWith('-')) {
      skills.push(arg);
    }
  }

  return { skills, options };
}
