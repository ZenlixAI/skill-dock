import { runImportGitWorkflow } from './import-git-workflow.ts';
import { formatImportReportSummary, formatPackagingWarnings } from './packaging-warning-output.ts';

interface ImportGitOptions {
  source?: string;
  registry?: string;
  strict?: boolean;
  packageName?: string;
  version?: string;
  publishRegistry?: string;
  skill?: string[];
  fullDepth?: boolean;
}

export function parseImportGitOptions(args: string[]): ImportGitOptions {
  const options: ImportGitOptions = {};
  const skills: string[] = [];
  const positional: string[] = [];

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (typeof arg !== 'string') {
      continue;
    }

    switch (arg) {
      case '--registry':
        options.registry = args[++i];
        break;
      case '--strict':
        options.strict = true;
        break;
      case '--package-name':
        options.packageName = args[++i];
        break;
      case '--version':
        options.version = args[++i];
        break;
      case '--publish-registry':
        options.publishRegistry = args[++i];
        break;
      case '--skill':
        if (typeof args[i + 1] === 'string') {
          skills.push(args[++i]!);
        }
        break;
      case '--full-depth':
        options.fullDepth = true;
        break;
      default:
        if (!arg.startsWith('-')) {
          positional.push(arg);
        }
        break;
    }
  }

  if (positional.length > 0) {
    options.source = positional[0];
  }
  if (skills.length > 0) {
    options.skill = skills;
  }

  return options;
}

export async function runImportGit(args: string[] = []): Promise<void> {
  const options = parseImportGitOptions(args);
  console.log();

  if (!options.source) {
    console.log('Import failed');
    console.log();
    console.log('- source is required');
    process.exitCode = 1;
    return;
  }

  const result = await runImportGitWorkflow({
    source: options.source,
    registry: options.registry,
    strict: options.strict,
    packageName: options.packageName,
    version: options.version,
    publishRegistry: options.publishRegistry,
    skill: options.skill,
    fullDepth: options.fullDepth,
  });

  if (!result.ok) {
    console.log('Import failed');
    console.log();
    for (const error of result.errors) {
      console.log(`- ${error}`);
    }
    process.exitCode = 1;
    return;
  }

  console.log('Import OK');
  console.log();
  console.log(`package_name: ${result.packageName}`);
  console.log(`version: ${result.version}`);
  console.log(`manifest_digest: ${result.manifestDigest}`);
  console.log(`components: ${result.components.join(', ')}`);

  const importSummary = formatImportReportSummary(result.importReport);
  if (importSummary.length > 0) {
    console.log();
    console.log('Import summary:');
    for (const line of importSummary) {
      console.log(`- ${line}`);
    }
  }
  if (result.warnings && result.warnings.length > 0) {
    const warningDetails = formatPackagingWarnings(result.warnings, result.warningDetails);
    console.log();
    console.log('Warnings:');
    for (const warning of warningDetails) {
      console.log(`- ${warning.title}`);
      console.log(`  code: ${warning.code}`);
      console.log(`  ${warning.message}`);
      if (warning.suggestedAction) {
        console.log(`  Suggested action: ${warning.suggestedAction}`);
      }
    }
  }
}
