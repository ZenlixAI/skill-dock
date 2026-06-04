import { beforeEach, describe, expect, it, vi } from 'vitest';
import { runRegistryInstallWorkflow } from './registry-install-workflow.ts';
import * as catalog from './catalog.ts';
import * as registryBundle from './registry-bundle.ts';
import * as installer from './installer.ts';
import * as skillLock from './skill-lock.ts';
import * as localLock from './local-lock.ts';

vi.mock('./catalog.ts', () => ({
  resolveInstallTarget: vi.fn(),
  getRegistryBaseUrl: vi.fn((value?: string) => value || 'https://skills.sh'),
}));

vi.mock('./registry-bundle.ts', () => ({
  downloadRegistrySkillBundle: vi.fn(),
}));

vi.mock('./installer.ts', () => ({
  installSkillForAgent: vi.fn(),
  deriveInstallName: vi.fn((value: string) => value.split('/').pop() || value),
  sanitizeName: vi.fn((value: string) =>
    value
      .replace(/^@/, '')
      .replace(/[^a-zA-Z0-9._]+/g, '-')
      .toLowerCase()
  ),
}));

vi.mock('./skill-lock.ts', () => ({
  addSkillToLock: vi.fn(),
  readSkillLock: vi.fn(() => Promise.resolve({ version: 3, skills: {} })),
}));

vi.mock('./local-lock.ts', () => ({
  addSkillToLocalLock: vi.fn(),
  computeSkillFolderHash: vi.fn(() => Promise.resolve('sha256-folder-hash')),
  readLocalLock: vi.fn(() => Promise.resolve({ version: 1, skills: {} })),
}));

describe('runRegistryInstallWorkflow', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('stores a real global skillFolderHash for registry installs', async () => {
    vi.mocked(catalog.resolveInstallTarget).mockResolvedValue({
      packageName: '@platform.skill/pdf-skill',
      version: '1.2.0',
      npmRegistry: 'https://npm.example.com',
      tarballUrl: 'https://npm.example.com/@platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz',
      integrity: 'sha512-test',
      skillManifestDigest: 'sha256-manifest',
      versionStatus: 'published',
    });

    const cleanup = vi.fn(() => Promise.resolve());
    vi.mocked(registryBundle.downloadRegistrySkillBundle).mockResolvedValue({
      extractedPath: '/tmp/extracted-skill',
      cleanup,
      skill: {
        name: 'pdf-skill',
        packageName: '@platform.skill/pdf-skill',
        installName: 'pdf-skill',
        path: '/tmp/extracted-skill',
        description: 'PDF skill',
      },
    });

    vi.mocked(installer.installSkillForAgent).mockResolvedValue({
      success: true,
      path: '/tmp/.agents/skills/pdf-skill',
      canonicalPath: '/tmp/.agents/skills/pdf-skill',
      mode: 'copy',
    });

    const result = await runRegistryInstallWorkflow({
      packageName: '@platform.skill/pdf-skill',
      version: '1.2.0',
      registry: 'https://catalog.example.com',
      installGlobally: true,
      installMode: 'copy',
      targetAgents: ['claude-code'],
      cwd: '/tmp/project',
    });

    expect(localLock.computeSkillFolderHash).toHaveBeenCalledWith('/tmp/extracted-skill');
    expect(skillLock.addSkillToLock).toHaveBeenCalledWith('@platform.skill/pdf-skill', {
      sourceType: 'registry',
      skillFolderHash: 'sha256-folder-hash',
      packageName: '@platform.skill/pdf-skill',
      version: '1.2.0',
      registry: 'https://catalog.example.com',
      integrity: 'sha512-test',
      skillManifestDigest: 'sha256-manifest',
      installName: 'pdf-skill',
    });
    expect(result.target.version).toBe('1.2.0');
    expect(result.successful).toHaveLength(1);
    expect(cleanup).toHaveBeenCalled();
  });

  it('stores minimal project registry lock metadata without legacy source fields', async () => {
    vi.mocked(catalog.resolveInstallTarget).mockResolvedValue({
      packageName: '@platform.skill/pdf-skill',
      version: '1.2.0',
      npmRegistry: 'https://npm.example.com',
      tarballUrl: 'https://npm.example.com/@platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz',
      integrity: 'sha512-test',
      skillManifestDigest: 'sha256-manifest',
      versionStatus: 'published',
    });

    const cleanup = vi.fn(() => Promise.resolve());
    vi.mocked(registryBundle.downloadRegistrySkillBundle).mockResolvedValue({
      extractedPath: '/tmp/extracted-skill',
      cleanup,
      skill: {
        name: 'pdf-skill',
        packageName: '@platform.skill/pdf-skill',
        installName: 'pdf-skill',
        path: '/tmp/extracted-skill',
        description: 'PDF skill',
      },
    });

    vi.mocked(installer.installSkillForAgent).mockResolvedValue({
      success: true,
      path: '/tmp/project/.agents/skills/pdf-skill',
      canonicalPath: '/tmp/project/.agents/skills/pdf-skill',
      mode: 'copy',
    });

    await runRegistryInstallWorkflow({
      packageName: '@platform.skill/pdf-skill',
      version: '1.2.0',
      registry: 'https://catalog.example.com',
      installGlobally: false,
      installMode: 'copy',
      targetAgents: ['claude-code'],
      cwd: '/tmp/project',
    });

    expect(localLock.addSkillToLocalLock).toHaveBeenCalledWith(
      '@platform.skill/pdf-skill',
      {
        sourceType: 'registry',
        computedHash: 'sha256-folder-hash',
        packageName: '@platform.skill/pdf-skill',
        version: '1.2.0',
        registry: 'https://catalog.example.com',
        integrity: 'sha512-test',
        skillManifestDigest: 'sha256-manifest',
        installName: 'pdf-skill',
      },
      '/tmp/project'
    );
    expect(cleanup).toHaveBeenCalled();
  });

  it('falls back to a scoped installName when another project package already owns the basename', async () => {
    vi.mocked(localLock.readLocalLock).mockResolvedValue({
      version: 1,
      skills: {
        '@other.skill/zoom-out': {
          sourceType: 'registry',
          packageName: '@other.skill/zoom-out',
          installName: 'zoom-out',
          computedHash: 'existing-hash',
        },
      },
    });
    vi.mocked(catalog.resolveInstallTarget).mockResolvedValue({
      packageName: '@zhenlikeji.skill/zoom-out',
      version: '1.2.0',
      npmRegistry: 'https://npm.example.com',
      tarballUrl: 'https://npm.example.com/@zhenlikeji.skill/zoom-out/-/zoom-out-1.2.0.tgz',
      integrity: 'sha512-test',
      skillManifestDigest: 'sha256-manifest',
      versionStatus: 'published',
    });

    const cleanup = vi.fn(() => Promise.resolve());
    vi.mocked(registryBundle.downloadRegistrySkillBundle).mockResolvedValue({
      extractedPath: '/tmp/extracted-skill',
      cleanup,
      skill: {
        name: 'zoom-out',
        packageName: '@zhenlikeji.skill/zoom-out',
        installName: 'zoom-out',
        path: '/tmp/extracted-skill',
        description: 'Zoom out skill',
      },
    });

    vi.mocked(installer.installSkillForAgent).mockImplementation(async (skill) => ({
      success: true,
      path: `/tmp/project/.agents/skills/${skill.installName}`,
      canonicalPath: `/tmp/project/.agents/skills/${skill.installName}`,
      mode: 'copy',
    }));

    await runRegistryInstallWorkflow({
      packageName: '@zhenlikeji.skill/zoom-out',
      version: '1.2.0',
      registry: 'https://catalog.example.com',
      installGlobally: false,
      installMode: 'copy',
      targetAgents: ['claude-code'],
      cwd: '/tmp/project',
    });

    expect(installer.installSkillForAgent).toHaveBeenCalledWith(
      expect.objectContaining({
        installName: 'zhenlikeji.skill-zoom-out',
      }),
      'claude-code',
      expect.anything()
    );
    expect(localLock.addSkillToLocalLock).toHaveBeenCalledWith(
      '@zhenlikeji.skill/zoom-out',
      expect.objectContaining({
        installName: 'zhenlikeji.skill-zoom-out',
      }),
      '/tmp/project'
    );
    expect(cleanup).toHaveBeenCalled();
  });

  it('preserves an existing fallback installName for the same global package', async () => {
    vi.mocked(skillLock.readSkillLock).mockResolvedValue({
      version: 3,
      skills: {
        '@zhenlikeji.skill/zoom-out': {
          sourceType: 'registry',
          packageName: '@zhenlikeji.skill/zoom-out',
          installName: 'zhenlikeji.skill-zoom-out',
          skillFolderHash: 'existing-hash',
          installedAt: '2026-06-03T00:00:00.000Z',
          updatedAt: '2026-06-03T00:00:00.000Z',
        },
      },
    });
    vi.mocked(catalog.resolveInstallTarget).mockResolvedValue({
      packageName: '@zhenlikeji.skill/zoom-out',
      version: '1.2.1',
      npmRegistry: 'https://npm.example.com',
      tarballUrl: 'https://npm.example.com/@zhenlikeji.skill/zoom-out/-/zoom-out-1.2.1.tgz',
      integrity: 'sha512-test',
      skillManifestDigest: 'sha256-manifest',
      versionStatus: 'published',
    });

    const cleanup = vi.fn(() => Promise.resolve());
    vi.mocked(registryBundle.downloadRegistrySkillBundle).mockResolvedValue({
      extractedPath: '/tmp/extracted-skill',
      cleanup,
      skill: {
        name: 'zoom-out',
        packageName: '@zhenlikeji.skill/zoom-out',
        installName: 'zoom-out',
        path: '/tmp/extracted-skill',
        description: 'Zoom out skill',
      },
    });

    vi.mocked(installer.installSkillForAgent).mockImplementation(async (skill) => ({
      success: true,
      path: `/tmp/.agents/skills/${skill.installName}`,
      canonicalPath: `/tmp/.agents/skills/${skill.installName}`,
      mode: 'copy',
    }));

    await runRegistryInstallWorkflow({
      packageName: '@zhenlikeji.skill/zoom-out',
      version: '1.2.1',
      registry: 'https://catalog.example.com',
      installGlobally: true,
      installMode: 'copy',
      targetAgents: ['claude-code'],
      cwd: '/tmp/project',
    });

    expect(installer.installSkillForAgent).toHaveBeenCalledWith(
      expect.objectContaining({
        installName: 'zhenlikeji.skill-zoom-out',
      }),
      'claude-code',
      expect.anything()
    );
    expect(skillLock.addSkillToLock).toHaveBeenCalledWith(
      '@zhenlikeji.skill/zoom-out',
      expect.objectContaining({
        installName: 'zhenlikeji.skill-zoom-out',
      })
    );
    expect(cleanup).toHaveBeenCalled();
  });
});
