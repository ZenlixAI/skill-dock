# One-Shot Packaging Design

Date: 2026-06-16
Status: Approved in conversation, pending written-spec review

## Context

The current packaging API still exposes a multi-request workspace session:

- `POST /api/v1/skill-packaging/prepare`
- `POST /api/v1/skill-packaging/validate`
- `POST /api/v1/skill-packaging/pack`
- `POST /api/v1/skill-packaging/publish`
- `POST /api/v1/skill-packaging/cleanup`
- `GET /api/v1/skill-packaging/bundle`

That protocol leaks an internal implementation detail into the public contract.
The public caller must understand `workspacePath`, intermediate validation
states, manual cleanup, and bundle-file download. Internally, the backend then
has to preserve cross-request workspace persistence, locking, bundle access, and
expiry cleanup semantics.

That design is now the main architectural mismatch in the packaging area:

- `workspacePath` is not a stable product concept; it is an implementation
  handle
- multi-request workspace persistence makes the server look more distributed
  than it actually is
- bundle download and cleanup routes exist only because the workspace session is
  public
- the current design complicates multi-writer deployment because local or
  shared-file-system assumptions escape into the API boundary

The user explicitly chose a stricter direction: keep the import-review step for
human confirmation, but remove the public workspace-session model entirely.

## Goals

- Replace the public multi-step packaging session with a one-shot, stateless
  packaging contract
- Preserve a review checkpoint before publish
- Remove `workspacePath` and `bundlePath` from the public write-side API
- Reduce backend packaging ownership to stateless orchestration plus temporary
  request-local filesystem helpers
- Delete obsolete cleanup, bundle-download, workspace-persistence, and
  session-lifecycle code
- Keep the client-side import boundary already established for external source
  packaging

## Non-Goals

- No attempt to preserve backward compatibility for the old multi-step packaging
  API
- No redesign of catalog read APIs
- No redesign of package-repository publish semantics or release-record
  semantics
- No distributed shared-workspace adapter
- No retained public debug-only workspace session

## Hard Constraints

- External source import must remain client-packaged via `upload_archive` or use
  `platform_authoring`
- The new public write flow must not expose `workspacePath`
- The new public write flow must not require bundle download from a server-side
  temporary path
- The review step must still expose warnings, import report, manifest digest,
  and components before publish
- `go test ./...` must pass after the redesign
- Relevant web and CLI tests must pass after the contract migration

## Design Summary

The packaging surface becomes two stateless write operations:

1. `import-review`
2. `publish`

`import-review` performs `prepare -> validate -> pack` inside a single request
and returns review data. `publish` accepts a fresh publish request, reruns the
same internal prepare/validate/pack steps inside the request, verifies any
review summary the client supplied, then performs publish and release-record
registration.

The backend still uses a temporary workspace directory internally, but that
directory becomes a request-local implementation detail rather than a public API
contract.

## Public API Contract

### `POST /api/v1/skill-packaging/import-review`

This route replaces the public `prepare`, `validate`, and `pack` progression.

#### Request

The request keeps the same top-level input shape used by `PrepareRequest`:

- `source`
- `target`
- `metadata`
- `options`
- `idempotencyKey`

An optional response-shaping flag may be added under `options` or a dedicated
request field:

- `includeBundleBase64`

That flag defaults to `false`.

#### Response

The response merges the meaningful review data previously spread across
`PrepareResponse`, `ValidateResponse`, and `PackResponse`:

- `ok`
- `packageName`
- `version`
- `manifestDigest`
- `components`
- `warnings`
- `warningDetails`
- `importReport`
- `errors`
- `bundleBase64` only when explicitly requested

The response must not return:

- `workspacePath`
- `bundlePath`

### `POST /api/v1/skill-packaging/publish`

This route remains the public publish entrypoint, but its request becomes
stateless.

#### Request

The request contains:

- `source` or the equivalent archive payload required to reproduce the package
  source
- `target`
- `metadata`
- `options`
- optional `review` summary

The optional `review` summary contains the fields the user previously saw during
the review step:

- `manifestDigest`
- `components`

This summary is used only as a consistency check. The server remains the source
of truth and recomputes the package bundle inside the publish request.

#### Response

The publish response keeps the stable release result fields:

- `packageName`
- `version`
- `manifestDigest`
- `components`
- `publishedToRegistry`
- `created`
- `idempotentReplay`
- `versionStatus`
- `installPolicy`
- `warnings`
- `warningDetails`
- `importReport`
- `errors`

The response must not return:

- `workspacePath`

## Deleted Public Routes

The following public write routes are removed:

- `POST /api/v1/skill-packaging/prepare`
- `POST /api/v1/skill-packaging/validate`
- `POST /api/v1/skill-packaging/pack`
- `POST /api/v1/skill-packaging/cleanup`
- `GET /api/v1/skill-packaging/bundle`

`POST /api/v1/skill-packaging/publish` remains, but its request semantics change
from workspace-based publish to stateless publish.

## Backend Design

### Workspace becomes request-local

The retained filesystem helper behavior stays:

- archive extraction
- inline file materialization
- bundle rules
- package metadata decoding

The retained helper no longer owns cross-request lifecycle concerns.

For both `import-review` and `publish`:

1. create a temporary workspace
2. materialize source into that workspace
3. run validation and bundle generation in that same request
4. clean up before returning

### Application-layer owners

The packaging application layer is reduced to two public orchestration owners:

- `ImportReviewService`
- `PublishService`

`ImportReviewService` owns:

- source preparation
- validation
- pack result generation
- review response composition

`PublishService` owns:

- rerunning source preparation, validation, and pack inside the publish request
- optional review-summary consistency validation
- package-repository publish
- release-record registration and recovery semantics

`WorkspaceService` may remain as an internal helper abstraction only if it
reduces duplication inside these two services. It must not remain the public
application-layer use-case owner.

### Removed backend seams

The redesign deletes the public-session support layer:

- `internal/packaging/http/access_workflow.go`
- `application.AccessService`
- service methods that exist only for public workspace-session lifecycle
- managed workspace persistence and lock orchestration
- background workspace-expiry cleanup semantics

Concretely, the following categories are removed:

- cleanup workflow ownership
- bundle download ownership
- workspace persistence store interfaces and records
- distributed-looking workspace lock and expiry infrastructure
- runtime config that exists only to support shared cross-request workspace
  state, such as shared workspace root semantics for public workspace handles

### Publish consistency check

The redesign accepts repeated validation work as the simpler tradeoff.

To preserve user trust during review-before-publish, the publish path adds one
extra guard:

- if the client sends `review.manifestDigest` or `review.components`
- and the recomputed publish-time values differ
- publish fails with a stable consistency error

This prevents a user from reviewing one package outcome and publishing a
different one.

### Idempotency and recovery

The redesign keeps the existing publish recovery classification:

- registry version already exists
- release record already exists or replays idempotently
- release record can be created after the artifact already exists

Those semantics remain the main protection against duplicate publish attempts.

The redesign does not try to cache cross-request review state in the server.

## Frontend Changes

### Import page

`ImportPage` is simplified from a manual workspace session to a two-step flow:

1. `import-review`
2. `publish`

The page still shows:

- `warnings`
- `warningDetails`
- `importReport`
- `manifestDigest`
- `components`

The page no longer shows or depends on:

- `workspacePath`
- `bundlePath`
- bundle download links
- manual cleanup controls
- manual per-step `prepare`, `validate`, or `pack` actions

The user retains a review checkpoint, but the page no longer acts as a public
workspace debugger.

### Release handoff

Release handoff query parameters and helper types are reduced to stable review
summary data:

- `packageName`
- `version`
- `publishRegistry`
- `manifestDigest`
- `components`

The handoff must not carry:

- `workspacePath`
- `bundlePath`

### Mock client and mock state

The web mock surface is aligned with the new API shape:

- remove mock `prepare`
- remove mock `validate`
- remove mock `pack`
- remove mock `cleanup`
- remove mock `bundle`
- add mock `import-review`
- keep mock `publish` with stateless request handling

This ensures the mock surface does not preserve the old workspace-session
mental model after the real backend has moved on.

## CLI Changes

### `skills publish`

The CLI no longer follows the current registry-first session:

1. `prepare`
2. `validate`
3. `pack`
4. npm publish
5. release record

Instead:

- review-oriented flows call `import-review`
- publish flows call the stateless catalog `publish`

The CLI may still support purely local validation when no registry is involved,
but it must stop modeling registry packaging as a server workspace session.

### Packaging-session helpers

Helpers such as `packaging-session.ts` must stop exposing registry-side
`workspacePath` as their center of control flow.

Registry-mode session results should collapse to:

- review data
- idempotency metadata when still useful
- publishable summary data

They must not expose or depend on:

- `workspacePath`
- bundle download URLs
- bundle path handles

## Documentation Changes

Update documentation to remove the public multi-step packaging story:

- CLI README examples
- API client usage guidance
- any docs that mention `workspacePath`
- any docs that describe bundle download from a temporary packaging route

Documentation must present the new model as:

- client-side source packaging
- one-shot review
- one-shot publish

## Testing

### Backend

Update backend tests to prove:

- `import-review` performs prepare, validate, and pack in one request
- `publish` no longer requires `workspacePath`
- publish rejects mismatched review summary data
- cleanup and bundle-download routes are gone
- managed workspace persistence behavior is removed from the retained contract

Delete tests whose only purpose is proving:

- bundle download from workspace session
- cleanup behavior for public workspace handles
- cross-request managed workspace persistence

### Frontend

Update web tests to prove:

- import page runs `import-review`
- review results render without `workspacePath`
- publish request sends stateless material
- no bundle-download or cleanup controls remain

### CLI

Update CLI tests to prove:

- review flows call `import-review`
- publish flows call stateless `publish`
- registry-mode packaging no longer expects `workspacePath`

### Full verification

Before completion:

- run `go test ./...`
- run relevant web type-check and test suites
- run relevant CLI test suites

## Expected Outcome

After this redesign:

- packaging review and publish remain available to users
- public API callers no longer manage workspace sessions
- backend packaging no longer needs to preserve a fake distributed workspace
  contract
- write-side deployment constraints become local to a single request rather
  than encoded into the public API
