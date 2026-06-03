import { lstat, readdir } from 'fs/promises';
import { resolve } from 'path';
import { inspectSkillDirectory, type SkillAuthoringInspection } from './authoring.ts';
import {
  type PackagingImportReport,
  prepareSkillPackaging,
  type PackagingWarningDetail,
  type PreparePackagingResponse,
  validateSkillPackaging,
  type ValidatePackagingResponse,
} from './catalog.ts';

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
  workspacePath: string;
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
  workspacePath?: string;
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
    'workspacePath' | 'idempotencyKey' | 'warnings' | 'warningDetails' | 'importReport' | 'manifestDigest'
  >
): RegistryPackagingSessionSuccessResult {
  return {
    ...inspection,
    ok: true,
    sessionKind: 'registry',
    workspacePath: fields.workspacePath,
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
  fields: Pick<RegistryPackagingSessionFailureResult, 'workspacePath' | 'idempotencyKey'> & {
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
    ...(fields.workspacePath ? { workspacePath: fields.workspacePath } : {}),
    idempotencyKey: fields.idempotencyKey,
  };
}

function externalErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function isStringArray(value: unknown): value is string[] {
  return Array.isArray(value) && value.every((item) => typeof item === 'string');
}

function isWarningDetailsArray(value: unknown): value is PackagingWarningDetail[] {
  return (
    Array.isArray(value) &&
    value.every(
      (item) =>
        item &&
        typeof item === 'object' &&
        typeof (item as PackagingWarningDetail).code === 'string' &&
        ((item as PackagingWarningDetail).severity === undefined ||
          typeof (item as PackagingWarningDetail).severity === 'string') &&
        typeof (item as PackagingWarningDetail).title === 'string' &&
        typeof (item as PackagingWarningDetail).message === 'string' &&
        ((item as PackagingWarningDetail).suggestedAction === undefined ||
          typeof (item as PackagingWarningDetail).suggestedAction === 'string')
    )
  );
}

function isImportReport(value: unknown): value is PackagingImportReport {
  return (
    !!value &&
    typeof value === 'object' &&
    ((value as PackagingImportReport).sourceType === undefined ||
      typeof (value as PackagingImportReport).sourceType === 'string') &&
    typeof (value as PackagingImportReport).strictMode === 'boolean' &&
    typeof (value as PackagingImportReport).normalized === 'boolean' &&
    typeof (value as PackagingImportReport).generatedPackageJson === 'boolean' &&
    typeof (value as PackagingImportReport).generatedCapabilityPlaceholder === 'boolean' &&
    typeof (value as PackagingImportReport).rewrotePackageIdentity === 'boolean' &&
    ((value as PackagingImportReport).generatedPaths === undefined ||
      isStringArray((value as PackagingImportReport).generatedPaths)) &&
    ((value as PackagingImportReport).warningCodes === undefined ||
      isStringArray((value as PackagingImportReport).warningCodes))
  );
}

function isPreparePackagingResponse(value: unknown): value is PreparePackagingResponse {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const candidate = value as {
    workspacePath?: unknown;
    packageName?: unknown;
    version?: unknown;
    warnings?: unknown;
    warningDetails?: unknown;
    importReport?: unknown;
    errors?: unknown;
  };
  return (
    typeof candidate.workspacePath === 'string' &&
    typeof candidate.packageName === 'string' &&
    typeof candidate.version === 'string' &&
    isStringArray(candidate.warnings) &&
    (candidate.warningDetails === undefined || isWarningDetailsArray(candidate.warningDetails)) &&
    (candidate.importReport === undefined || isImportReport(candidate.importReport)) &&
    isStringArray(candidate.errors)
  );
}

function isValidatePackagingResponse(value: unknown): value is ValidatePackagingResponse {
  if (!isPreparePackagingResponse(value)) {
    return false;
  }

  const candidate = value as {
    ok?: unknown;
    manifestDigest?: unknown;
    components?: unknown;
  };
  return (
    typeof candidate.ok === 'boolean' &&
    (candidate.manifestDigest === undefined || typeof candidate.manifestDigest === 'string') &&
    (candidate.components === undefined || isStringArray(candidate.components))
  );
}

async function validateLocalWorkspace(directory: string): Promise<string[]> {
  const capabilityRoots = new Set(['commands', 'agents', 'hooks', 'scripts', 'docs']);
  const capabilityFiles = new Set(['plugin.json', '.mcp.json']);
  const lowercasePaths = new Map<string, string>();
  let nestedSkill = false;
  let hasCapability = false;
  let unsafePathEntry = false;

  async function walk(currentDir: string, relativePrefix = ''): Promise<void> {
    const entries = await readdir(currentDir, { withFileTypes: true });

    for (const entry of entries) {
      const relativePath = relativePrefix ? `${relativePrefix}/${entry.name}` : entry.name;
      const stats = await lstat(`${currentDir}/${entry.name}`);

      if (stats.isSymbolicLink()) {
        unsafePathEntry = true;
        continue;
      }

      const lowercasePath = relativePath.toLowerCase();
      const existing = lowercasePaths.get(lowercasePath);
      if (existing && existing !== relativePath) {
        unsafePathEntry = true;
      } else {
        lowercasePaths.set(lowercasePath, relativePath);
      }

      if (entry.isDirectory()) {
        if (capabilityRoots.has(relativePath.split('/')[0] || '')) {
          hasCapability = true;
        }
        await walk(`${currentDir}/${entry.name}`, relativePath);
        continue;
      }

      if (relativePath.endsWith('/SKILL.md')) {
        nestedSkill = true;
      }
      if (capabilityFiles.has(relativePath)) {
        hasCapability = true;
      }
    }
  }

  await walk(directory);

  const errors: string[] = [];
  if (unsafePathEntry) {
    errors.push('unsafe_path_entry');
  }
  if (nestedSkill) {
    errors.push('nested_skill_md');
  }
  if (!hasCapability) {
    errors.push('missing_capability_content');
  }
  return errors;
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
  let prepared: PreparePackagingResponse;
  try {
    prepared = await prepareSkillPackaging({
      registry: options.registry,
      idempotencyKey,
      source: {
        type: 'local',
        location: directory,
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
  } catch (error) {
    return registryFailureResult(inspection, {
      errors: [externalErrorMessage(error)],
      idempotencyKey,
    });
  }

  if (!isPreparePackagingResponse(prepared)) {
    return registryFailureResult(inspection, {
      errors: ['catalog prepare returned malformed success payload'],
      idempotencyKey,
    });
  }

  if (prepared.errors.length > 0) {
    return registryFailureResult(inspection, {
      errors: prepared.errors,
      workspacePath: prepared.workspacePath,
      idempotencyKey,
    });
  }

  let validated: ValidatePackagingResponse;
  try {
    validated = await validateSkillPackaging({
      registry: options.registry,
      idempotencyKey,
      workspacePath: prepared.workspacePath,
      target: {
        packageName: inspection.packageName,
        version: inspection.version,
      },
    });
  } catch (error) {
    return registryFailureResult(inspection, {
      errors: [externalErrorMessage(error)],
      workspacePath: prepared.workspacePath,
      idempotencyKey,
    });
  }

  if (!isValidatePackagingResponse(validated)) {
    return registryFailureResult(inspection, {
      errors: ['catalog validate returned malformed success payload'],
      workspacePath: prepared.workspacePath,
      idempotencyKey,
    });
  }

  if (!validated.ok || validated.errors.length > 0) {
    return registryFailureResult(inspection, {
      packageName: validated.packageName || inspection.packageName,
      version: validated.version || inspection.version,
      components: validated.components || inspection.components,
      errors: validated.errors.length > 0 ? validated.errors : ['packaging validation failed'],
      workspacePath: validated.workspacePath,
      idempotencyKey,
    });
  }

  return registrySuccessResult(
    {
      ...inspection,
      packageName: validated.packageName || inspection.packageName,
      version: validated.version || inspection.version,
      components: validated.components || inspection.components,
      errors: validated.errors,
    },
    {
      workspacePath: validated.workspacePath,
      idempotencyKey,
      warnings: prepared.warnings,
      warningDetails: prepared.warningDetails,
      importReport: validated.importReport ?? prepared.importReport,
      manifestDigest: validated.manifestDigest,
    }
  );
}
