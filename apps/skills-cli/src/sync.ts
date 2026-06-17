import * as p from '@clack/prompts';
import { runSyncWorkflow } from './sync-workflow.ts';
import { hasProjectSkills } from './update.ts';

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

async function resolveSyncOptions(options: SyncOptions): Promise<SyncOptions> {
  if (options.global || options.project) {
    return options;
  }

  if (options.yes || !process.stdin.isTTY) {
    return hasProjectSkills() ? { ...options, project: true } : { ...options, global: true };
  }

  const scope = await p.select({
    message: 'Sync scope',
    options: [
      {
        value: 'project',
        label: 'Project',
        hint: 'Refresh source-installed skills in current directory',
      },
      {
        value: 'global',
        label: 'Global',
        hint: 'Refresh source-installed skills in home directory',
      },
    ],
  });

  if (p.isCancel(scope)) {
    p.cancel('Cancelled');
    process.exit(0);
  }

  return scope === 'global' ? { ...options, global: true } : { ...options, project: true };
}

export async function runSync(args: string[] = []): Promise<void> {
  const options = await resolveSyncOptions(parseSyncOptions(args));

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
