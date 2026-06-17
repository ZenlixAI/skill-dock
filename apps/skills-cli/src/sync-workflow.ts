import { join } from 'path';
import { cloneRepo, cleanupTempDir } from './git.ts';
import { listInstalledView, type InstalledSkillView, type InstalledViewScope } from './installed-view.ts';
import { installSkillForAgent } from './installer.ts';
import {
  addSkillToLocalLock,
  computeSkillFolderHash,
  type LocalSkillLegacyMetadata,
} from './local-lock.ts';
import { addSkillToLock, type SkillLockLegacyMetadata } from './skill-lock.ts';
import { discoverSkills } from './skills.ts';
import { getRecordedSkillPath, parseLegacyReplaySource, selectDiscoveredSkillForSync } from './sync-source.ts';
import type { SyncOptions } from './sync.ts';

export interface SyncWorkflowFailure {
  skill: string;
  error: string;
}

export interface SyncWorkflowResult {
  successCount: number;
  failCount: number;
  skippedCount: number;
  failures: SyncWorkflowFailure[];
  skipped: string[];
}

function recordLabel(record: InstalledSkillView): string {
  return record.packageName || record.displayName;
}

function matchesFilter(record: InstalledSkillView, requested?: string[]): boolean {
  if (!requested || requested.length === 0) {
    return true;
  }

  const aliases = new Set(record.aliases.map((alias) => alias.toLowerCase()));
  if (record.packageName) {
    aliases.add(record.packageName.toLowerCase());
  }

  return requested.some((value) => aliases.has(value.toLowerCase()));
}

function isSyncableSourceType(sourceType: string): boolean {
  return sourceType === 'local' || sourceType === 'github' || sourceType === 'git';
}

function resolveScope(options: SyncOptions): InstalledViewScope {
  return options.global ? 'global' : 'project';
}

function buildGlobalLegacyMetadata(record: InstalledSkillView): SkillLockLegacyMetadata | undefined {
  const legacy =
    record.lockEntry && 'legacy' in record.lockEntry
      ? (record.lockEntry.legacy as SkillLockLegacyMetadata | undefined)
      : undefined;
  if (!legacy) {
    return undefined;
  }

  return {
    ...(legacy.source ? { source: legacy.source } : {}),
    ...(legacy.sourceUrl ? { sourceUrl: legacy.sourceUrl } : {}),
    ...(legacy.ref ? { ref: legacy.ref } : {}),
    ...(legacy.skillPath ? { skillPath: legacy.skillPath } : {}),
  };
}

function buildLocalLegacyMetadata(record: InstalledSkillView): LocalSkillLegacyMetadata | undefined {
  const legacy = record.lockEntry && 'legacy' in record.lockEntry ? record.lockEntry.legacy : undefined;
  if (!legacy) {
    return undefined;
  }

  return {
    ...(legacy.source ? { source: legacy.source } : {}),
    ...(legacy.ref ? { ref: legacy.ref } : {}),
    ...(legacy.skillPath ? { skillPath: legacy.skillPath } : {}),
  };
}

export async function runSyncWorkflow(options: SyncOptions): Promise<SyncWorkflowResult> {
  const scope = resolveScope(options);
  const records = await listInstalledView({
    scope,
    cwd: process.cwd(),
  });
  const selectedRecords = records.filter((record) => matchesFilter(record, options.skills));

  const result: SyncWorkflowResult = {
    successCount: 0,
    failCount: 0,
    skippedCount: 0,
    failures: [],
    skipped: [],
  };

  for (const record of selectedRecords) {
    if (!isSyncableSourceType(record.sourceType) || record.orphaned) {
      result.skippedCount++;
      result.skipped.push(recordLabel(record));
      continue;
    }

    const parsedSource = parseLegacyReplaySource(record.lockEntry || {});
    const recordedSkillPath = getRecordedSkillPath(record.lockEntry || {});

    if (!parsedSource || !recordedSkillPath) {
      result.failCount++;
      result.failures.push({
        skill: recordLabel(record),
        error: 'legacy.source and legacy.skillPath are required for sync',
      });
      continue;
    }

    let cleanupRoot: string | null = null;

    try {
      const sourceRoot =
        parsedSource.type === 'local'
          ? parsedSource.localPath || parsedSource.url
          : await cloneRepo(parsedSource.url, parsedSource.ref);
      cleanupRoot = parsedSource.type === 'local' ? null : sourceRoot;

      const searchRoot = parsedSource.subpath ? join(sourceRoot, parsedSource.subpath) : sourceRoot;
      const discoveredSkills = await discoverSkills(sourceRoot, parsedSource.subpath, {
        fullDepth: false,
      });
      const discoveredSkill = selectDiscoveredSkillForSync(
        discoveredSkills,
        searchRoot,
        recordedSkillPath
      );

      const targetAgents = Array.from(
        new Set(record.agentLinks.filter((link) => link.kind !== 'missing').map((link) => link.agent))
      );

      if (targetAgents.length === 0) {
        throw new Error('Installed skill has no materialized agent targets to refresh.');
      }

      for (const agent of targetAgents) {
        const installResult = await installSkillForAgent(discoveredSkill, agent, {
          global: scope === 'global',
          cwd: process.cwd(),
        });
        if (!installResult.success) {
          throw new Error(installResult.error || 'Failed to reinstall skill.');
        }
      }

      const nextHash = await computeSkillFolderHash(discoveredSkill.path);

      if (scope === 'global') {
        await addSkillToLock(record.lockKey || record.installName, {
          packageName: record.packageName,
          installName: record.installName,
          version: record.version,
          registry: record.registry,
          sourceType: record.sourceType,
          skillFolderHash: nextHash,
          ...(record.pluginName ? { pluginName: record.pluginName } : {}),
          ...(buildGlobalLegacyMetadata(record) ? { legacy: buildGlobalLegacyMetadata(record) } : {}),
        });
      } else {
        await addSkillToLocalLock(
          record.lockKey || record.installName,
          {
            packageName: record.packageName,
            installName: record.installName,
            version: record.version,
            registry: record.registry,
            sourceType: record.sourceType,
            computedHash: nextHash,
            ...(buildLocalLegacyMetadata(record) ? { legacy: buildLocalLegacyMetadata(record) } : {}),
          },
          process.cwd()
        );
      }

      result.successCount++;
    } catch (error) {
      result.failCount++;
      result.failures.push({
        skill: recordLabel(record),
        error: error instanceof Error ? error.message : String(error),
      });
    } finally {
      if (cleanupRoot) {
        await cleanupTempDir(cleanupRoot).catch(() => {});
      }
    }
  }

  return result;
}
