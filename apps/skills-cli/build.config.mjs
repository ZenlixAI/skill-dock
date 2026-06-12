import { defineBuildConfig } from 'obuild/config';
import { builtinModules } from 'node:module';

const builtinRuntimeExternals = [
  ...builtinModules,
  ...builtinModules.map((moduleName) => `node:${moduleName}`),
];

// https://github.com/unjs/obuild
export default defineBuildConfig({
  entries: [{ type: 'bundle', input: './src/cli.ts' }],
  hooks: {
    rolldownConfig(config) {
      config.external = builtinRuntimeExternals;
    },
  },
});
