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
}));

vi.mock('./skill-lock.ts', () => ({
  addSkillToLock: vi.fn(),
}));

vi.mock('./local-lock.ts', () => ({
  addSkillToLocalLock: vi.fn(),
  computeSkillFolderHash: vi.fn(() => Promise.resolve('sha256-folder-hash')),
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
    expect(skillLock.addSkillToLock).toHaveBeenCalledWith(
      '@platform.skill/pdf-skill',
      {
        sourceType: 'registry',
        skillFolderHash: 'sha256-folder-hash',
        packageName: '@platform.skill/pdf-skill',
        version: '1.2.0',
        registry: 'https://catalog.example.com',
        integrity: 'sha512-test',
        skillManifestDigest: 'sha256-manifest',
      }
    );
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
      },
      '/tmp/project'
    );
    expect(cleanup).toHaveBeenCalled();
  });
});
