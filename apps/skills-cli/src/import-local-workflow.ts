import { spawnSync } from 'child_process';
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'fs';
import { tmpdir } from 'os';
import { basename, join, resolve } from 'path';
import {
  getRegistryBaseUrl,
  packSkillPackaging,
  prepareSkillPackaging,
  putReleaseRecord,
  type PackagingImportReport,
  type PackagingWarningDetail,
  validateSkillPackaging,
} from './catalog.ts';
import { hasSkillPackageScope, isExactVersion, isPackageName } from './authoring.ts';
import { inspectImportSourceDirectory } from './import-source.ts';
import { resolveNpmUserConfigPath } from './npm-auth.ts';
import { normalizeRegistryUrl } from './registry-url.ts';
import { createSourceArchiveBase64 } from './source-archive.ts';

export interface ImportLocalWorkflowOptions {
  path?: string;
  registry?: string;
  strict?: boolean;
  packageName?: string;
  version?: string;
  publishRegistry?: string;
  npmRegistry?: string;
}

export interface ImportLocalWorkflowSuccessResult {
  ok: true;
  packageName: string;
  version: string;
  bundlePath: string;
  warnings?: string[];
  warningDetails?: PackagingWarningDetail[];
  importReport?: PackagingImportReport;
}

export interface ImportLocalWorkflowFailureResult {
  ok: false;
  errors: string[];
}

export type ImportLocalWorkflowResult =
  | ImportLocalWorkflowSuccessResult
  | ImportLocalWorkflowFailureResult;

interface PackedBundle {
  bundlePath: string;
  manifestDigest: string;
  components: string[];
  cleanup: () => void;
}

function failure(errors: string[]): ImportLocalWorkflowFailureResult {
  return { ok: false, errors };
}

function resolveReleaseRecordPublishRegistry(options: ImportLocalWorkflowOptions, inspectedPublishRegistry?: string): string | undefined {
  return normalizeRegistryUrl(options.publishRegistry) || normalizeRegistryUrl(inspectedPublishRegistry) || undefined;
}

function packageNameIsValid(value: string): boolean {
  return isPackageName(value) && hasSkillPackageScope(value);
}

function writePackedBundleToTemp(
  packageName: string,
  version: string,
  bundleBase64: string
): PackedBundle {
  const tempDir = mkdtempSync(join(tmpdir(), 'skills-import-local-'));
  const bundleName =
    `${basename(packageName).replace(/^@/, '').replace(/\//g, '-')}-${version}.tgz`;
  const bundlePath = join(tempDir, bundleName);
  writeFileSync(bundlePath, Buffer.from(bundleBase64, 'base64'));
  return {
    bundlePath,
    manifestDigest: '',
    components: [],
    cleanup: () => rmSync(tempDir, { recursive: true, force: true }),
  };
}

function publishBundle(directory: string, bundlePath: string, npmRegistry?: string): void {
  const args = ['publish', bundlePath];
  if (typeof npmRegistry === 'string' && npmRegistry.trim() !== '') {
    args.push('--registry', npmRegistry.trim());
  }
  const userConfigPath = process.env.NPM_CONFIG_USERCONFIG || resolveNpmUserConfigPath({ cwd: directory });
  const result = spawnSync('npm', args, {
    cwd: directory,
    stdio: ['inherit', 'pipe', 'pipe'],
    encoding: 'utf-8',
    env: {
      ...process.env,
      ...(userConfigPath ? { NPM_CONFIG_USERCONFIG: userConfigPath } : {}),
    },
    shell: process.platform === 'win32',
  });

  if (result.status !== 0) {
    throw new Error(result.stderr || result.stdout || 'npm publish failed');
  }
}

export async function runImportLocalWorkflow(
  options: ImportLocalWorkflowOptions = {}
): Promise<ImportLocalWorkflowResult> {
  const directory = resolve(options.path || process.cwd());
  const registry = getRegistryBaseUrl(options.registry);
  const inspection = await inspectImportSourceDirectory(directory);

  const packageName = options.packageName || inspection.packageName;
  const version = options.version || inspection.version;
  const errors = inspection.errors.filter(
    (error) =>
      !(options.packageName && error === 'package.json name must use a scoped @<company>.skill/<skill-name> package name') &&
      !(options.version && error === 'package.json version must be an exact semver')
  );

  if (!packageName) {
    errors.push('packageName is required; pass --package-name when package.json is missing');
  } else if (!packageNameIsValid(packageName)) {
    errors.push('packageName must use a scoped @<company>.skill/<skill-name> package name');
  }

  if (!version) {
    errors.push('version is required; pass --version when package.json is missing');
  } else if (!isExactVersion(version)) {
    errors.push('version must be an exact semver');
  }

  if (!inspection.displayName || !inspection.description) {
    return failure(errors);
  }
  if (errors.length > 0) {
    return failure(errors);
  }
  const finalPackageName = packageName as string;
  const finalVersion = version as string;
  const finalDisplayName = inspection.displayName as string;
  const finalDescription = inspection.description as string;

  try {
    const publishRegistry = resolveReleaseRecordPublishRegistry(options, inspection.publishRegistry);
    const preparePayload = await prepareSkillPackaging({
      registry,
      source: {
        type: 'upload_archive',
        archiveBase64: createSourceArchiveBase64(directory),
      },
      target: {
        packageName: finalPackageName,
        version: finalVersion,
      },
      metadata: {
        displayName: finalDisplayName,
        description: finalDescription,
        sourceType: inspection.sourceType || 'imported',
      },
      options: {
        ...(publishRegistry ? { publishRegistry } : {}),
        ...(options.strict ? { strict: true } : {}),
      },
    });
    if (preparePayload.errors.length > 0) {
      return failure(preparePayload.errors);
    }

    const validatePayload = await validateSkillPackaging({
      registry,
      workspacePath: preparePayload.workspacePath,
      target: {
        packageName: finalPackageName,
        version: finalVersion,
      },
    });
    if (!validatePayload.ok || validatePayload.errors.length > 0) {
      return failure(validatePayload.errors.length > 0 ? validatePayload.errors : ['packaging validation failed']);
    }

    const packPayload = await packSkillPackaging({
      registry,
      workspacePath: preparePayload.workspacePath,
      target: {
        packageName: finalPackageName,
        version: finalVersion,
      },
      includeBundle: true,
    });
    if (packPayload.errors.length > 0) {
      return failure(packPayload.errors);
    }
    if (!packPayload.bundleBase64) {
      return failure(['packaging API did not return bundleBase64']);
    }

    const bundle = writePackedBundleToTemp(finalPackageName, finalVersion, packPayload.bundleBase64);
    bundle.manifestDigest = packPayload.manifestDigest;
    bundle.components = packPayload.components;
    let retainBundleUntilExit = false;
    try {
      publishBundle(directory, bundle.bundlePath, options.npmRegistry || publishRegistry);
      await putReleaseRecord(finalPackageName, finalVersion, {
        registry,
        publishRegistry,
        manifestDigest: packPayload.manifestDigest,
        components: packPayload.components,
      });
      retainBundleUntilExit = true;
      process.once('exit', bundle.cleanup);
      return {
        ok: true,
        packageName: finalPackageName,
        version: finalVersion,
        bundlePath: bundle.bundlePath,
        ...(preparePayload.warnings.length > 0 ? { warnings: preparePayload.warnings } : {}),
        ...(preparePayload.warningDetails ? { warningDetails: preparePayload.warningDetails } : {}),
        ...(preparePayload.importReport ? { importReport: preparePayload.importReport } : {}),
      };
    } finally {
      if (!retainBundleUntilExit) {
        bundle.cleanup();
      }
    }
  } catch (error) {
    return failure([error instanceof Error ? error.message : String(error)]);
  }
}
