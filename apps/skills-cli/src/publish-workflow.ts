import { spawnSync } from 'child_process';
import { mkdtempSync, rmSync, writeFileSync } from 'fs';
import { tmpdir } from 'os';
import { basename, join, resolve } from 'path';
import {
  getRegistryBaseUrl,
  packSkillPackaging,
  type PackagingImportReport,
  putReleaseRecord,
  type PackagingWarningDetail,
  type PackPackagingResponse,
} from './catalog.ts';
import { resolveNpmUserConfigPath } from './npm-auth.ts';
import { runPackagingSession } from './packaging-session.ts';

export interface PublishWorkflowOptions {
  path?: string;
  registry?: string;
  strict?: boolean;
}

export interface PublishWorkflowSuccessResult {
  ok: true;
  packageName: string;
  version: string;
  bundlePath: string;
  warnings?: string[];
  warningDetails?: PackagingWarningDetail[];
  importReport?: PackagingImportReport;
}

export interface PublishWorkflowFailureResult {
  ok: false;
  errors: string[];
}

export type PublishWorkflowResult = PublishWorkflowSuccessResult | PublishWorkflowFailureResult;

interface PackedSkill {
  bundlePath: string;
  cleanup: () => void;
}

function materializePackedBundle(packed: PackPackagingResponse): PackedSkill {
  if (!packed.bundleBase64) {
    throw new Error('packaging API did not return bundle content');
  }

  const tempDir = mkdtempSync(join(tmpdir(), 'skills-publish-'));
  const bundleName =
    basename(packed.bundlePath) ||
    `${packed.packageName.replace(/^@/, '').replace(/\//g, '-')}-${packed.version}.tgz`;
  const bundlePath = join(tempDir, bundleName);

  writeFileSync(bundlePath, Buffer.from(packed.bundleBase64, 'base64'));

  return {
    bundlePath,
    cleanup: () => rmSync(tempDir, { recursive: true, force: true }),
  };
}

function publishBundle(directory: string, bundlePath: string): void {
  const result = spawnSync('npm', ['publish', bundlePath], {
    cwd: directory,
    stdio: ['inherit', 'pipe', 'pipe'],
    encoding: 'utf-8',
    env: {
      ...process.env,
      ...(resolveNpmUserConfigPath({ cwd: directory })
        ? { NPM_CONFIG_USERCONFIG: resolveNpmUserConfigPath({ cwd: directory }) }
        : {}),
    },
    shell: process.platform === 'win32',
  });

  if (result.status !== 0) {
    throw new Error(result.stderr || result.stdout || 'npm publish failed');
  }
}

function failure(errors: string[]): PublishWorkflowFailureResult {
  return { ok: false, errors };
}

export async function runPublishWorkflow(
  options: PublishWorkflowOptions = {}
): Promise<PublishWorkflowResult> {
  const directory = resolve(options.path || process.cwd());
  const registry = getRegistryBaseUrl(options.registry);
  const session = await runPackagingSession({
    path: options.path,
    registry,
    strict: options.strict,
  });

  if (!session.ok || session.sessionKind !== 'registry' || !session.packageName || !session.version) {
    return failure(session.errors);
  }

  try {
    const packed = await packSkillPackaging({
      registry,
      idempotencyKey: session.idempotencyKey,
      workspacePath: session.workspacePath,
      target: {
        packageName: session.packageName,
        version: session.version,
      },
      includeBundle: true,
    });
    if (packed.errors.length > 0) {
      return failure(packed.errors);
    }

    const bundle = materializePackedBundle(packed);
    let retainBundleUntilExit = false;
    try {
      publishBundle(directory, bundle.bundlePath);

      await putReleaseRecord(session.packageName, session.version, {
        registry,
        publishRegistry: session.publishRegistry,
        manifestDigest: packed.manifestDigest,
        components: packed.components,
      });

      retainBundleUntilExit = true;
      process.once('exit', bundle.cleanup);

      return {
        ok: true,
        packageName: session.packageName,
        version: session.version,
        bundlePath: bundle.bundlePath,
        ...(session.warnings ? { warnings: session.warnings } : {}),
        ...(session.warningDetails ? { warningDetails: session.warningDetails } : {}),
        ...(session.importReport ? { importReport: session.importReport } : {}),
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
