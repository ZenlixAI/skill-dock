import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { mkdirSync, rmSync, writeFileSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';

vi.mock('./import-local-workflow.ts', () => ({
  runImportLocalWorkflow: vi.fn(),
}));

import { syncAddedSkillToBackend } from './add-sync.ts';
import { runImportLocalWorkflow } from './import-local-workflow.ts';

describe('add sync helper', () => {
  let root: string;
  let configHome: string;
  let originalConfigHome: string | undefined;

  beforeEach(() => {
    root = join(
      tmpdir(),
      `skills-add-sync-test-${Date.now()}-${Math.random().toString(36).slice(2)}`
    );
    configHome = join(root, 'config');
    originalConfigHome = process.env.XDG_CONFIG_HOME;
    process.env.XDG_CONFIG_HOME = configHome;
    mkdirSync(root, { recursive: true });
    vi.mocked(runImportLocalWorkflow).mockReset();
    vi.mocked(runImportLocalWorkflow).mockResolvedValue({
      ok: true,
      packageName: '@acme.skill/demo-skill',
      version: '0.1.0',
      manifestDigest: 'sha256:test',
      components: [],
      publishedToRegistry: true,
      created: true,
      idempotentReplay: false,
      versionStatus: 'published',
      installPolicy: 'resolvable',
    });
  });

  afterEach(() => {
    if (originalConfigHome === undefined) {
      delete process.env.XDG_CONFIG_HOME;
    } else {
      process.env.XDG_CONFIG_HOME = originalConfigHome;
    }
    rmSync(root, { recursive: true, force: true });
    vi.restoreAllMocks();
  });

  it('uses project defaults when package metadata is absent', async () => {
    writeFileSync(
      join(root, 'SKILL.md'),
      ['---', 'name: Demo Skill', 'description: Demo skill', '---', '', '# Demo Skill'].join('\n')
    );
    writeFileSync(
      join(root, 'skills.config.json'),
      JSON.stringify(
        {
          syncOnAdd: true,
          packageNamePrefix: '@acme.skill/',
          defaultVersion: '0.1.0',
          registry: 'https://catalog.example.com',
        },
        null,
        2
      )
    );

    const result = await syncAddedSkillToBackend(root, {
      cwd: root,
      nameHint: 'ignored',
    });

    expect(result).toBeNull();
    expect(runImportLocalWorkflow).toHaveBeenCalledWith({
      path: root,
      registry: 'https://catalog.example.com',
      packageName: '@acme.skill/demo-skill',
      version: '0.1.0',
    });
  });

  it('uses global defaults when project configuration is absent', async () => {
    writeFileSync(
      join(root, 'SKILL.md'),
      ['---', 'name: Demo Skill', 'description: Demo skill', '---', '', '# Demo Skill'].join('\n')
    );
    mkdirSync(join(configHome, 'skills'), { recursive: true });
    writeFileSync(
      join(configHome, 'skills', 'config.json'),
      JSON.stringify(
        {
          syncOnAdd: true,
          packageNamePrefix: '@global.skill/',
          defaultVersion: '0.3.0',
          registry: 'https://global-catalog.example.com',
        },
        null,
        2
      )
    );

    const result = await syncAddedSkillToBackend(root, { cwd: root });

    expect(result).toBeNull();
    expect(runImportLocalWorkflow).toHaveBeenCalledWith({
      path: root,
      registry: 'https://global-catalog.example.com',
      packageName: '@global.skill/demo-skill',
      version: '0.3.0',
    });
  });

  it('merges the nearest project configuration over global defaults', async () => {
    const nestedDirectory = join(root, 'packages', 'demo');
    mkdirSync(nestedDirectory, { recursive: true });
    writeFileSync(
      join(nestedDirectory, 'SKILL.md'),
      ['---', 'name: Demo Skill', 'description: Demo skill', '---', '', '# Demo Skill'].join('\n')
    );
    mkdirSync(join(configHome, 'skills'), { recursive: true });
    writeFileSync(
      join(configHome, 'skills', 'config.json'),
      JSON.stringify(
        {
          syncOnAdd: true,
          packageNamePrefix: '@global.skill/',
          defaultVersion: '0.1.0',
          registry: 'https://global-catalog.example.com',
        },
        null,
        2
      )
    );
    writeFileSync(
      join(root, 'skills.config.json'),
      JSON.stringify(
        {
          defaultVersion: '0.2.0',
          registry: 'https://project-catalog.example.com',
        },
        null,
        2
      )
    );

    const result = await syncAddedSkillToBackend(nestedDirectory, { cwd: nestedDirectory });

    expect(result).toBeNull();
    expect(runImportLocalWorkflow).toHaveBeenCalledWith({
      path: nestedDirectory,
      registry: 'https://project-catalog.example.com',
      packageName: '@global.skill/demo-skill',
      version: '0.2.0',
    });
  });

  it('allows project configuration to disable globally enabled sync', async () => {
    writeFileSync(
      join(root, 'SKILL.md'),
      ['---', 'name: Demo Skill', 'description: Demo skill', '---', '', '# Demo Skill'].join('\n')
    );
    mkdirSync(join(configHome, 'skills'), { recursive: true });
    writeFileSync(
      join(configHome, 'skills', 'config.json'),
      JSON.stringify({ syncOnAdd: true }, null, 2)
    );
    writeFileSync(join(root, 'skills.config.json'), JSON.stringify({ syncOnAdd: false }, null, 2));

    const result = await syncAddedSkillToBackend(root, { cwd: root });

    expect(result).toBeNull();
    expect(runImportLocalWorkflow).not.toHaveBeenCalled();
  });

  it('syncs by default when no configuration is present', async () => {
    writeFileSync(
      join(root, 'SKILL.md'),
      ['---', 'name: From Package', 'description: Demo skill', '---', '', '# Demo Skill'].join('\n')
    );
    writeFileSync(
      join(root, 'package.json'),
      JSON.stringify(
        {
          name: '@acme.skill/from-package',
          version: '1.2.3',
        },
        null,
        2
      )
    );

    const result = await syncAddedSkillToBackend(root, { cwd: root });

    expect(result).toBeNull();
    expect(runImportLocalWorkflow).toHaveBeenCalledWith({
      path: root,
      registry: undefined,
      packageName: '@acme.skill/from-package',
      version: '1.2.3',
    });
  });

  it('prefers package metadata over configuration defaults', async () => {
    writeFileSync(
      join(root, 'SKILL.md'),
      ['---', 'name: From Package', 'description: Demo skill', '---', '', '# Demo Skill'].join('\n')
    );
    writeFileSync(
      join(root, 'package.json'),
      JSON.stringify(
        {
          name: '@acme.skill/from-package',
          version: '1.2.3',
        },
        null,
        2
      )
    );
    writeFileSync(
      join(root, 'skills.config.json'),
      JSON.stringify(
        {
          syncOnAdd: true,
          packageNamePrefix: '@ignored.skill/',
          defaultVersion: '0.1.0',
        },
        null,
        2
      )
    );

    const result = await syncAddedSkillToBackend(root, { cwd: root });

    expect(result).toBeNull();
    expect(runImportLocalWorkflow).toHaveBeenCalledWith({
      path: root,
      registry: undefined,
      packageName: '@acme.skill/from-package',
      version: '1.2.3',
    });
  });
});
