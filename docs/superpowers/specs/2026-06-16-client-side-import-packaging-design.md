# Client-Side Import Packaging Design

Date: 2026-06-16
Status: Approved in conversation, pending written-spec review

## Context

The current external-skill import flow still allows the catalog server to act as
the networked source retriever for imported skills.

Today that behavior exists in two places:

- the packaging API accepts `source.type=git`
- the backend prepares that source by cloning or copying the source tree inside
  `internal/packaging/infra/workspace_source.go`

That design conflicts with the operating constraint the user set for this
system: the server must not depend on outbound external network access for skill
import.

The repository already has a cleaner pattern available. The existing CLI local
import flow packages a source tree on the client side, uploads it as
`upload_archive`, and then uses the same `prepare -> validate -> pack ->
publish` workflow that the server already supports.

This design standardizes the system around that boundary and removes the old
server-side git source preparation path.

## Goals

- Make client-side packaging the only import path for externally sourced skills
- Ensure the server no longer clones, fetches, or otherwise retrieves external
  source code during skill import
- Keep the retained packaging workflow stable once an archive reaches the server
- Delete obsolete backend git-import code and its tests
- Update the web import workflow so external import produces
  `source.type=upload_archive`

## Non-Goals

- No redesign of `platform_authoring`
- No redesign of validate, pack, publish, or release-record semantics
- No attempt to preserve backward compatibility for server-side `source.type=git`
- No generic browser-side package-manager feature beyond the skill import flow

## Hard Constraints

- The server must accept only `upload_archive` and `platform_authoring` as
  packaging source types
- `source.type=git` must become a stable rejected request, not a soft fallback
- The client-side archive path must follow the same ignore rules already used by
  the packaging bundle flow for `.git`, `.dist`, and import-report artifacts
- `go test ./...` must pass after the cleanup
- Relevant web tests must pass after the import flow change

## Design Summary

The import boundary becomes:

- client downloads or materializes source content
- client builds a filtered `tar.gz`
- client sends `source.type=upload_archive` with `archiveBase64`
- server extracts, validates, packs, and publishes as it already does today

The server stops understanding "fetch this external source for me" as a valid
packaging request.

## API Contract

### Accepted source types

`POST /api/v1/skill-packaging/prepare` accepts only:

- `upload_archive`
- `platform_authoring`

### Rejected source type

`source.type=git` becomes invalid.

The server returns HTTP `422` with a stable catalog error that clearly states:

- server-side git import has been removed
- callers must package the source on the client side first
- the replacement path is `source.type=upload_archive`

This error is intentionally explicit so stale web clients, scripts, and tests
fail fast and point directly to the new contract.

## Client-Side Flow

### Web external import

For an external skill import, the web application performs the following steps:

1. accept an external source location from the user
2. fetch the source content in the browser-side import workflow
3. materialize a local in-memory file tree for the selected skill subtree
4. filter ignored paths using the retained packaging ignore rules
5. create a `tar.gz`
6. base64-encode the archive
7. call `preparePackaging` with `source.type=upload_archive`

After that point, the workflow remains unchanged:

- `validatePackaging`
- `packSkill`
- `publishSkill`

### Platform authoring

`platform_authoring` remains unchanged. It already matches the desired
boundary because the client provides the file content directly.

### CLI local import

The CLI local import flow already packages locally and uploads
`upload_archive`. That flow remains valid and is not redesigned.

## Backend Changes

### Remove server-side git preparation

Delete the git-specific preparation path from
`internal/packaging/infra/workspace_source.go`, including:

- `prepareGitSource`
- `normalizeGitLocation`
- `runGitCommand`
- git clone timeout support and related environment-driven helpers
- local directory copy behavior that is only reachable through `source.type=git`

After the change, the workspace preparation switch has only two retained paths:

- archive extraction for `upload_archive`
- inline file materialization for `platform_authoring`

### Keep archive extraction and validation

Retain the existing archive extraction path:

- base64 decode
- gzip validation
- tar validation
- safe path enforcement
- symlink rejection

Retain the existing bundle rules and downstream packaging workflow unchanged.

## Frontend Changes

### Replace server-facing git source mode

The import page must stop modeling external import as a server-facing `git`
source mode.

The UI may still let the user think in terms of an external repository URL, but
that input now feeds a browser-side acquisition-and-packaging workflow rather
than a server-side git request.

### Prepare payload shape

For external import, the request sent to the server must use:

- `source.type=upload_archive`
- `source.archiveBase64=<client-generated tar.gz>`

It must not send:

- `source.type=git`
- `source.location=<remote URL>`

### Ignore-rule reuse

The browser-side packager should reuse the existing packaging exclusion rules as
closely as practical so CLI, web, and backend packaging agree on which files
are excluded.

The critical ignored paths are:

- `.git`
- `.dist`
- `.skill-packaging-import-report.json`

The intent is one packaging story, not separate web-vs-CLI interpretations.

## Deletion Scope

### Backend deletions

Delete obsolete git-import behavior and supporting test scaffolding from:

- `internal/packaging/infra/workspace_source.go`
- backend tests that assert successful local or remote git import
- git fixture helpers used only by those success-path tests

### Frontend deletions

Delete the old server-facing git source path from:

- import page source-type handling
- default external-import request construction
- tests that expect `preparePackaging` to receive `source.type=git`

### Documentation deletions

Remove or rewrite wording that implies:

- the catalog server downloads external skill source code
- server-side git import remains a supported contract

## Error Handling

### Browser-side acquisition failure

If the browser cannot fetch or materialize the external source, the UI stops
before calling `preparePackaging` and shows an error that points to the source
location or browser/network restriction rather than the catalog server.

### Browser-side archive build failure

If tarball creation fails in the client workflow, the UI stops before
`preparePackaging` and reports a local packaging failure.

### Rejected legacy source type

If an old client still submits `source.type=git`, the server returns the stable
`422` contract error described above.

### Invalid archive payload

Invalid `upload_archive` payloads continue to use the existing validation
errors for malformed base64, gzip, tar, path traversal, symlink entries, and
unsupported archive entry types.

## Testing

### Backend

Update backend tests to prove:

- `source.type=git` is rejected
- `upload_archive` still succeeds
- `platform_authoring` still succeeds

Delete tests whose only purpose is proving successful server-side local or
remote git import.

### Frontend

Update web tests to prove:

- the default external import flow no longer sends `source.type=git`
- external import sends `source.type=upload_archive`
- failure before archive creation prevents a `preparePackaging` call

### Full verification

Before completion:

- run the relevant web test command for the changed import flow
- run `go test ./...`

## Risks and Mitigations

### Risk: stale clients still send `git`

Mitigation:

- explicit `422`
- explicit replacement guidance in the error body

### Risk: browser packaging behavior diverges from existing CLI behavior

Mitigation:

- reuse the same ignore semantics
- add tests that assert the emitted prepare payload is `upload_archive`

### Risk: cleanup leaves hidden server-side git dependencies behind

Mitigation:

- delete success tests for git import rather than rewriting them to keep the old
  path alive
- search the codebase for `source.type=git`, git-import fixtures, and clone
  helpers as part of implementation

## Success Criteria

This design is complete when all of the following are true:

- the server cannot import a skill by cloning a remote or local git source
- the web external import flow packages client-side and uploads an archive
- `platform_authoring` still works
- the retained packaging workflow after `prepare` is unchanged
- obsolete git-import code and tests are deleted
