import { execFileSync } from 'node:child_process';
import {
  cpSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readdirSync,
  readFileSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { builtinModules } from 'node:module';
import { describe, expect, it } from 'vitest';

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

function stagePackageSource(sourceDir: string, packagedFiles: string[]) {
  mkdirSync(sourceDir, { recursive: true });

  const filesToCopy = ['package.json', 'build.config.mjs', 'tsconfig.json', ...packagedFiles];
  const directoriesToCopy = ['src', 'scripts'];

  for (const file of filesToCopy) {
    cpSync(join(rootDir, file), join(sourceDir, file), { recursive: true });
  }

  for (const directory of directoriesToCopy) {
    cpSync(join(rootDir, directory), join(sourceDir, directory), { recursive: true });
  }

  const rootNodeModules = join(rootDir, 'node_modules');
  expect(existsSync(rootNodeModules)).toBe(true);
  symlinkSync(rootNodeModules, join(sourceDir, 'node_modules'), 'dir');
}

describe('skills cli build config', () => {
  it('keeps builtins external while forcing manifest dependencies inline for function externals', async () => {
    const { default: buildConfig } = await import('../build.config.mjs');
    const config = {
      external: (id: string) =>
        id === 'picocolors' || id.startsWith('picocolors/') || id === 'left-pad',
    };

    await buildConfig.hooks?.rolldownConfig?.(config, {
      pkgDir: rootDir,
      pkg: {
        name: 'skills',
        dependencies: { picocolors: '^1.1.1' },
        peerDependencies: { yaml: '^2.8.3' },
      },
    });

    expect(typeof config.external).toBe('function');
    expect(config.external('node:path')).toBe(true);
    expect(config.external('path')).toBe(true);
    expect(config.external('picocolors')).toBe(false);
    expect(config.external('picocolors/colors')).toBe(false);
    expect(config.external('yaml')).toBe(false);
    expect(config.external('yaml/types')).toBe(false);
    expect(config.external('left-pad')).toBe(true);
  });
});

describe('packed skills cli package', () => {
  it(
    'installs from a local tarball without fetching third-party runtime dependencies',
    { timeout: 120000 },
    () => {
      const workspaceDir = mkdtempSync(join(tmpdir(), 'skills-self-contained-'));
      const sourceDir = join(workspaceDir, 'source');
      const packDir = join(workspaceDir, 'pack');
      const installDir = join(workspaceDir, 'install');
      const npmEnv = createNpmEnv(join(workspaceDir, 'npm'));
      const packageManifest = JSON.parse(readFileSync(join(rootDir, 'package.json'), 'utf-8')) as {
        dependencies?: Record<string, string>;
        devDependencies?: Record<string, string>;
        optionalDependencies?: Record<string, string>;
        peerDependencies?: Record<string, string>;
        files?: string[];
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
      const packagedStaticFiles = (packageManifest.files ?? []).filter((entry) => entry !== 'dist');

      try {
        expect(undeclaredRuntimeImports).toEqual([]);
        mkdirSync(packDir, { recursive: true });
        mkdirSync(installDir, { recursive: true });
        stagePackageSource(sourceDir, packagedStaticFiles);

        run('npm', ['run', 'build'], sourceDir, npmEnv);

        const packed = run(
          'npm',
          ['pack', sourceDir, '--json', '--ignore-scripts', '--pack-destination', packDir],
          sourceDir,
          npmEnv
        );
        const [{ filename }] = parsePackOutput(packed);
        const tarballPath = join(packDir, filename);
        const packedManifest = JSON.parse(
          run('tar', ['-xOf', tarballPath, 'package/package.json'], packDir)
        ) as {
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
