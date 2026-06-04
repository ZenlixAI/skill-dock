import { describe, expect, it } from 'vitest';
import { mkdirSync, writeFileSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { mkdtempSync, rmSync } from 'fs';
import { getCanonicalPath, installSkillForAgent } from './installer.ts';

describe('registry install naming', () => {
  it('derives canonical paths from package basename instead of scoped package identity', () => {
    expect(getCanonicalPath('@zhenlikeji.skill/zoom-out', { cwd: '/tmp/project' })).toBe(
      '/tmp/project/.agents/skills/zoom-out'
    );
  });

  it('installs registry skills using installName for the local directory', async () => {
    const cwd = mkdtempSync(join(tmpdir(), 'skills-installer-test-'));
    const sourceDir = join(cwd, 'source');
    mkdirSync(sourceDir, { recursive: true });
    writeFileSync(
      join(sourceDir, 'SKILL.md'),
      `---
name: Zoom Out
description: Broader context
---

# Zoom Out
`
    );

    try {
      const result = await installSkillForAgent(
        {
          name: 'Zoom Out',
          description: 'Broader context',
          path: sourceDir,
          packageName: '@zhenlikeji.skill/zoom-out',
          installName: 'zoom-out',
        },
        'codex',
        {
          cwd,
          mode: 'copy',
        }
      );

      expect(result.success).toBe(true);
      expect(result.path).toBe(join(cwd, '.agents/skills/zoom-out'));
      expect(result.path.includes('zhenlikeji.skill-zoom-out')).toBe(false);
    } finally {
      rmSync(cwd, { recursive: true, force: true });
    }
  });
});
