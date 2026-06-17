import { cleanupTempDir } from './git.ts';
import { resolveExternalSourceSkills } from './external-source.ts';
import { runImportLocalWorkflow, type ImportLocalWorkflowResult } from './import-local-workflow.ts';
import { parseSourceInput } from './source-parser.ts';

export interface ImportGitWorkflowOptions {
  source: string;
  registry?: string;
  strict?: boolean;
  packageName?: string;
  version?: string;
  publishRegistry?: string;
  skill?: string[];
  fullDepth?: boolean;
}

function failure(errors: string[]): ImportLocalWorkflowResult {
  return { ok: false, errors };
}

export async function runImportGitWorkflow(
  options: ImportGitWorkflowOptions
): Promise<ImportLocalWorkflowResult> {
  const parsedSource = parseSourceInput(options.source);

  if (!parsedSource || parsedSource.type === 'local') {
    return failure(['Unsupported source. Use owner/repo or a Git URL for external imports.']);
  }

  let cleanupRoot: string | null = null;

  try {
    const resolved = await resolveExternalSourceSkills(parsedSource, {
      requestedSkillNames: options.skill,
      fullDepth: options.fullDepth,
    });
    cleanupRoot = resolved.cleanupRoot;

    if (resolved.matchedSkills.length > 1) {
      return failure([
        `Multiple skills found. Pass --skill <name> to choose one: ${resolved.matchedSkills.map((skill) => skill.name).join(', ')}`,
      ]);
    }

    return await runImportLocalWorkflow({
      path: resolved.matchedSkills[0]!.path,
      registry: options.registry,
      strict: options.strict,
      packageName: options.packageName,
      version: options.version,
      publishRegistry: options.publishRegistry,
    });
  } catch (error) {
    return failure([error instanceof Error ? error.message : String(error)]);
  } finally {
    if (cleanupRoot) {
      await cleanupTempDir(cleanupRoot).catch(() => {});
    }
  }
}
