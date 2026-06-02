import { describe, it, expect } from 'vitest';
import { readFileSync } from 'fs';
import { join } from 'path';
import { runCliOutput, stripLogo, hasLogo } from './test-utils.ts';

describe('skills CLI', () => {
  describe('--help', () => {
    it('should display help message', () => {
      const output = runCliOutput(['--help']);
      expect(output).toContain('Usage: skills <command> [options]');
      expect(output).toContain('Manage Skills:');
      expect(output).toContain('init [name]');
      expect(output).toContain('add <package_name>');
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
      expect(output.trim()).toMatch(/^\d+\.\d+\.\d+$/);
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
    it('should display banner', () => {
      const output = stripLogo(runCliOutput([]));
      expect(output).toContain('The open agent skills ecosystem');
      expect(output).toContain('npx skills add');
      expect(output).toContain('npx skills update');
      expect(output).toContain('npx skills init');
      expect(output).toContain('skills.sh');
      expect(output).toContain('@platform.skill/pdf-skill');
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
