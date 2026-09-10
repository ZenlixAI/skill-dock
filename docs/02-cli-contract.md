# CLI contract and command model

## Role

`skills` is the operational entry point for developers and automation. It talks to the server for registry-backed and packaging workflows, while performing source discovery and local materialization on the client.

## Command model

- `find` searches the catalog and optional `skills.sh` results;
- `add` installs a registry package or a skill discovered from a local or git source;
- `list` shows local installed state;
- `remove` uninstalls local skills;
- `update` refreshes registry-installed skills;
- `sync` refreshes source-installed skills;
- `init` creates a local scaffold;
- `inspect` validates a local skill directory;
- `publish` packages and publishes a local skill through the server;
- `import-local` packages and publishes a local source tree;
- `import-git` clones, discovers, and publishes one git-sourced skill;
- `import-npm` downloads an npm package and publishes its extracted source through the server.

## Core rules

- `SKILLS_REGISTRY_URL` is the catalog server base URL;
- `--registry` is a per-invocation catalog override and is not persisted;
- registry package installs resolve through the server;
- source installs may use a local path, GitHub shorthand, or git URL;
- `update` applies only to registry-installed entries;
- `sync` applies only to local or git source-installed entries;
- `list` and `remove` query and mutate local installed state only;
- `skills --help` is the command-level source for the supported flags and examples.

## Automatic source import

For a successful local or git `add`, the CLI runs `syncAddedSkillToBackend` after writing the local installation lock entry. It calls the existing `import-local` server workflow.

This is enabled by default. The CLI merges the optional global `${XDG_CONFIG_HOME:-~/.config}/skills/config.json` with the nearest project `skills.config.json` found from the current working directory upward. Project values override global values; set `syncOnAdd` to `false` to opt out explicitly.

Either file can set:

```json
{
  "syncOnAdd": true,
  "packageNamePrefix": "@acme.skill/",
  "defaultVersion": "0.1.0",
  "registry": "http://127.0.0.1:8080"
}
```

`packageNamePrefix` and `defaultVersion` are used only when the discovered source has no package metadata. `defaultVersion` must be an exact semver. `registry` selects the catalog endpoint for this automatic import. Invalid or unreadable config is ignored; set `syncOnAdd` to `false` to opt out explicitly.

The global configuration path is `${XDG_CONFIG_HOME:-~/.config}/skills/config.json`; project configuration is `skills.config.json`.

Automatic import is best-effort: a server failure produces a warning but does not roll back the local installation. It does not apply to registry package installs, and it does not replace `skills sync`.

## Registry install behavior

The server returns a complete install target including package name, exact version, integrity, manifest digest, and `registryAccessMode`:

- `proxy`: download through the server proxy;
- `direct`: download from the upstream registry and read matching auth from npm configuration when needed.

The CLI does not choose or reinterpret that mode.

## Source refresh behavior

`sync` replays the source recorded in `skills-lock.json` or `.skill-lock.json`, rediscovers the recorded skill path, reinstalls it for the recorded agents, and updates its local content hash. It does not query the catalog and does not automatically publish changed source content.

## Boundaries

The CLI does not own catalog truth, release visibility, governance, artifact storage, or server-side packaging decisions. It must not publish directly to the artifact repository or invent a second registry model.
