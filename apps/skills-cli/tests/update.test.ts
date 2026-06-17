import { describe, it, expect, vi, beforeEach } from 'vitest';
import { runUpdate, updateProjectSkills, updateGlobalSkills } from '../src/update.ts';
import * as workflow from '../src/registry-install-workflow.ts';
import * as catalog from '../src/catalog.ts';
import * as installedView from '../src/installed-view.ts';
import { runCliOutput } from '../src/test-utils.ts';

vi.mock('../src/registry-install-workflow.ts', () => ({
  runRegistryInstallWorkflow: vi.fn(),
}));

vi.mock('../src/catalog.ts');
vi.mock('../src/installed-view.ts', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../src/installed-view.ts')>();
  return {
    ...actual,
    listInstalledView: vi.fn(() => Promise.resolve([])),
  };
});
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
    vi.mocked(installedView.listInstalledView).mockResolvedValue([]);
    Object.defineProperty(process.stdin, 'isTTY', {
      value: true,
      configurable: true,
    });
  });

  describe('updateProjectSkills', () => {
    it('shows dedicated update help output', () => {
      const output = runCliOutput(['update', '--help']);
      expect(output).toContain('Usage: skills update [skills...] [options]');
      expect(output).toContain('Update installed registry skills to latest versions');
      expect(output).toContain('--global');
      expect(output).toContain('skills update @company.skill/pdf-skill');
      expect(output).not.toContain('npx skills');
    });

    it('uses concise install guidance when no project skills are installed', async () => {
      const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});

      await updateProjectSkills({ yes: true });

      expect(logSpy).toHaveBeenCalledWith(
        expect.stringContaining('Install project skills with')
      );
      expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('skills add <package_name>'));
      expect(logSpy).not.toHaveBeenCalledWith(expect.stringContaining('npx skills'));
      logSpy.mockRestore();
    });

    it('re-resolves registry-installed project skills by package name', async () => {
      vi.mocked(installedView.listInstalledView).mockResolvedValue([
        {
          packageName: '@platform.skill/pdf-skill',
          displayName: 'PDF Skill',
          installName: 'pdf-skill',
          aliases: ['@platform.skill/pdf-skill', 'pdf-skill'],
          scope: 'project',
          canonicalPath: '/tmp/project/.agents/skills/pdf-skill',
          agentLinks: [],
          sourceType: 'registry',
          version: '1.0.0',
          registry: 'https://catalog.example.com',
          orphaned: false,
          orphanReasons: [],
        },
      ]);

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
      vi.mocked(installedView.listInstalledView).mockResolvedValue([
        {
          displayName: 'Legacy Skill',
          installName: 'legacy-skill',
          aliases: ['legacy-skill'],
          scope: 'project',
          canonicalPath: '/tmp/project/.agents/skills/legacy-skill',
          agentLinks: [],
          sourceType: 'github',
          orphaned: false,
          orphanReasons: [],
        },
      ]);

      await updateProjectSkills({ yes: true });

      expect(catalog.resolveInstallTarget).not.toHaveBeenCalled();
      expect(workflow.runRegistryInstallWorkflow).not.toHaveBeenCalled();
    });

    it('skips reinstall when the installed version already matches the resolved target', async () => {
      const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});

      vi.mocked(installedView.listInstalledView).mockResolvedValue([
        {
          packageName: '@platform.skill/pdf-skill',
          displayName: 'PDF Skill',
          installName: 'pdf-skill',
          aliases: ['@platform.skill/pdf-skill', 'pdf-skill'],
          scope: 'project',
          canonicalPath: '/tmp/project/.agents/skills/pdf-skill',
          agentLinks: [],
          sourceType: 'registry',
          version: '1.2.0',
          registry: 'https://catalog.example.com',
          orphaned: false,
          orphanReasons: [],
        },
      ]);

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
      expect(result).toEqual({
        successCount: 0,
        failCount: 0,
        matchedCount: 1,
        updatableCount: 1,
        skippedCount: 0,
      });
      expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('Already up to date'));

      logSpy.mockRestore();
    });

    it('prints a no-op summary when every targeted skill is already current', async () => {
      const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});

      vi.mocked(installedView.listInstalledView).mockResolvedValue([
        {
          packageName: '@platform.skill/pdf-skill',
          displayName: 'PDF Skill',
          installName: 'pdf-skill',
          aliases: ['@platform.skill/pdf-skill', 'pdf-skill'],
          scope: 'project',
          canonicalPath: '/tmp/project/.agents/skills/pdf-skill',
          agentLinks: [],
          sourceType: 'registry',
          version: '1.2.0',
          registry: 'https://catalog.example.com',
          orphaned: false,
          orphanReasons: [],
        },
      ]);

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

    it('does not print a clean no-op summary when every matched project skill is skipped as orphaned', async () => {
      const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});

      vi.mocked(installedView.listInstalledView).mockResolvedValue([
        {
          packageName: '@platform.skill/pdf-skill',
          displayName: 'PDF Skill',
          installName: 'pdf-skill',
          aliases: ['@platform.skill/pdf-skill', 'pdf-skill'],
          scope: 'project',
          canonicalPath: '/tmp/project/.agents/skills/pdf-skill',
          agentLinks: [],
          sourceType: 'registry',
          version: '1.2.0',
          registry: 'https://catalog.example.com',
          orphaned: true,
          orphanReasons: ['missing_lock_entry'],
        },
      ]);

      await runUpdate(['-y']);

      expect(logSpy).not.toHaveBeenCalledWith(expect.stringContaining('No updates were needed.'));
      expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('repair local installed state'));
      expect(logSpy).toHaveBeenCalledWith(
        expect.stringContaining('need local state repair before update')
      );

      logSpy.mockRestore();
    });

    it('selects a project registry record by alias through installed view filtering', async () => {
      vi.mocked(installedView.listInstalledView).mockResolvedValue([
        {
          packageName: '@platform.skill/pdf-skill',
          displayName: 'PDF Skill',
          installName: 'pdf-skill',
          aliases: ['@platform.skill/pdf-skill', 'pdf-skill'],
          scope: 'project',
          canonicalPath: '/tmp/project/.agents/skills/pdf-skill',
          agentLinks: [],
          sourceType: 'registry',
          version: '1.0.0',
          registry: 'https://catalog.example.com',
          orphaned: false,
          orphanReasons: [],
        },
        {
          packageName: '@platform.skill/other-skill',
          displayName: 'Other Skill',
          installName: 'other-skill',
          aliases: ['@platform.skill/other-skill', 'other-skill'],
          scope: 'project',
          canonicalPath: '/tmp/project/.agents/skills/other-skill',
          agentLinks: [],
          sourceType: 'registry',
          version: '2.0.0',
          registry: 'https://catalog.example.com',
          orphaned: false,
          orphanReasons: [],
        },
      ]);

      vi.mocked(catalog.resolveInstallTarget).mockResolvedValue({
        packageName: '@platform.skill/pdf-skill',
        version: '1.1.0',
        npmRegistry: 'https://npm.example.com',
        tarballUrl: 'https://npm.example.com/@platform.skill/pdf-skill/-/pdf-skill-1.1.0.tgz',
        integrity: 'sha512-test',
        skillManifestDigest: 'sha256-test',
        versionStatus: 'published',
      });

      await updateProjectSkills({ yes: true, skills: ['pdf-skill'] });

      expect(catalog.resolveInstallTarget).toHaveBeenCalledTimes(1);
      expect(catalog.resolveInstallTarget).toHaveBeenCalledWith('@platform.skill/pdf-skill', {
        registry: 'https://catalog.example.com',
        version: undefined,
      });
      expect(workflow.runRegistryInstallWorkflow).toHaveBeenCalledWith(
        expect.objectContaining({
          packageName: '@platform.skill/pdf-skill',
          version: '1.1.0',
          installGlobally: false,
        })
      );
    });
  });

  describe('updateGlobalSkills', () => {
    it('skips orphaned registry installs and tells the user to repair local state first', async () => {
      const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});

      vi.mocked(installedView.listInstalledView).mockResolvedValue([
        {
          packageName: '@platform.skill/pdf-skill',
          displayName: 'PDF Skill',
          installName: 'pdf-skill',
          aliases: ['@platform.skill/pdf-skill', 'pdf-skill'],
          scope: 'global',
          canonicalPath: '/tmp/home/.agents/skills/pdf-skill',
          agentLinks: [],
          sourceType: 'registry',
          version: '1.2.3',
          registry: 'https://catalog.example.com',
          orphaned: true,
          orphanReasons: ['missing_lock_entry'],
        },
      ]);

      await updateGlobalSkills();

      expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('repair local installed state'));
      expect(catalog.resolveInstallTarget).not.toHaveBeenCalled();

      logSpy.mockRestore();
    });

    it('re-resolves registry-installed global skills by package name', async () => {
      vi.mocked(installedView.listInstalledView).mockResolvedValue([
        {
          packageName: '@platform.skill/pdf-skill',
          displayName: 'PDF Skill',
          installName: 'pdf-skill',
          aliases: ['@platform.skill/pdf-skill', 'pdf-skill'],
          scope: 'global',
          canonicalPath: '/tmp/home/.agents/skills/pdf-skill',
          agentLinks: [],
          sourceType: 'registry',
          version: '1.0.0',
          registry: 'https://catalog.example.com',
          orphaned: false,
          orphanReasons: [],
        },
      ]);

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
      vi.mocked(installedView.listInstalledView).mockResolvedValue([
        {
          displayName: 'Legacy Skill',
          installName: 'legacy-skill',
          aliases: ['legacy-skill'],
          scope: 'global',
          canonicalPath: '/tmp/home/.agents/skills/legacy-skill',
          agentLinks: [],
          sourceType: 'github',
          orphaned: false,
          orphanReasons: [],
        },
      ]);

      await updateGlobalSkills({ yes: true });

      expect(catalog.resolveInstallTarget).not.toHaveBeenCalled();
      expect(workflow.runRegistryInstallWorkflow).not.toHaveBeenCalled();
    });

    it('updates healthy installed-view registry skills without inferring package identity from package.json', async () => {
      vi.mocked(installedView.listInstalledView).mockResolvedValue([
        {
          packageName: '@company.skill/my-skill',
          displayName: 'My Skill',
          installName: 'my-skill',
          aliases: ['@company.skill/my-skill', 'my-skill'],
          scope: 'global',
          canonicalPath: '/Users/test/.agents/skills/my-skill',
          agentLinks: [],
          sourceType: 'registry',
          version: '0.1.0',
          orphaned: false,
          orphanReasons: [],
        },
      ]);
      vi.mocked(catalog.resolveInstallTarget).mockResolvedValue({
        packageName: '@company.skill/my-skill',
        version: '0.1.1',
        npmRegistry: 'https://npm.example.com',
        tarballUrl: 'https://npm.example.com/@company.skill/my-skill/-/my-skill-0.1.1.tgz',
        integrity: 'sha512-test',
        skillManifestDigest: 'sha256-test',
        versionStatus: 'published',
      });

      await updateGlobalSkills({ yes: true });

      expect(catalog.resolveInstallTarget).toHaveBeenCalledWith('@company.skill/my-skill', {
        registry: undefined,
        version: undefined,
      });
      expect(workflow.runRegistryInstallWorkflow).toHaveBeenCalledWith(
        expect.objectContaining({
          packageName: '@company.skill/my-skill',
          version: '0.1.1',
          installGlobally: true,
        })
      );
    });
  });
});
