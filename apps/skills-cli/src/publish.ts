import { runPublishWorkflow } from './publish-workflow.ts';

interface PublishOptions {
  path?: string;
  registry?: string;
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
  });

  if (result.ok) {
    console.log('Publish OK');
    console.log();
    console.log(`package_name: ${result.packageName}`);
    console.log(`version: ${result.version}`);
    console.log(`bundle: ${result.bundlePath}`);
    return;
  }

  console.log('Publish failed');
  console.log();
  for (const error of result.errors) {
    console.log(`- ${error}`);
  }
  process.exitCode = 1;
}
