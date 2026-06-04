import { addSkillToLocalLock, computeSkillFolderHash, readLocalLock } from './local-lock.ts';
import { addSkillToLock, readSkillLock } from './skill-lock.ts';
import { getRegistryBaseUrl, resolveInstallTarget, type InstallTarget } from './catalog.ts';
import { downloadRegistrySkillBundle } from './registry-bundle.ts';
import {
  deriveInstallName,
  installSkillForAgent,
  sanitizeName,
  type InstallMode,
} from './installer.ts';
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

function buildFallbackInstallName(packageName: string): string {
  return sanitizeName(packageName);
}

async function resolveRegistryInstallName(
  packageName: string,
  installGlobally: boolean,
  cwd: string
): Promise<string> {
  const preferredInstallName = deriveInstallName(packageName);
  const preferredSanitized = sanitizeName(preferredInstallName);
  const fallbackInstallName = buildFallbackInstallName(packageName);
  const lock = installGlobally ? await readSkillLock() : await readLocalLock(cwd);
  const existingEntry = lock.skills[packageName];

  if (existingEntry?.installName && existingEntry.installName.trim().length > 0) {
    return existingEntry.installName;
  }

  const hasConflict = Object.entries(lock.skills).some(([lockedPackageName, entry]) => {
    if (lockedPackageName === packageName) {
      return false;
    }
    const candidateInstallName =
      entry.installName || deriveInstallName(entry.packageName || lockedPackageName);
    return sanitizeName(candidateInstallName) === preferredSanitized;
  });

  if (hasConflict) {
    return fallbackInstallName;
  }

  return preferredInstallName;
}

export async function runRegistryInstallWorkflow(
  options: RegistryInstallWorkflowOptions
): Promise<RegistryInstallWorkflowResult> {
  const cwd = options.cwd || process.cwd();
  const target = await resolveInstallTarget(options.packageName, {
    registry: options.registry,
    version: options.version,
  });
  const bundle = await downloadRegistrySkillBundle(target, { cwd });

  try {
    bundle.skill.installName = await resolveRegistryInstallName(
      options.packageName,
      options.installGlobally,
      cwd
    );

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
    const installName = bundle.skill.installName || deriveInstallName(options.packageName);

    if (options.installGlobally) {
      await addSkillToLock(options.packageName, {
        sourceType: 'registry',
        skillFolderHash: installedSkillFolderHash,
        packageName: options.packageName,
        installName,
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
          installName,
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
