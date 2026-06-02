import { addSkillToLocalLock, computeSkillFolderHash } from './local-lock.ts';
import { addSkillToLock } from './skill-lock.ts';
import { getRegistryBaseUrl, resolveInstallTarget, type InstallTarget } from './catalog.ts';
import { downloadRegistrySkillBundle } from './registry-bundle.ts';
import { installSkillForAgent, type InstallMode } from './installer.ts';
import type { AgentType } from './types.ts';

export interface RegistryInstallWorkflowOptions {
  packageName: string;
  version?: string;
  registry?: string;
  targetAgents: AgentType[];
  installGlobally: boolean;
  installMode: InstallMode;
  cwd?: string;
}

export interface RegistryInstallWorkflowResult {
  packageName: string;
  target: InstallTarget;
  successful: Array<{
    agent: AgentType;
    result: {
      success: true;
      path: string;
      canonicalPath?: string;
      mode: InstallMode;
      symlinkFailed?: boolean;
      skipped?: boolean;
    };
  }>;
  failed: Array<{
    agent: AgentType;
    result: {
      success: false;
      path: string;
      mode: InstallMode;
      symlinkFailed?: boolean;
      skipped?: boolean;
      error?: string;
    };
  }>;
}

export async function runRegistryInstallWorkflow(
  options: RegistryInstallWorkflowOptions
): Promise<RegistryInstallWorkflowResult> {
  const cwd = options.cwd || process.cwd();
  const target = await resolveInstallTarget(options.packageName, {
    registry: options.registry,
    version: options.version,
  });
  const bundle = await downloadRegistrySkillBundle(target);

  try {
    const results = await Promise.all(
      options.targetAgents.map(async (agent) => ({
        agent,
        result: await installSkillForAgent(bundle.skill, agent, {
          global: options.installGlobally,
          cwd,
          mode: options.installMode,
        }),
      }))
    );

    const successful = results.filter(
      (entry): entry is RegistryInstallWorkflowResult['successful'][number] => entry.result.success
    );
    const failed = results.filter(
      (entry): entry is RegistryInstallWorkflowResult['failed'][number] => !entry.result.success
    );

    if (successful.length === 0) {
      throw new Error(
        failed[0]?.result.error ||
          `Failed to install ${options.packageName} for the selected agents`
      );
    }

    const installedSkillFolderHash = await computeSkillFolderHash(bundle.extractedPath);

    if (options.installGlobally) {
      await addSkillToLock(options.packageName, {
        sourceType: 'registry',
        skillFolderHash: installedSkillFolderHash,
        packageName: options.packageName,
        version: target.version,
        registry: getRegistryBaseUrl(options.registry),
        integrity: target.integrity,
        skillManifestDigest: target.skillManifestDigest,
      });
    } else {
      await addSkillToLocalLock(
        options.packageName,
        {
          sourceType: 'registry',
          computedHash: installedSkillFolderHash,
          packageName: options.packageName,
          version: target.version,
          registry: getRegistryBaseUrl(options.registry),
          integrity: target.integrity,
          skillManifestDigest: target.skillManifestDigest,
        },
        cwd
      );
    }

    return {
      packageName: options.packageName,
      target,
      successful,
      failed,
    };
  } finally {
    await bundle.cleanup();
  }
}
