import { downloadAndExtractNpmPackage, fetchNpmPackageSelection } from './npm-registry.ts';
import { runImportLocalWorkflow, type ImportLocalWorkflowResult } from './import-local-workflow.ts';

export interface ImportNpmWorkflowOptions {
  packageName: string;
  registry?: string;
  catalogRegistry?: string;
  version?: string;
  strict?: boolean;
  hosted?: boolean;
}

export interface ImportNpmWorkflowSuccessResult extends Extract<ImportLocalWorkflowResult, { ok: true }> {
  sourceRegistry: string;
  importMode: 'mirror' | 'hosted';
}

export type ImportNpmWorkflowResult =
  | ImportNpmWorkflowSuccessResult
  | Extract<ImportLocalWorkflowResult, { ok: false }>;

export async function runImportNpmWorkflow(
  options: ImportNpmWorkflowOptions
): Promise<ImportNpmWorkflowResult> {
  const selection = await fetchNpmPackageSelection(options.packageName, {
    registry: options.registry,
    version: options.version,
  });
  const downloaded = await downloadAndExtractNpmPackage(selection, { cwd: process.cwd() });

  try {
    const result = await runImportLocalWorkflow({
      path: downloaded.extractedPath,
      registry: options.catalogRegistry || process.env.SKILLS_REGISTRY_URL,
      strict: options.strict,
      packageName: selection.packageName,
      version: selection.version,
    });

    if (!result.ok) {
      return result;
    }

    return {
      ...result,
      sourceRegistry: selection.registryUrl,
      importMode: options.hosted ? 'hosted' : 'mirror',
    };
  } finally {
    await downloaded.cleanup();
  }
}
