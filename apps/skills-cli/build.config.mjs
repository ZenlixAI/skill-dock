import { builtinModules } from 'node:module';
import { defineBuildConfig } from 'obuild/config';

const builtinRuntimeExternals = [
  ...builtinModules,
  ...builtinModules.map((moduleName) => `node:${moduleName}`),
];
const builtinRuntimeExternalSet = new Set(builtinRuntimeExternals);

function manifestPackageNames(pkg) {
  return new Set([...Object.keys(pkg.dependencies || {}), ...Object.keys(pkg.peerDependencies || {})]);
}

function matchesConfiguredExternal(external, id, parentId, isResolved) {
  if (!external) {
    return false;
  }

  if (typeof external === 'function') {
    return Boolean(external(id, parentId, isResolved));
  }

  const entries = Array.isArray(external) ? external : [external];

  return entries.some((entry) => {
    if (typeof entry === 'string') {
      return entry === id;
    }

    return entry.test(id);
  });
}

function isManifestPackageExternal(id, pkg) {
  const [scopeOrName, maybeName] = id.split('/');
  const packageName = scopeOrName.startsWith('@') ? `${scopeOrName}/${maybeName || ''}` : scopeOrName;

  return manifestPackageNames(pkg).has(packageName);
}

function mergeExternalConfig(existingExternal, pkg) {
  return (id, parentId, isResolved) => {
    if (builtinRuntimeExternalSet.has(id)) {
      return true;
    }

    if (isManifestPackageExternal(id, pkg)) {
      return false;
    }

    return matchesConfiguredExternal(existingExternal, id, parentId, isResolved);
  };
}

// https://github.com/unjs/obuild
export default defineBuildConfig({
  entries: [{ type: 'bundle', input: './src/cli.ts' }],
  hooks: {
    rolldownConfig(config, ctx) {
      config.external = mergeExternalConfig(config.external, ctx.pkg);
    },
  },
});
