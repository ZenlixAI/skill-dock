import { spawnSync } from 'child_process';
import { mkdtempSync, readFileSync, rmSync } from 'fs';
import { tmpdir } from 'os';
import { join, resolve } from 'path';

const IMPORT_REPORT_PATH = '.skill-packaging-import-report.json';

export function createSourceArchiveBase64(directory: string): string {
  const sourceDirectory = resolve(directory);
  const tempDir = mkdtempSync(join(tmpdir(), 'skills-source-archive-'));
  const archivePath = join(tempDir, 'source.tgz');

  try {
    const result = spawnSync(
      'tar',
      [
        '-czf',
        archivePath,
        '--exclude',
        '.git',
        '--exclude',
        '.dist',
        '--exclude',
        IMPORT_REPORT_PATH,
        '-C',
        sourceDirectory,
        '.',
      ],
      {
        stdio: ['inherit', 'pipe', 'pipe'],
        encoding: 'utf-8',
        env: {
          ...process.env,
          COPYFILE_DISABLE: '1',
        },
        shell: process.platform === 'win32',
      }
    );

    if (result.status !== 0) {
      throw new Error(result.stderr || result.stdout || 'tar archive creation failed');
    }

    return readFileSync(archivePath).toString('base64');
  } finally {
    rmSync(tempDir, { recursive: true, force: true });
  }
}
