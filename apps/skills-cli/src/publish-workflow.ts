import { resolve } from 'path';
import {
  getRegistryBaseUrl,
  publishSkillPackaging,
  type PackagingImportReport,
  type PackagingWarningDetail,
} from './catalog.ts';
import { packLocalSkillDirectory } from './local-packaging.ts';
import { runPackagingSession } from './packaging-session.ts';
import { createSourceArchiveBase64 } from './source-archive.ts';

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
    registry: options.registry,
    strict: options.strict,
  });

  if (!session.ok || !session.packageName || !session.version) {
    return failure(session.errors);
  }

  try {
    const bundle: PackedSkill = await packLocalSkillDirectory({
      directory,
    });
    let retainBundleUntilExit = false;
    try {
      const publishPayload = await publishSkillPackaging({
        registry,
        source: {
          type: 'upload_archive',
          archiveBase64: createSourceArchiveBase64(directory),
        },
        target: {
          packageName: session.packageName,
          version: session.version,
        },
        metadata: {
          displayName: session.displayName || '',
          description: session.description || '',
          sourceType: session.sourceType || 'native',
        },
        options: {
          ...(options.strict ? { strict: true } : {}),
        },
        review: {
          manifestDigest: bundle.manifestDigest,
          components: session.components,
        },
      });

      if (publishPayload.errors.length > 0) {
        return failure(publishPayload.errors);
      }

      retainBundleUntilExit = true;
      process.once('exit', bundle.cleanup);

      return {
        ok: true,
        packageName: publishPayload.packageName,
        version: publishPayload.version,
        bundlePath: bundle.bundlePath,
        ...(publishPayload.warnings.length > 0 ? { warnings: publishPayload.warnings } : {}),
        ...(publishPayload.warningDetails ? { warningDetails: publishPayload.warningDetails } : {}),
        ...(publishPayload.importReport ? { importReport: publishPayload.importReport } : {}),
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
