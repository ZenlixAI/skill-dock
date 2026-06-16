import { resolve } from 'path';
import { inspectSkillDirectory, type SkillAuthoringInspection } from './authoring.ts';
import {
  importReviewPackaging,
  type PackagingImportReport,
  type PackagingWarningDetail,
} from './catalog.ts';
import { createSourceArchiveBase64 } from './source-archive.ts';
import { validateLocalWorkspace } from './local-packaging.ts';

export interface PackagingSessionOptions {
  path?: string;
  registry?: string;
  strict?: boolean;
}

export type PackagingSessionKind = 'local' | 'registry';

export interface LocalPackagingSessionSuccessResult extends SkillAuthoringInspection {
  ok: true;
  sessionKind: 'local';
}

export interface RegistryPackagingSessionSuccessResult extends SkillAuthoringInspection {
  ok: true;
  sessionKind: 'registry';
  idempotencyKey: string;
  warnings?: string[];
  warningDetails?: PackagingWarningDetail[];
  importReport?: PackagingImportReport;
  manifestDigest?: string;
}

export interface LocalPackagingSessionFailureResult extends SkillAuthoringInspection {
  ok: false;
  sessionKind: 'local';
}

export interface RegistryPackagingSessionFailureResult extends SkillAuthoringInspection {
  ok: false;
  sessionKind: 'registry';
  idempotencyKey: string;
}

export type PackagingSessionResult =
  | LocalPackagingSessionSuccessResult
  | RegistryPackagingSessionSuccessResult
  | LocalPackagingSessionFailureResult
  | RegistryPackagingSessionFailureResult;

function packagingIdempotencyKey(): string {
  return `packaging-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function localSuccessResult(
  inspection: SkillAuthoringInspection
): LocalPackagingSessionSuccessResult {
  return {
    ...inspection,
    ok: true,
    sessionKind: 'local',
  };
}

function registrySuccessResult(
  inspection: SkillAuthoringInspection,
  fields: Pick<
    RegistryPackagingSessionSuccessResult,
    'idempotencyKey' | 'warnings' | 'warningDetails' | 'importReport' | 'manifestDigest'
  >
): RegistryPackagingSessionSuccessResult {
  return {
    ...inspection,
    ok: true,
    sessionKind: 'registry',
    idempotencyKey: fields.idempotencyKey,
    warnings: fields.warnings,
    warningDetails: fields.warningDetails,
    importReport: fields.importReport,
    ...(fields.manifestDigest ? { manifestDigest: fields.manifestDigest } : {}),
  };
}

function localFailureResult(
  inspection: SkillAuthoringInspection,
  errors: string[] = inspection.errors
): LocalPackagingSessionFailureResult {
  return {
    ...inspection,
    ok: false,
    sessionKind: 'local',
    errors,
  };
}

function registryFailureResult(
  inspection: SkillAuthoringInspection,
  fields: Pick<RegistryPackagingSessionFailureResult, 'idempotencyKey'> & {
    errors: string[];
    packageName?: string;
    version?: string;
    components?: string[];
  }
): RegistryPackagingSessionFailureResult {
  return {
    ...inspection,
    ok: false,
    sessionKind: 'registry',
    ...(fields.packageName ?? inspection.packageName
      ? { packageName: fields.packageName ?? inspection.packageName }
      : {}),
    ...(fields.version ?? inspection.version
      ? { version: fields.version ?? inspection.version }
      : {}),
    components: fields.components ?? inspection.components,
    errors: fields.errors,
    idempotencyKey: fields.idempotencyKey,
  };
}

function externalErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

export async function runPackagingSession(
  options: PackagingSessionOptions = {}
): Promise<PackagingSessionResult> {
  const directory = resolve(options.path || process.cwd());
  const inspection = await inspectSkillDirectory(directory);

  if (!inspection.ok || !inspection.packageName || !inspection.version) {
    return localFailureResult(inspection);
  }

  if (!options.registry) {
    const localValidationErrors = await validateLocalWorkspace(directory);
    if (localValidationErrors.length > 0) {
      return localFailureResult(inspection, localValidationErrors);
    }
    return localSuccessResult(inspection);
  }

  const idempotencyKey = packagingIdempotencyKey();
  try {
    const review = await importReviewPackaging({
      registry: options.registry,
      idempotencyKey,
      source: {
        type: 'upload_archive',
        archiveBase64: createSourceArchiveBase64(directory),
      },
      target: {
        packageName: inspection.packageName,
        version: inspection.version,
      },
      metadata: {
        displayName: inspection.displayName,
        description: inspection.description,
        sourceType: inspection.sourceType || 'native',
      },
      options: {
        ...(inspection.publishRegistry ? { publishRegistry: inspection.publishRegistry } : {}),
        ...(options.strict ? { strict: true } : {}),
      },
    });

    const reviewedComponents = review.components ?? inspection.components;
    if (!review.ok || review.errors.length > 0) {
      return registryFailureResult(inspection, {
        packageName: review.packageName || inspection.packageName,
        version: review.version || inspection.version,
        components: reviewedComponents,
        errors: review.errors.length > 0 ? review.errors : ['packaging import review failed'],
        idempotencyKey,
      });
    }

    return registrySuccessResult(
      {
        ...inspection,
        packageName: review.packageName || inspection.packageName,
        version: review.version || inspection.version,
        components: reviewedComponents,
        errors: review.errors,
      },
      {
        idempotencyKey,
        warnings: review.warnings,
        warningDetails: review.warningDetails,
        importReport: review.importReport,
        manifestDigest: review.manifestDigest,
      }
    );
  } catch (error) {
    return registryFailureResult(inspection, {
      errors: [externalErrorMessage(error)],
      idempotencyKey,
    });
  }
}
