import { resolve } from 'path';
import {
  runPackagingSession,
  type RegistryPackagingSessionSuccessResult,
} from './packaging-session.ts';
import { formatImportReportSummary, formatPackagingWarnings } from './packaging-warning-output.ts';

interface InspectOptions {
  path?: string;
  registry?: string;
  strict?: boolean;
}

function isRegistryValue(value: string | undefined): value is string {
  if (typeof value !== 'string' || value.trim() === '') {
    return false;
  }

  try {
    const url = new URL(value);
    return url.protocol === 'http:' || url.protocol === 'https:';
  } catch {
    return false;
  }
}

export function parseInspectOptions(args: string[]): InspectOptions {
  const options: InspectOptions = {};
  const positional: string[] = [];

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];

    if (arg === '--registry') {
      const candidate = args[i + 1];
      if (isRegistryValue(candidate)) {
        options.registry = candidate;
        i++;
      }
      continue;
    }

    if (arg === '--strict') {
      options.strict = true;
      continue;
    }

    if (typeof arg === 'string' && !arg.startsWith('-')) {
      positional.push(arg);
    }
  }

  if (positional.length > 0) {
    options.path = positional[0];
  }

  return options;
}

function isRegistryPackagingSuccess(
  result: { ok: boolean; sessionKind: 'local' | 'registry' } | RegistryPackagingSessionSuccessResult
): result is RegistryPackagingSessionSuccessResult {
  return 'sessionKind' in result && result.ok && result.sessionKind === 'registry';
}

export async function runInspect(args: string[] = []): Promise<void> {
  const options = parseInspectOptions(args);
  const registry = options.registry || process.env.SKILLS_REGISTRY_URL?.trim();
  const result = await runPackagingSession({
    path: resolve(options.path || process.cwd()),
    registry,
    strict: options.strict,
  });

  console.log();

  if (!result.ok) {
    console.log('Inspect failed');
    console.log();
    for (const error of result.errors) {
      console.log(`- ${error}`);
    }
    process.exitCode = 1;
    return;
  }

  console.log('Inspect OK');
  console.log();
  console.log(`Directory: ${result.directory}`);
  console.log(`SKILL.md: ${result.skillMdPath}`);
  console.log(`package.json: ${result.packageJsonPath}`);
  if (result.packageName) {
    console.log(`package_name: ${result.packageName}`);
  }
  if (result.version) {
    console.log(`version: ${result.version}`);
  }
  if (isRegistryPackagingSuccess(result)) {
    console.log(`workspace_handle: ${result.workspacePath}`);
    const importSummary = formatImportReportSummary(result.importReport);
    if (importSummary.length > 0) {
      console.log('Import summary:');
      for (const line of importSummary) {
        console.log(`- ${line}`);
      }
    }
    if (result.warnings && result.warnings.length > 0) {
      console.log('Warnings:');
      for (const warning of formatPackagingWarnings(result.warnings, result.warningDetails)) {
        console.log(`- ${warning.title}`);
        console.log(`  code: ${warning.code}`);
        console.log(`  ${warning.message}`);
        if (warning.suggestedAction) {
          console.log(`  Suggested action: ${warning.suggestedAction}`);
        }
      }
    }
  }
  if (result.components.length > 0) {
    console.log(`components: ${result.components.join(', ')}`);
  }
}
