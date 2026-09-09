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

  beforeEach(() => {
    root = join(tmpdir(), `skills-add-sync-test-${Date.now()}`);
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
    rmSync(root, { recursive: true, force: true });
    vi.restoreAllMocks();
  });

  it('uses configured package name prefix and default version when package metadata is absent', async () => {
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

  it('syncs by default when no skills.config.json is present', async () => {
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

  it('prefers package metadata over config defaults when available', async () => {
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
