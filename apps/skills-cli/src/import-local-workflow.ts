import {
  CatalogError,
  getRegistryBaseUrl,
  publishSkillPackaging,
  type PackagingImportReport,
  type PackagingWarningDetail,
} from './catalog.ts';
import { hasSkillPackageScope, isExactVersion, isPackageName } from './authoring.ts';
import { inspectImportSourceDirectory } from './import-source.ts';
import { normalizeRegistryUrl } from './registry-url.ts';
import { createSourceArchiveBase64 } from './source-archive.ts';

export interface ImportLocalWorkflowOptions {
  path?: string;
  registry?: string;
  strict?: boolean;
  packageName?: string;
  version?: string;
  publishRegistry?: string;
}

export interface ImportLocalWorkflowSuccessResult {
  ok: true;
  packageName: string;
  version: string;
  manifestDigest: string;
  components: string[];
  publishedToRegistry: boolean;
  created: boolean;
  idempotentReplay: boolean;
  versionStatus: string;
  installPolicy: string;
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

function failure(errors: string[]): ImportLocalWorkflowFailureResult {
  return { ok: false, errors };
}

function formatImportLocalError(error: unknown): string {
  if (error instanceof CatalogError) {
    const context = [`catalog_status=${error.status}`, `catalog_code=${error.code}`];
    if (error.requestId) {
      context.push(`catalog_request_id=${error.requestId}`);
    }
    if (error.details) {
      context.push(`catalog_details=${JSON.stringify(error.details)}`);
    }
    const advice =
      error.code === 'internal_error' && error.message === 'Failed to create packaging workspace.'
        ? ' Catalog write-side packaging workspace root is missing or not writable; check SKILL_REGISTRY_SHARED_WORKSPACE_ROOT and its mounted directory.'
        : '';
    return `${error.message} (${context.join(', ')})${advice}`;
  }

  return error instanceof Error ? error.message : String(error);
}

function resolvePublishRegistry(
  options: ImportLocalWorkflowOptions,
  inspectedPublishRegistry?: string
): string | undefined {
  return (
    normalizeRegistryUrl(options.publishRegistry) ||
    normalizeRegistryUrl(inspectedPublishRegistry) ||
    undefined
  );
}

function packageNameIsValid(value: string): boolean {
  return isPackageName(value) && hasSkillPackageScope(value);
}

export async function runImportLocalWorkflow(
  options: ImportLocalWorkflowOptions = {}
): Promise<ImportLocalWorkflowResult> {
  const directory = options.path || process.cwd();
  const registry = getRegistryBaseUrl(options.registry);
  const inspection = await inspectImportSourceDirectory(directory);

  const packageName = options.packageName || inspection.packageName;
  const version = options.version || inspection.version;
  const errors = inspection.errors.filter(
    (error) =>
      !(
        options.packageName &&
        error === 'package.json name must use a scoped @<company>.skill/<skill-name> package name'
      ) && !(options.version && error === 'package.json version must be an exact semver')
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
    const publishRegistry = resolvePublishRegistry(options, inspection.publishRegistry);
    const publishPayload = await publishSkillPackaging({
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

    if (publishPayload.errors.length > 0) {
      return failure(publishPayload.errors);
    }

    return {
      ok: true,
      packageName: publishPayload.packageName,
      version: publishPayload.version,
      manifestDigest: publishPayload.manifestDigest,
      components: publishPayload.components,
      publishedToRegistry: publishPayload.publishedToRegistry,
      created: publishPayload.created,
      idempotentReplay: publishPayload.idempotentReplay,
      versionStatus: publishPayload.versionStatus,
      installPolicy: publishPayload.installPolicy,
      ...(publishPayload.warnings.length > 0 ? { warnings: publishPayload.warnings } : {}),
      ...(publishPayload.warningDetails ? { warningDetails: publishPayload.warningDetails } : {}),
      ...(publishPayload.importReport ? { importReport: publishPayload.importReport } : {}),
    };
  } catch (error) {
    return failure([formatImportLocalError(error)]);
  }
}
