# Publish and import workflows

## Authority

The CLI collects source input and calls the server. The server reviews and normalizes the package, creates the artifact, writes the release record, and records audit data. The CLI never publishes directly to the artifact repository.

## Commands

- `skills publish [path]` publishes a local skill directory;
- `skills import-local [path]` imports a local source tree, with optional package name and version injection;
- `skills import-git <source>` clones a git source, discovers one skill, and imports it;
- `skills import-npm <package>` downloads an npm package in the CLI and submits its extracted directory through the same local import path.

## Common server workflow

1. inspect `SKILL.md` and package metadata;
2. validate package identity, exact version, and skill structure;
3. normalize imported source when allowed;
4. produce a manifest digest and component list;
5. package the source archive;
6. publish bundle bytes to the configured npm-compatible repository;
7. create the server-owned release record;
8. return publication state, warnings, import report, and errors;
9. record the mutation audit event.

The packaging API accepts uploaded archives and platform-authored structured files. The CLI's local, git, and npm import paths currently submit `upload_archive` after source collection or extraction. The server does not accept a git, local, npm, or generic archive source type as a normalized client request.

## Import behavior

`import-local` and `import-git` require a valid `SKILL.md`, display name, description, scoped package name, and exact semver. The CLI flags `--package-name` and `--version` can supply missing metadata. `import-git` requires a single discovered skill or an explicit `--skill` selector.

The `--strict` option disables imported workspace normalization. Without it, the server may generate required package or capability files and reports those changes in `importReport` and warnings.

`import-npm` preserves the requested package name and selected exact version. By default it reports mirror posture; `--hosted` changes the client-side requested posture, but release acceptance remains a server decision.

## Idempotency and failures

Packaging requests may be replayed safely with an idempotency key. A successful response reports whether the release was newly created or an idempotent replay. Review summaries used by `publish` must match the server's packaging result.

A failed review or publish produces explicit errors and does not count as a published release. Governance and release history remain server-owned.
