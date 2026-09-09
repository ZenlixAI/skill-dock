# Install resolution and refresh

## Commands

- `skills find` searches catalog data and optional external search results;
- `skills add` installs a registry package or a discovered local/git source;
- `skills update` refreshes registry-installed entries;
- `skills sync` refreshes local/git source-installed entries.

## Registry installs

For a package identity, the CLI asks the server for an install target. The target supplies the exact version, integrity, manifest digest, tarball URL, and registry access mode. The CLI materializes that bundle for the selected scope and agents, then records registry identity in the local or global lock file.

`proxy` targets are fetched through the server. `direct` targets are fetched from the upstream npm-compatible repository using matching npm credentials when needed. The CLI does not select a fallback release.

## Source installs

A source install discovers skills from:

- a local filesystem path;
- GitHub shorthand such as `owner/repo`;
- a git URL.

The CLI materializes the discovered skill for selected agents and records its source, ref, skill path, and content hash in the project `skills-lock.json` or global `.skill-lock.json`.

After a successful local or git `add`, automatic server import runs by default. It is best-effort and uses `skills.config.json`; it does not change the source lock classification or registry refresh behavior.

## Refresh semantics

- `update` applies only to entries with `sourceType: registry` and follows server release resolution;
- `sync` applies only to entries with source types `local`, `github`, or `git`;
- `sync` replays the recorded source and skill path, reinstalls for recorded agents, and updates the content hash;
- `sync` does not query the catalog or publish changed source content;
- registry and source refresh paths must remain separate.

Entries with missing source metadata or orphaned installations are skipped or reported as failures rather than guessed.

## Local state

`list` and `remove` operate on installed files and lock files. They do not change catalog records. Project state is stored in `skills-lock.json`; global state is stored under the user's XDG state directory or `~/.agents/.skill-lock.json`.
