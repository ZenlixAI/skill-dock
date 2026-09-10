import { existsSync } from 'fs';
import { readFile } from 'fs/promises';
import { homedir } from 'os';
import { basename, dirname, join, resolve } from 'path';
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

const PROJECT_CONFIG_FILE = 'skills.config.json';
const GLOBAL_CONFIG_FILE = 'config.json';

function resolveGlobalConfigPath(): string {
  const configHome = process.env.XDG_CONFIG_HOME?.trim() || join(homedir(), '.config');
  return join(configHome, 'skills', GLOBAL_CONFIG_FILE);
}

function findProjectConfigPath(cwd: string): string | null {
  let directory = resolve(cwd);

  while (true) {
    const configPath = join(directory, PROJECT_CONFIG_FILE);
    if (existsSync(configPath)) {
      return configPath;
    }

    const parent = dirname(directory);
    if (parent === directory) {
      return null;
    }
    directory = parent;
  }
}

async function readConfigFile(configPath: string | null): Promise<AddSyncConfig | null> {
  if (!configPath || !existsSync(configPath)) {
    return null;
  }

  try {
    return JSON.parse(await readFile(configPath, 'utf-8')) as AddSyncConfig;
  } catch {
    return null;
  }
}

async function readAddSyncConfig(cwd?: string): Promise<AddSyncConfig | null> {
  const workingDirectory = cwd || process.cwd();
  const [globalConfig, projectConfig] = await Promise.all([
    readConfigFile(resolveGlobalConfigPath()),
    readConfigFile(findProjectConfigPath(workingDirectory)),
  ]);

  if (!globalConfig && !projectConfig) {
    return null;
  }

  return { ...globalConfig, ...projectConfig };
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
