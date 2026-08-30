import { describe, it, expect } from 'vitest';
import { mkdtempSync, readFileSync, rmSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';
import { runCliOutput, hasLogo } from './test-utils.ts';

describe('skills CLI', () => {
  describe('--help', () => {
    it('should display help message', () => {
      const output = runCliOutput(['--help']);
      expect(output).toContain('Usage: skills <command> [options]');
      expect(output).toContain('Manage Skills:');
      expect(output).toContain('init [name]');
      expect(output).toContain('add <source>');
      expect(output).toContain('update');
      expect(output).toContain('Add Options:');
      expect(output).toContain('-g, --global');
      expect(output).toContain('-a, --agent');
      expect(output).toContain('-y, --yes');
      expect(output).toContain('--registry <url>');
      expect(output).toContain('SKILLS_REGISTRY_URL');
      expect(output).toContain('--version <version>');
      expect(output).toContain('--verbose');
      expect(output).not.toContain('vercel-labs/agent-skills');
    });

    it('should show same output for -h alias', () => {
      const helpOutput = runCliOutput(['--help']);
      const hOutput = runCliOutput(['-h']);
      expect(hOutput).toBe(helpOutput);
    });
  });

  describe('--version', () => {
    it('should display version number', () => {
      const output = runCliOutput(['--version']);
      expect(output.trim()).toMatch(/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/);
    });

    it('should match package.json version', () => {
      const output = runCliOutput(['--version']);
      const pkg = JSON.parse(
        readFileSync(join(import.meta.dirname, '..', 'package.json'), 'utf-8')
      );
      expect(output.trim()).toBe(pkg.version);
    });
  });

  describe('no arguments', () => {
    it('should display concise help instead of the marketing banner', () => {
      const output = runCliOutput([]);
      expect(output).toContain('Usage: skills <command> [options]');
      expect(output).toContain('Manage Skills:');
      expect(output).not.toContain('The open agent skills ecosystem');
      expect(output).not.toContain('npx skills add');
      expect(output).not.toContain('skills.sh');
      expect(output).not.toContain('vercel-labs/agent-skills');
    });
  });

  describe('unknown command', () => {
    it('should show error for unknown command', () => {
      const output = runCliOutput(['unknown-command']);
      expect(output).toMatchInlineSnapshot(`
        "Unknown command: unknown-command
        Run skills --help for usage.
        "
      `);
    });
  });

  describe('logo display', () => {
    it('should not display logo for init command', () => {
      const testDir = mkdtempSync(join(tmpdir(), 'skills-cli-logo-test-'));
      try {
        const output = runCliOutput(['init', 'smoke-skill'], testDir);
        expect(hasLogo(output)).toBe(false);
      } finally {
        rmSync(testDir, { recursive: true, force: true });
      }
    });

    it('should not display logo for list command', () => {
      const output = runCliOutput(['list']);
      expect(hasLogo(output)).toBe(false);
    });

    it('should not display logo for update command', () => {
      // Note: update command makes GitHub API calls, so we just verify initial output
      const output = runCliOutput(['update']);
      expect(hasLogo(output)).toBe(false);
    }, 60000);
  });

  describe('command surface', () => {
    it('should reject the removed check compatibility alias', () => {
      const output = runCliOutput(['check']);
      expect(output).toContain('Unknown command: check');
      expect(output).toContain('Run skills --help for usage.');
    });

    it('should reject the removed upgrade compatibility alias', () => {
      const output = runCliOutput(['upgrade']);
      expect(output).toContain('Unknown command: upgrade');
      expect(output).toContain('Run skills --help for usage.');
    });

    it('should reject removed find compatibility aliases', () => {
      expect(runCliOutput(['search'])).toContain('Unknown command: search');
      expect(runCliOutput(['f'])).toContain('Unknown command: f');
      expect(runCliOutput(['s'])).toContain('Unknown command: s');
    });
  });
});
