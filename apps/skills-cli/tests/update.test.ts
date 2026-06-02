import { describe, it, expect, vi, beforeEach } from 'vitest';
import { runUpdate, updateProjectSkills, updateGlobalSkills } from '../src/update.ts';
import * as workflow from '../src/registry-install-workflow.ts';
import * as catalog from '../src/catalog.ts';
import * as localLock from '../src/local-lock.ts';
import * as skillLock from '../src/skill-lock.ts';
import { runCliOutput } from '../src/test-utils.ts';

vi.mock('../src/registry-install-workflow.ts', () => ({
  runRegistryInstallWorkflow: vi.fn(),
}));

vi.mock('../src/catalog.ts');
vi.mock('../src/local-lock.ts');
vi.mock('../src/skill-lock.ts');
vi.mock('../src/agents.ts', () => ({
  agents: {
    'claude-code': {
      displayName: 'Claude Code',
      skillsDir: '.claude/skills',
      globalSkillsDir: '/tmp/.claude/skills',
    },
  },
  detectInstalledAgents: vi.fn(() => Promise.resolve(['claude-code'])),
  getUniversalAgents: vi.fn(() => ['claude-code']),
}));

vi.mock('fs', async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...actual,
    existsSync: vi.fn().mockReturnValue(true),
  };
});

describe('update command', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    Object.defineProperty(process.stdin, 'isTTY', {
      value: true,
      configurable: true,
    });
  });

  describe('updateProjectSkills', () => {
    it('shows dedicated update help output', () => {
      const output = runCliOutput(['update', '--help']);
      expect(output).toContain('Usage: skills update [skills...] [options]');
      expect(output).toContain('Update installed skills to latest versions');
      expect(output).toContain('--global');
      expect(output).toContain('skills update @platform.skill/pdf-skill');
    });

    it('re-resolves registry-installed project skills by package name', async () => {
      vi.mocked(localLock.readLocalLock).mockResolvedValue({
        version: 1,
        skills: {
          '@platform.skill/pdf-skill': {
            sourceType: 'registry',
            packageName: '@platform.skill/pdf-skill',
            version: '1.0.0',
            registry: 'https://catalog.example.com',
            computedHash: 'abc',
          },
        },
      });

      vi.mocked(catalog.resolveInstallTarget).mockResolvedValue({
        packageName: '@platform.skill/pdf-skill',
        version: '1.2.0',
        npmRegistry: 'https://npm.example.com',
        tarballUrl: 'https://npm.example.com/@platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz',
        integrity: 'sha512-test',
        skillManifestDigest: 'sha256-test',
        versionStatus: 'published',
      });

      await updateProjectSkills({ yes: true });

      expect(catalog.resolveInstallTarget).toHaveBeenCalledWith('@platform.skill/pdf-skill', {
        registry: 'https://catalog.example.com',
        version: undefined,
      });
      expect(workflow.runRegistryInstallWorkflow).toHaveBeenCalledWith(
        expect.objectContaining({
          packageName: '@platform.skill/pdf-skill',
          version: '1.2.0',
          registry: 'https://catalog.example.com',
          installGlobally: false,
        })
      );
    });

    it('ignores non-registry project entries', async () => {
      vi.mocked(localLock.readLocalLock).mockResolvedValue({
        version: 1,
        skills: {
          'legacy-skill': {
            sourceType: 'github',
            computedHash: 'abc',
            legacy: {
              source: 'owner/repo',
              skillPath: 'skills/legacy-skill/SKILL.md',
            },
          },
        },
      });

      await updateProjectSkills({ yes: true });

      expect(catalog.resolveInstallTarget).not.toHaveBeenCalled();
      expect(workflow.runRegistryInstallWorkflow).not.toHaveBeenCalled();
    });

    it('skips reinstall when the installed version already matches the resolved target', async () => {
      const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});

      vi.mocked(localLock.readLocalLock).mockResolvedValue({
        version: 1,
        skills: {
          '@platform.skill/pdf-skill': {
            sourceType: 'registry',
            packageName: '@platform.skill/pdf-skill',
            version: '1.2.0',
            registry: 'https://catalog.example.com',
            computedHash: 'abc',
          },
        },
      });

      vi.mocked(catalog.resolveInstallTarget).mockResolvedValue({
        packageName: '@platform.skill/pdf-skill',
        version: '1.2.0',
        npmRegistry: 'https://npm.example.com',
        tarballUrl: 'https://npm.example.com/@platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz',
        integrity: 'sha512-test',
        skillManifestDigest: 'sha256-test',
        versionStatus: 'published',
      });

      const result = await updateProjectSkills({ yes: true });

      expect(workflow.runRegistryInstallWorkflow).not.toHaveBeenCalled();
      expect(result).toEqual({ successCount: 0, failCount: 0, foundCount: 1 });
      expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('Already up to date'));

      logSpy.mockRestore();
    });

    it('prints a no-op summary when every targeted skill is already current', async () => {
      const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});

      vi.mocked(localLock.readLocalLock).mockResolvedValue({
        version: 1,
        skills: {
          '@platform.skill/pdf-skill': {
            sourceType: 'registry',
            packageName: '@platform.skill/pdf-skill',
            version: '1.2.0',
            registry: 'https://catalog.example.com',
            computedHash: 'abc',
          },
        },
      });

      vi.mocked(catalog.resolveInstallTarget).mockResolvedValue({
        packageName: '@platform.skill/pdf-skill',
        version: '1.2.0',
        npmRegistry: 'https://npm.example.com',
        tarballUrl: 'https://npm.example.com/@platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz',
        integrity: 'sha512-test',
        skillManifestDigest: 'sha256-test',
        versionStatus: 'published',
      });

      await runUpdate(['-y']);

      expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('No updates were needed.'));
      logSpy.mockRestore();
    });
  });

  describe('updateGlobalSkills', () => {
    it('re-resolves registry-installed global skills by package name', async () => {
      vi.mocked(skillLock.readSkillLock).mockResolvedValue({
        version: 3,
        skills: {
          '@platform.skill/pdf-skill': {
            sourceType: 'registry',
            skillFolderHash: '',
            installedAt: '',
            updatedAt: '',
            packageName: '@platform.skill/pdf-skill',
            version: '1.0.0',
            registry: 'https://catalog.example.com',
          },
        },
      });

      vi.mocked(catalog.resolveInstallTarget).mockResolvedValue({
        packageName: '@platform.skill/pdf-skill',
        version: '1.2.0',
        npmRegistry: 'https://npm.example.com',
        tarballUrl: 'https://npm.example.com/@platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz',
        integrity: 'sha512-test',
        skillManifestDigest: 'sha256-test',
        versionStatus: 'published',
      });

      await updateGlobalSkills({ yes: true });

      expect(catalog.resolveInstallTarget).toHaveBeenCalledWith('@platform.skill/pdf-skill', {
        registry: 'https://catalog.example.com',
        version: undefined,
      });
      expect(workflow.runRegistryInstallWorkflow).toHaveBeenCalledWith(
        expect.objectContaining({
          packageName: '@platform.skill/pdf-skill',
          version: '1.2.0',
          registry: 'https://catalog.example.com',
          installGlobally: true,
        })
      );
    });

    it('ignores non-registry global entries', async () => {
      vi.mocked(skillLock.readSkillLock).mockResolvedValue({
        version: 3,
        skills: {
          'legacy-skill': {
            sourceType: 'github',
            skillFolderHash: 'abc',
            installedAt: '',
            updatedAt: '',
            legacy: {
              source: 'owner/repo',
              sourceUrl: 'https://github.com/owner/repo.git',
              skillPath: 'skills/legacy-skill/SKILL.md',
            },
          },
        },
      });

      await updateGlobalSkills({ yes: true });

      expect(catalog.resolveInstallTarget).not.toHaveBeenCalled();
      expect(workflow.runRegistryInstallWorkflow).not.toHaveBeenCalled();
    });
  });
});
