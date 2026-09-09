# External-source implementation status

This document separates implemented behavior from deferred design work. The command and API references are authoritative for current contracts.

## Implemented

### External catalog search

The server has one configurable external search adapter for `skills.sh`. `find` forwards `--include-external`, `--source`, `--registry`, `--scope`, and `--origin` to the catalog API. External results are opt-in and remain non-installable until a hosted release exists.

### Provenance response fields

Catalog domain responses support source, origin, import-mode, hosting, mirroring, and installability fields. The `skills.sh` adapter populates the relevant external-search fields. Existing hosted release data may not have additional provenance values.

### npm import command

`skills import-npm <package>` resolves package metadata from an npm-compatible registry, downloads and extracts the selected tarball in the CLI, then sends an archive to the existing server local-import workflow. It preserves the selected package name and version in the submission.

`--hosted` currently changes only the CLI's reported `import_mode` value. It is not included in the server request and does not alter the created release record or server ownership model.

### Local and git source installation

`skills add` supports local paths, GitHub shorthand, and git URLs. Successful source installs are recorded in local state and trigger a best-effort `import-local` server workflow by default. A project `skills.config.json` can disable it or supply missing package metadata.

## Deferred work

The repository contains response-model seams for richer provenance, but it does not currently implement:

- searching arbitrary npm registries, git repositories, or local paths through the server;
- direct CLI npm search as an external `find` source;
- normalized npm/local source objects in packaging API requests;
- server persistence of npm, git, or local source provenance from import requests;
- mirror-versus-hosted release semantics or promotion;
- external source installation through `skills add` by package name;
- JSON output for import commands.

## Design constraints for future work

- Preserve package name plus exact version as the release identity.
- Keep the server authoritative for hosted release, visibility, and governance decisions.
- Use server API extensions rather than a parallel registry model.
- Do not claim an external result is installable unless the server can resolve an install target.
- Make provenance writes, collision behavior, recovery, and governance semantics explicit before adding source promotion.
