import { existsSync } from 'fs';
import { readFile } from 'fs/promises';
import { basename, join } from 'path';
import { inspectImportSourceDirectory } from './import-source.ts';
import { runImportLocalWorkflow } from './import-local-workflow.ts';
import { sanitizeName } from './installer.ts';
import { isExactVersion } from './authoring.ts';

export interface AddSyncOptions {
  cwd?: string;
  registry?: string;
  nameHint?: string;
}

interface AddSyncConfig {
  syncOnAdd?: boolean;
  packageNamePrefix?: string;
  defaultVersion?: string;
  registry?: string;
}

const CONFIG_FILE = 'skills.config.json';

function resolveConfigPath(cwd?: string): string {
  return join(cwd || process.cwd(), CONFIG_FILE);
}

async function readAddSyncConfig(cwd?: string): Promise<AddSyncConfig | null> {
  const configPath = resolveConfigPath(cwd);
  if (!existsSync(configPath)) {
    return null;
  }

  try {
    return JSON.parse(await readFile(configPath, 'utf-8')) as AddSyncConfig;
  } catch {
    return null;
  }
}

function normalizePrefix(value?: string): string | undefined {
  const trimmed = value?.trim();
  if (!trimmed) {
    return undefined;
  }
  return trimmed.endsWith('/') ? trimmed : `${trimmed}/`;
}

function resolvePackageName(
  inspection: Awaited<ReturnType<typeof inspectImportSourceDirectory>>,
  config: AddSyncConfig | null,
  nameHint?: string
): string | undefined {
  if (inspection.packageName) {
    return inspection.packageName;
  }

  const prefix = normalizePrefix(config?.packageNamePrefix);
  if (!prefix) {
    return undefined;
  }

  const hint = inspection.displayName?.trim() || nameHint?.trim() || basename(inspection.directory);
  return `${prefix}${sanitizeName(hint || basename(inspection.directory))}`;
}

function resolveVersion(
  inspection: Awaited<ReturnType<typeof inspectImportSourceDirectory>>,
  config: AddSyncConfig | null
): string | undefined {
  if (inspection.version) {
    return inspection.version;
  }

  const configuredVersion = config?.defaultVersion?.trim();
  if (!configuredVersion || !isExactVersion(configuredVersion)) {
    return undefined;
  }

  return configuredVersion;
}

function resolveRegistry(config: AddSyncConfig | null, override?: string): string | undefined {
  const trimmedOverride = override?.trim();
  if (trimmedOverride) {
    return trimmedOverride;
  }

  const trimmedConfig = config?.registry?.trim();
  return trimmedConfig ? trimmedConfig : undefined;
}

export async function syncAddedSkillToBackend(
  skillPath: string,
  options: AddSyncOptions = {}
): Promise<string | null> {
  const config = await readAddSyncConfig(options.cwd);
  if (config?.syncOnAdd === false) {
    return null;
  }

  const inspection = await inspectImportSourceDirectory(skillPath);
  const packageName = resolvePackageName(inspection, config, options.nameHint);
  const version = resolveVersion(inspection, config);

  if (!packageName) {
    return 'Set skills.config.json packageNamePrefix or add a package.json name before backend sync.';
  }
  if (!version) {
    return 'Set skills.config.json defaultVersion or add a package.json version before backend sync.';
  }

  const result = await runImportLocalWorkflow({
    path: skillPath,
    registry: resolveRegistry(config, options.registry),
    packageName,
    version,
  });

  if (!result.ok) {
    return result.errors.join('; ');
  }

  return null;
}
