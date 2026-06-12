# Skills CLI Self-Contained Package Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the published `skills` CLI package installable and runnable from a private npm registry without npm fetching runtime dependencies such as `yaml` from a second registry.

**Architecture:** Keep the existing `obuild` bundle pipeline, but change the bundle contract so only Node built-ins remain external at runtime. Add a packaging regression test that packs the local tarball, installs it under a deliberately broken registry, and proves the CLI still launches.

**Tech Stack:** Node.js, npm, Vitest, obuild, rolldown

---

## File Map

- Modify: `apps/skills-cli/build.config.mjs`
  - Override `obuild`'s default external list so bundled CLI output keeps Node built-ins external but inlines package runtime dependencies.
- Modify: `apps/skills-cli/package.json`
  - Move `yaml` out of runtime dependency semantics and wire the new packaging regression test into the existing verification scripts.
- Create: `apps/skills-cli/tests/self-contained-package.test.ts`
  - End-to-end packaging regression that builds, packs, installs the tarball into a temp project with a broken registry, and launches the CLI.
- Re-check: `apps/skills-cli/tests/dist.test.ts`
  - Keep the existing built-artifact smoke test unchanged unless the new packaging test makes it redundant.

### Task 1: Lock the private-registry failure with a packaging regression test

**Files:**
- Create: `apps/skills-cli/tests/self-contained-package.test.ts`

- [ ] **Step 1: Write the failing packaging regression test**

Create `apps/skills-cli/tests/self-contained-package.test.ts` with this content:

```ts
import { execFileSync } from 'node:child_process';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const rootDir = resolve(import.meta.dirname, '..');

function run(command: string, args: string[], cwd: string) {
  return execFileSync(command, args, {
    cwd,
    encoding: 'utf-8',
    stdio: 'pipe',
  });
}

describe('packed skills cli package', () => {
  it('installs from a local tarball without fetching third-party runtime dependencies', { timeout: 120000 }, () => {
    const packDir = mkdtempSync(join(tmpdir(), 'skills-pack-'));
    const installDir = mkdtempSync(join(tmpdir(), 'skills-install-'));

    try {
      run('npm', ['run', 'build'], rootDir);

      const packed = run('npm', ['pack', rootDir, '--json'], packDir);
      const [{ filename }] = JSON.parse(packed) as Array<{ filename: string }>;
      const tarballPath = join(packDir, filename);

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
        ],
        installDir
      );

      const help = run('node', ['node_modules/skills/dist/cli.mjs', '--help'], installDir);

      expect(help).toContain('skills');
    } finally {
      rmSync(packDir, { recursive: true, force: true });
      rmSync(installDir, { recursive: true, force: true });
    }
  });
});
```

- [ ] **Step 2: Run the new test to verify the current package fails**

Run:

```bash
npx vitest run tests/self-contained-package.test.ts
```

Expected: FAIL during `npm install <tarball>` with a registry fetch error for `yaml`, `ECONNREFUSED`, or equivalent proof that npm still tries to resolve a non-bundled runtime dependency.

- [ ] **Step 3: Commit the red test harness on its own branch state if your workflow requires checkpointing**

```bash
git status --short
```

Expected: only `tests/self-contained-package.test.ts` is new at this point. Do not commit yet if your team prefers green-only commits.

### Task 2: Bundle runtime dependencies into the published CLI artifact

**Files:**
- Modify: `apps/skills-cli/build.config.mjs`
- Modify: `apps/skills-cli/package.json`

- [ ] **Step 1: Update the build config so only Node built-ins remain external**

Replace `apps/skills-cli/build.config.mjs` with:

```js
import { builtinModules } from 'node:module';
import { defineBuildConfig } from 'obuild/config';

const builtinRuntimeExternals = [
  ...builtinModules,
  ...builtinModules.map((moduleName) => `node:${moduleName}`),
];

export default defineBuildConfig({
  entries: [{ type: 'bundle', input: './src/cli.ts' }],
  hooks: {
    rolldownConfig(config) {
      config.external = builtinRuntimeExternals;
    },
  },
});
```

- [ ] **Step 2: Move `yaml` out of npm runtime dependency resolution**

Edit `apps/skills-cli/package.json` so `yaml` is no longer under `"dependencies"` and instead lives under `"devDependencies"`.

The relevant JSON shape should become:

```json
{
  "devDependencies": {
    "@clack/prompts": "^0.11.0",
    "@types/bun": "latest",
    "@types/node": "^22.10.0",
    "@vercel/detect-agent": "^1.2.1",
    "husky": "^9.1.7",
    "license-checker": "^25.0.1",
    "lint-staged": "^16.2.7",
    "obuild": "^0.4.22",
    "picocolors": "^1.1.1",
    "prettier": "^3.8.1",
    "simple-git": "^3.27.0",
    "typescript": "^5.9.3",
    "vitest": "^4.0.17",
    "xdg-basedir": "^5.1.0",
    "yaml": "^2.8.3"
  }
}
```

There should be no top-level `"dependencies"` block left unless another true runtime external dependency is still intentionally shipped.

- [ ] **Step 3: Re-run the packaging regression test to verify the fix**

Run:

```bash
npx vitest run tests/self-contained-package.test.ts
```

Expected: PASS. The tarball installs successfully even with `--registry http://127.0.0.1:9`, and `node node_modules/skills/dist/cli.mjs --help` prints help output containing `skills`.

- [ ] **Step 4: Inspect the packed manifest to confirm the contract**

Run:

```bash
npm pack --json
```

Expected: PASS and a tarball filename. Then inspect it:

```bash
tar -xOf skills-*.tgz package/package.json
```

Expected: the packed manifest does not declare `yaml` under `"dependencies"`.

- [ ] **Step 5: Commit the self-contained bundle fix**

```bash
git add build.config.mjs package.json tests/self-contained-package.test.ts
git commit -m "Bundle skills CLI runtime dependencies"
```

### Task 3: Make the new regression part of normal verification

**Files:**
- Modify: `apps/skills-cli/package.json`
- Re-check: `apps/skills-cli/tests/dist.test.ts`

- [ ] **Step 1: Add the new packaging regression to the dist verification script**

Update the `test:dist` script in `apps/skills-cli/package.json` to:

```json
{
  "scripts": {
    "test:dist": "vitest run tests/dist.test.ts tests/self-contained-package.test.ts"
  }
}
```

- [ ] **Step 2: Run the dist-focused verification suite**

Run:

```bash
npm run test:dist
```

Expected: PASS with both:

- `tests/dist.test.ts`
- `tests/self-contained-package.test.ts`

- [ ] **Step 3: Run the broader CLI verification suite**

Run:

```bash
npm run test
```

Expected: PASS with no regressions in the existing CLI unit and integration tests.

- [ ] **Step 4: Commit the verification wiring**

```bash
git add package.json
git commit -m "Add self-contained package regression coverage"
```

## Spec Coverage Check

- Packaging regression harness: Task 1
- Keep `obuild` as first implementation path: Task 2
- Make runtime dependencies self-contained after `npm pack`: Task 2
- Add ongoing regression coverage: Task 3
- Preserve CLI behavior and built-artifact smoke coverage: Task 3

## Placeholder Scan

- No `TODO` or `TBD` markers remain.
- Every file path referenced above exists or is explicitly created.
- Every verification step includes an exact command and expected result.

## Type and Contract Consistency

- The plan keeps the npm bin entrypoint at `bin/cli.mjs`.
- The built runtime artifact remains `dist/cli.mjs`.
- The regression checks the packed tarball boundary, not just source-tree behavior.
