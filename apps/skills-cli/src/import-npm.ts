import { runImportNpmWorkflow } from './import-npm-workflow.ts';
import { formatImportReportSummary, formatPackagingWarnings } from './packaging-warning-output.ts';

interface ImportNpmOptions {
  packageName?: string;
  registry?: string;
  version?: string;
  strict?: boolean;
  hosted?: boolean;
}

export function parseImportNpmOptions(args: string[]): ImportNpmOptions {
  const options: ImportNpmOptions = {};
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
      case '--version':
        options.version = args[++i];
        break;
      case '--strict':
        options.strict = true;
        break;
      case '--hosted':
        options.hosted = true;
        break;
      case '--package-name':
        options.packageName = args[++i];
        break;
      default:
        if (!arg.startsWith('-')) {
          positional.push(arg);
        }
        break;
    }
  }

  if (positional.length > 0) {
    options.packageName = options.packageName || positional[0];
  }

  return options;
}

export async function runImportNpm(args: string[] = []): Promise<void> {
  const options = parseImportNpmOptions(args);
  console.log();

  if (!options.packageName) {
    console.log('Import failed');
    console.log();
    console.log('- packageName is required');
    process.exitCode = 1;
    return;
  }

  const result = await runImportNpmWorkflow({
    packageName: options.packageName,
    registry: options.registry,
    version: options.version,
    strict: options.strict,
    hosted: options.hosted,
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
  console.log(`source_registry: ${result.sourceRegistry}`);
  console.log(`import_mode: ${result.importMode}`);

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
