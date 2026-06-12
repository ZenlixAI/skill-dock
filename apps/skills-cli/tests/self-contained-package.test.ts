import { execFileSync } from 'node:child_process';
import {
  mkdirSync,
  mkdtempSync,
  readdirSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { builtinModules } from 'node:module';
import { describe, expect, it } from 'vitest';
import { gunzipSync } from 'node:zlib';

const rootDir = resolve(import.meta.dirname, '..');
const builtinModuleSpecifiers = new Set([
  ...builtinModules,
  ...builtinModules.map((moduleName) => `node:${moduleName}`),
]);

function run(command: string, args: string[], cwd: string, env: NodeJS.ProcessEnv = {}) {
  return execFileSync(command, args, {
    cwd,
    encoding: 'utf-8',
    env: { ...process.env, ...env },
    stdio: 'pipe',
  });
}

function parsePackOutput(output: string) {
  const jsonStart = output.search(/\[\s*\{/);
  if (jsonStart === -1) {
    throw new Error(`npm pack did not return JSON output:\n${output}`);
  }

  return JSON.parse(output.slice(jsonStart)) as Array<{ filename: string }>;
}

function commandOutput(error: unknown) {
  if (!(error instanceof Error)) {
    return String(error);
  }

  const { stdout = '', stderr = '' } = error as Error & {
    stdout?: string | Buffer;
    stderr?: string | Buffer;
  };

  return `${stdout.toString()}${stderr.toString()}`;
}

function readTarballEntry(tarballPath: string, entryName: string) {
  const archive = gunzipSync(readFileSync(tarballPath));
  let offset = 0;

  while (offset + 512 <= archive.length) {
    const header = archive.subarray(offset, offset + 512);

    if (header.every((byte) => byte === 0)) {
      break;
    }

    const name = header.subarray(0, 100).toString('utf-8').replace(/\0.*$/, '');
    const sizeText = header.subarray(124, 136).toString('utf-8').replace(/\0.*$/, '').trim();
    const size = sizeText ? Number.parseInt(sizeText, 8) : 0;
    const dataStart = offset + 512;
    const dataEnd = dataStart + size;

    if (name === entryName) {
      return archive.subarray(dataStart, dataEnd).toString('utf-8');
    }

    offset = dataStart + Math.ceil(size / 512) * 512;
  }

  throw new Error(`Entry not found in tarball: ${entryName}`);
}

function packageNameFromSpecifier(specifier: string) {
  if (specifier.startsWith('@')) {
    return specifier.split('/').slice(0, 2).join('/');
  }

  return specifier.split('/')[0];
}

function collectRuntimePackageImports(dir: string) {
  const packageImports = new Set<string>();
  const importPatterns = [
    /(?:import|export)\s+(?:[^'"]*?\s+from\s+)?['"]([^'"]+)['"]/g,
    /import\(\s*['"]([^'"]+)['"]\s*\)/g,
  ];

  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const fullPath = join(dir, entry.name);

    if (entry.isDirectory()) {
      for (const specifier of collectRuntimePackageImports(fullPath)) {
        packageImports.add(specifier);
      }

      continue;
    }

    if (!entry.isFile() || !fullPath.endsWith('.ts')) {
      continue;
    }

    const source = readFileSync(fullPath, 'utf-8');

    for (const pattern of importPatterns) {
      for (const match of source.matchAll(pattern)) {
        const [, specifier] = match;

        if (
          !specifier ||
          specifier.startsWith('.') ||
          specifier.startsWith('/') ||
          builtinModuleSpecifiers.has(specifier)
        ) {
          continue;
        }

        packageImports.add(packageNameFromSpecifier(specifier));
      }
    }
  }

  return [...packageImports].sort();
}

function createNpmEnv(baseDir: string): NodeJS.ProcessEnv {
  const homeDir = join(baseDir, 'home');
  const cacheDir = join(baseDir, 'cache');
  const configDir = join(baseDir, 'config');
  const userConfigPath = join(configDir, 'npmrc');

  mkdirSync(homeDir, { recursive: true });
  mkdirSync(cacheDir, { recursive: true });
  mkdirSync(configDir, { recursive: true });
  writeFileSync(userConfigPath, '', 'utf-8');

  return {
    HOME: homeDir,
    USERPROFILE: homeDir,
    XDG_CACHE_HOME: cacheDir,
    XDG_CONFIG_HOME: configDir,
    npm_config_cache: cacheDir,
    npm_config_userconfig: userConfigPath,
    npm_config_update_notifier: 'false',
  };
}

describe('skills cli build config', () => {
  it('keeps only Node builtins external at runtime', async () => {
    const { default: buildConfig } = await import('../build.config.mjs');
    const config = {
      external: (id: string) =>
        id === 'picocolors' || id.startsWith('picocolors/') || id === 'left-pad',
    };

    await buildConfig.hooks?.rolldownConfig?.(config, { pkgDir: rootDir, pkg: { name: 'skills' } });

    expect(config.external).toEqual(
      expect.arrayContaining(['node:path', 'path', 'node:fs', 'fs', 'node:module', 'module'])
    );
    expect(Array.isArray(config.external)).toBe(true);
    expect((config.external as Array<string | RegExp>).includes('picocolors')).toBe(false);
    expect((config.external as Array<string | RegExp>).includes('left-pad')).toBe(false);
  });
});

describe('packed skills cli package', () => {
  it(
    'installs from a local tarball without fetching third-party runtime dependencies',
    { timeout: 120000 },
    () => {
      const workspaceDir = mkdtempSync(join(tmpdir(), 'skills-self-contained-'));
      const packDir = join(workspaceDir, 'pack');
      const installDir = join(workspaceDir, 'install');
      const npmEnv = createNpmEnv(join(workspaceDir, 'npm'));
      const packageManifest = JSON.parse(readFileSync(join(rootDir, 'package.json'), 'utf-8')) as {
        dependencies?: Record<string, string>;
        devDependencies?: Record<string, string>;
        optionalDependencies?: Record<string, string>;
        peerDependencies?: Record<string, string>;
      };
      const declaredPackages = new Set([
        ...Object.keys(packageManifest.dependencies ?? {}),
        ...Object.keys(packageManifest.devDependencies ?? {}),
        ...Object.keys(packageManifest.optionalDependencies ?? {}),
        ...Object.keys(packageManifest.peerDependencies ?? {}),
      ]);
      const undeclaredRuntimeImports = collectRuntimePackageImports(join(rootDir, 'src')).filter(
        (packageName) => !declaredPackages.has(packageName)
      );

      try {
        expect(undeclaredRuntimeImports).toEqual([]);
        mkdirSync(packDir, { recursive: true });
        mkdirSync(installDir, { recursive: true });

        run('npm', ['run', 'build'], rootDir, npmEnv);

        const packed = run(
          'npm',
          ['pack', '--json', '--ignore-scripts', '--pack-destination', packDir],
          rootDir,
          npmEnv
        );
        const [{ filename }] = parsePackOutput(packed);
        const tarballPath = join(packDir, filename);
        const packedManifest = JSON.parse(readTarballEntry(tarballPath, 'package/package.json')) as {
          dependencies?: Record<string, string>;
        };

        expect(Object.hasOwn(packedManifest, 'dependencies')).toBe(false);
        expect(Object.keys(packedManifest.dependencies ?? {})).toEqual([]);

        writeFileSync(
          join(installDir, 'package.json'),
          JSON.stringify(
            {
              name: 'skills-self-contained-smoke',
              private: true,
            },
            null,
            2
          )
        );

        let installError: unknown;
        try {
          run(
            'npm',
            [
              'install',
              tarballPath,
              '--registry',
              'http://127.0.0.1:9',
              '--no-audit',
              '--no-fund',
              '--ignore-scripts',
              '--fetch-retries',
              '0',
              '--fetch-timeout',
              '1000',
            ],
            installDir,
            npmEnv
          );
        } catch (error) {
          installError = error;
        }

        if (installError) {
          const output = commandOutput(installError);

          throw new Error(`npm install failed for a self-contained tarball:\n${output}`);
        }

        const help = run('node', ['node_modules/skills/dist/cli.mjs', '--help'], installDir);

        expect(help).toContain('skills');
      } finally {
        rmSync(workspaceDir, { recursive: true, force: true });
      }
    }
  );
});
