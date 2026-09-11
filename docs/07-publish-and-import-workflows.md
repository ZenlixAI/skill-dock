# Publish and import workflows

The CLI collects source input and the server validates, packages, publishes, and records the release. The CLI never publishes directly to the artifact repository.

## Commands

- `skills publish [path]` publishes a local skill directory.
- `skills import-local [path]` publishes a local source tree.
- `skills import-git <source>` discovers and publishes one skill from a git source.
- `skills import-npm <package>` downloads and extracts an npm package, then submits the extracted source to the same server workflow.

## Server workflow

1. Validate `SKILL.md`, package identity, exact version, and source structure.
2. Normalize imported source unless strict mode is selected.
3. Produce the manifest digest and component list.
4. Package and publish the bundle to the configured npm-compatible repository.
5. Create the server-owned release record and audit event.

The packaging API accepts `upload_archive` source input and `platform_authoring` structured files. CLI imports submit `upload_archive` after collecting the source.

`--package-name` and `--version` supply missing source metadata. `--strict` disables source normalization. A failed review or publish does not create a published release. Replayed publishing requests are idempotent when they use the same idempotency key.
