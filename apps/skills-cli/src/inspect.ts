import { resolve } from 'path';
import { inspectSkillDirectory } from './authoring.ts';
import { validateLocalWorkspace } from './local-packaging.ts';

interface InspectOptions {
  path?: string;
}

export function parseInspectOptions(args: string[]): InspectOptions {
  const options: InspectOptions = {};
  const positional: string[] = [];

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];

    if (typeof arg === 'string' && !arg.startsWith('-')) {
      positional.push(arg);
    }
  }

  if (positional.length > 0) {
    options.path = positional[0];
  }

  return options;
}

export async function runInspect(args: string[] = []): Promise<void> {
  const options = parseInspectOptions(args);
  const directory = resolve(options.path || process.cwd());
  const inspection = await inspectSkillDirectory(directory);
  const localValidationErrors =
    inspection.ok && inspection.packageName && inspection.version
      ? await validateLocalWorkspace(directory)
      : [];
  const errors =
    inspection.errors.length > 0 ? inspection.errors : localValidationErrors;

  console.log();

  if (errors.length > 0) {
    console.log('Inspect failed');
    console.log();
    for (const error of errors) {
      console.log(`- ${error}`);
    }
    process.exitCode = 1;
    return;
  }

  console.log('Inspect OK');
  console.log();
  console.log(`Directory: ${inspection.directory}`);
  console.log(`SKILL.md: ${inspection.skillMdPath}`);
  console.log(`package.json: ${inspection.packageJsonPath}`);
  if (inspection.packageName) {
    console.log(`package_name: ${inspection.packageName}`);
  }
  if (inspection.version) {
    console.log(`version: ${inspection.version}`);
  }
  if (inspection.components.length > 0) {
    console.log(`components: ${inspection.components.join(', ')}`);
  }
}
