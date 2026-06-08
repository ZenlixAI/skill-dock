import { spawnSync } from 'child_process';
import { resolve } from 'path';
import {
  getRegistryBaseUrl,
  putReleaseRecord,
  type PackagingWarningDetail,
} from './catalog.ts';
import type { PackagingImportReport } from './catalog.ts';
import { packLocalSkillDirectory } from './local-packaging.ts';
import { resolveNpmUserConfigPath } from './npm-auth.ts';
import { runPackagingSession } from './packaging-session.ts';
import { normalizeRegistryUrl } from './registry-url.ts';
import { ensurePublishedVersion } from './npm-publish-recovery.ts';

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
  manifestDigest: string;
  cleanup: () => void;
}

function resolvePublishUserConfigPath(directory: string): string | undefined {
  return process.env.NPM_CONFIG_USERCONFIG || resolveNpmUserConfigPath({ cwd: directory });
}

function publishBundle(directory: string, bundlePath: string, npmRegistry?: string): void {
  const args = ['publish', bundlePath];
  if (typeof npmRegistry === 'string' && npmRegistry.trim() !== '') {
    args.push('--registry', npmRegistry.trim());
  }
  const userConfigPath = resolvePublishUserConfigPath(directory);
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
    strict: options.strict,
  });

  if (!session.ok || !session.packageName || !session.version) {
    return failure(session.errors);
  }

  try {
    const npmPublishRegistry =
      session.publishRegistry?.trim() ||
      process.env.npm_config_registry ||
      process.env.NPM_CONFIG_REGISTRY ||
      undefined;
    const bundle: PackedSkill = await packLocalSkillDirectory({
      directory,
    });
    let retainBundleUntilExit = false;
    try {
      await ensurePublishedVersion({
        directory,
        packageName: session.packageName,
        version: session.version,
        npmRegistry: npmPublishRegistry,
        publish: () => publishBundle(directory, bundle.bundlePath, npmPublishRegistry),
      });

      await putReleaseRecord(session.packageName, session.version, {
        registry,
        publishRegistry: normalizeRegistryUrl(session.publishRegistry),
        manifestDigest: bundle.manifestDigest,
        components: session.components,
      });

      retainBundleUntilExit = true;
      process.once('exit', bundle.cleanup);

      return {
        ok: true,
        packageName: session.packageName,
        version: session.version,
        bundlePath: bundle.bundlePath,
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
