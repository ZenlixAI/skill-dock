# Skills CLI Self-Contained Package Design

## Summary

The private-registry install flow currently fails when users run:

```bash
npx --registry <private-npm> skills add <package> --version <version> --registry <catalog-base-url>
```

because npm can fetch the `skills` package itself from the private npm repository, but still tries to fetch runtime dependencies such as `yaml` separately. In deployments where the private npm repository does not proxy public npm packages, `npx` fails before the CLI starts.

The design goal is to make the published `skills` CLI package self-contained at runtime so that fetching the `skills` tarball is sufficient to execute the CLI.

## Problem

Today the `skills` CLI is built with `obuild` in bundle mode, but the packaged artifact still behaves as if at least one runtime dependency remains external. In practice:

- `npx` downloads `skills`
- npm resolves the `skills` dependency tree
- npm attempts to fetch `yaml@^2.8.3`
- private npm repositories that only host private packages return `404`
- installation fails before `skills add` can talk to Catalog

This failure mode is product-hostile for private deployments because it requires repository operators to maintain a public npm proxy just to launch the CLI.

## Goals

- Make the published `skills` package runnable after downloading only the `skills` tarball.
- Preserve the existing user-facing CLI surface and command behavior.
- Keep the existing `obuild`-based build path if it can satisfy the packaging requirement.
- Add verification that catches any future regression where runtime dependencies escape the bundle.

## Non-Goals

- Reworking private registry topology or Nexus group/proxy configuration.
- Changing Catalog API semantics.
- Replacing the CLI runtime with a native binary.
- Bundling development-only dependencies into the distributed package.

## Chosen Approach

Use the current `obuild` bundle entrypoint as the primary implementation path, but tighten the packaging contract so runtime dependencies are bundled into the published artifact rather than left for npm to resolve at install time.

The important shift is treating “self-contained after `npm pack`” as an explicit product requirement instead of an accidental property of the current bundle.

### Packaging Contract

The packaged CLI must satisfy all of the following:

- `npm pack` output contains the runnable CLI entrypoint and any files it directly imports at runtime.
- Launching the packed CLI must not require npm to download `yaml` or any other secondary runtime package.
- The package may still declare metadata and development dependencies as usual, but runtime execution after install must not depend on registry resolution of third-party packages.

## Options Considered

### Option 1: Keep `obuild`, fix runtime bundling

This is the recommended option.

Pros:

- Minimal disruption to the current build pipeline.
- Keeps current dist shape and release process mostly intact.
- Fastest path to shipping the fix.

Cons:

- Requires understanding why `yaml` still escapes bundling today.
- May depend on `obuild` configuration details that are not yet enforced by tests.

### Option 2: Switch to a more explicit bundler config

Example candidates: `esbuild`, `tsup`, or direct `rolldown`.

Pros:

- More explicit control over externalization and single-file output.
- Easier to reason about if `obuild` proves opaque.

Cons:

- Larger migration surface.
- Changes the repo’s CLI build conventions for a problem that may not require it.

### Option 3: Mirror specific public dependencies into private npm

Pros:

- No CLI code change.

Cons:

- Treats the symptom, not the product defect.
- Breaks again when a new transitive runtime dependency appears.
- Pushes packaging complexity onto every deployment.

This option is rejected.

## Architecture

The build remains:

- source entrypoint: `src/cli.ts`
- npm bin entrypoint: `bin/cli.mjs`
- built runtime artifact: `dist/cli.mjs`

The design change is not a new runtime architecture. It is a stricter boundary around what counts as a complete published artifact.

### Required Behavior

At publish time:

- build produces a runnable bundled CLI artifact
- package files include that artifact
- package installation does not require fetching runtime dependencies beyond the `skills` tarball itself

At runtime:

- `bin/cli.mjs` still delegates into `dist/cli.mjs`
- CLI behavior is unchanged
- all existing commands continue to function under Node >= 18

## Implementation Plan Shape

Implementation should follow this sequence:

1. Reproduce the current failure with a packaging-level test or harness that proves npm tries to fetch `yaml`.
2. Inspect the built artifact and packaging manifest to determine why `yaml` remains external.
3. Adjust build configuration so runtime dependencies used by the CLI are included in the bundle.
4. Re-run the packaging-level repro to prove the packed artifact no longer requires external runtime dependency fetches.
5. Keep existing command-level tests green.

## Testing Strategy

### Existing Coverage to Keep

- CLI unit and integration tests
- existing dist smoke tests

### New Required Coverage

Add a packaging regression test around the packed artifact, not just the raw source tree.

The test should verify the real product boundary:

- build the package
- create a packed tarball with `npm pack`
- install or inspect that tarball in a minimal environment
- verify that launching the CLI does not require fetching `yaml` from the registry

The preferred signal is an automated test that fails if npm attempts to resolve `yaml` externally during package execution setup.

## Risks

### Risk: Bundling changes Node runtime behavior

Mitigation:

- keep the bin entrypoint shape unchanged
- run existing CLI tests and dist smoke tests

### Risk: `obuild` cannot reliably produce a self-contained runtime bundle

Mitigation:

- treat `obuild` as the first implementation path, not an immutable constraint
- if runtime dependency externalization cannot be disabled cleanly, switch to an explicit bundler in a follow-up implementation step

### Risk: Future dependencies regress the package back to non-self-contained

Mitigation:

- enforce the packaging-level regression test in CI

## Rollout

No user migration is required.

Once shipped, private deployments should be able to use:

```bash
npx --registry <private-npm> skills add <package> --version <version> --registry <catalog-base-url>
```

without requiring the private npm repository to proxy public packages like `yaml`.

## Open Questions

None for the product direction.

The remaining uncertainty is implementation detail only: whether `obuild` can be configured to satisfy the contract directly, or whether the implementation should fall back to a more explicit bundler.
