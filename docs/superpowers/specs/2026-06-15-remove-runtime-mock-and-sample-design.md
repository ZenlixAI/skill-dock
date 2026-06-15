# Remove Runtime Mock And Sample Design

## Scope

This design removes non-production runtime features from the backend and its directly coupled tests under:

- `cmd/skill-registry`
- `app/catalogapi`
- `foundation/artifact`
- runtime-facing docs and README sections that mention mock or sample runtime modes

Out of scope:

- `apps/skills-web`
- package-internal test fixtures that are only used as unit-test data helpers
- business-layer behavior unrelated to runtime mock/sample wiring

## Goals

1. Remove runtime mock registry support from the production backend.
2. Remove runtime sample-data and sample-artifact boot modes from the production backend.
3. Remove tests that exist only to exercise those deleted runtime features.
4. Keep ordinary unit/integration tests intact by retaining generic in-memory test fixtures where they are not part of runtime behavior.
5. Leave the production startup path with only real repository, real artifact source, and real packaging wiring.

## Non-Goals

1. Rewrite the entire test suite to avoid all helper data builders such as `catalog.SampleData()`.
2. Remove every test double or in-memory store from the repository.
3. Change public API semantics other than removing `/mock-npm` and the related runtime switches.
4. Redesign catalog domain models.

## Current Problems

The production entrypoint still carries non-production branches:

- `SKILL_REGISTRY_ENABLE_SAMPLE_DATA`
- `SKILL_REGISTRY_ENABLE_MOCK_REGISTRY`
- sample-data seeding
- sample artifact fallback
- `/mock-npm` runtime publishing and tarball serving

That leaves the deployed binary with multiple modes that are useful for local experimentation but not for a production-ready service. It also drags test-only infrastructure into runtime wiring and makes startup harder to reason about.

## Chosen Approach

Remove runtime mock/sample capabilities from the application boundary, but keep generic test fixtures that are not runtime features.

This means:

- production startup no longer supports mock or sample mode toggles
- the server no longer exposes `/mock-npm`
- runtime config no longer accepts mock registry dependencies
- the artifact-layer mock registry implementation is deleted
- tests dedicated to mock registry and runtime sample boot behavior are deleted or rewritten
- test-only data helpers such as `catalog.SampleData()` may remain if they are just fixtures for repository or handler tests

This is the cleanest way to make the service production-only without unnecessarily exploding test rewrite scope.

## Rejected Alternatives

### 1. Remove only `/mock-npm`

This would still leave sample boot and seeded runtime modes in the main binary, which does not meet the production-only goal.

### 2. Remove every helper fixture, including `catalog.SampleData()`

This is broader than necessary. `catalog.SampleData()` is widely used as test input, not only as runtime behavior. Removing it now would turn a runtime cleanup into a repo-wide test rewrite.

## Runtime Design After Cleanup

### Startup

`cmd/skill-registry/main.go` should have a single production-oriented startup path:

- open the configured SQL database
- build the repository with no sample seed input
- choose artifact source from explicit configuration only
- build persistence stores
- start the HTTP server

Removed behavior:

- no `SKILL_REGISTRY_ENABLE_SAMPLE_DATA`
- no `configuredSampleData`
- no implicit sample artifact source fallback
- no `SKILL_REGISTRY_ENABLE_MOCK_REGISTRY`
- no `configuredMockRegistry`

### Artifact Source

Artifact source selection becomes explicit and production-safe:

1. `SKILL_REGISTRY_ARTIFACTS_PATH`
2. `SKILL_REGISTRY_NPM_REGISTRY_URL`
3. otherwise empty/static production-safe source, depending on what the existing repository constructor requires

The startup path must not silently fall back to bundled sample artifacts.

### HTTP Surface

Delete mock registry support end-to-end:

- remove `/mock-npm` route registration
- remove `handleMockRegistry`
- remove `mockRegistryModule`
- remove `ServerConfig.MockRegistry`
- remove `ServerConfig.DisableMockRegistry`
- remove any compatibility logic added only for `/mock-npm`

After cleanup, requests to `/mock-npm` should naturally return `404` because the route no longer exists.

## Code Structure Changes

### Files To Remove

- `app/catalogapi/mock_registry_module.go`
- `foundation/artifact/mock_registry.go`

### Files To Modify

- `cmd/skill-registry/main.go`
- `app/catalogapi/server.go`
- `app/catalogapi/routes.go`
- `app/catalogapi/router/router.go`
- `app/catalogapi/router/system.go`
- `app/catalogapi/handlers_system.go`
- docs that advertise or explain runtime mock/sample modes

### Tests To Remove Or Rewrite

Delete tests that only verify:

- `/mock-npm` metadata, publish, tarball, redirects, or disable/enable behavior
- startup behavior for `SKILL_REGISTRY_ENABLE_SAMPLE_DATA`
- startup behavior for `SKILL_REGISTRY_ENABLE_MOCK_REGISTRY`

Retain tests that use sample data only as ordinary fixture input for unrelated behavior.

## Testing Strategy

The change should be verified in layers:

1. `app/catalogapi`
   - remove mock-registry-specific tests
   - keep normal API tests green
2. `cmd/skill-registry`
   - update startup tests for the production-only path
3. `foundation/artifact`
   - remove mock-registry tests together with the deleted implementation
4. full Go verification
   - `go test ./...`

Important regression checks:

- normal catalog routes still work
- startup still succeeds with explicit real configuration
- `/mock-npm/...` is no longer registered
- no production code references mock registry types or sample runtime env vars

## Risks

1. Some existing tests may use `/mock-npm` indirectly to exercise publishing flows; those tests need either deletion or replacement with production-path tests.
2. Removing sample runtime seeding may expose hidden assumptions in startup or SQL repository initialization.
3. Docs and local workflows may still mention deleted env vars or routes; they must be updated in the same change.

## Rollout Notes

Make this a focused cleanup, not a general refactor:

1. remove runtime config branches
2. remove HTTP mock route and server wiring
3. delete mock registry implementation
4. remove or rewrite feature-specific tests
5. update docs

That keeps the change easy to review and aligned with the production-only goal.
