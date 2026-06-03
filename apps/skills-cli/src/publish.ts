import { runPublishWorkflow } from './publish-workflow.ts';
import { formatImportReportSummary, formatPackagingWarnings } from './packaging-warning-output.ts';

interface PublishOptions {
  path?: string;
  registry?: string;
  strict?: boolean;
}

export function parsePublishOptions(args: string[]): PublishOptions {
  const options: PublishOptions = {};
  const positional: string[] = [];

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];

    if (arg === '--registry') {
      options.registry = args[++i];
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

export async function runPublish(args: string[] = []): Promise<void> {
  const options = parsePublishOptions(args);
  console.log();
  const result = await runPublishWorkflow({
    path: options.path,
    registry: options.registry,
    strict: options.strict,
  });

  if (result.ok) {
    console.log('Publish OK');
    console.log();
    console.log(`package_name: ${result.packageName}`);
    console.log(`version: ${result.version}`);
    console.log(`bundle: ${result.bundlePath}`);
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
    return;
  }

  console.log('Publish failed');
  console.log();
  for (const error of result.errors) {
    console.log(`- ${error}`);
  }
  process.exitCode = 1;
}
