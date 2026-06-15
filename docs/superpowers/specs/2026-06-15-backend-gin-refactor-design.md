# Backend Gin Refactor Design

## Scope

This design covers backend HTTP adapter cleanup and local developer workflow simplification under:

- `app/catalogapi`
- `cmd/skill-registry`
- `Makefile`

Out of scope:

- `business/catalog`
- `business/packaging`
- `foundation/*`
- `apps/skills-web`
- changing API semantics or response payload shapes unless required for framework migration

## Goals

1. Split backend HTTP code by feature/module instead of keeping route registration, handlers, middleware, and request parsing in one package surface.
2. Replace custom HTTP glue that overlaps with common framework features with `gin`, especially route registration, path/query/JSON binding, and middleware chaining.
3. Keep the existing business workflows and error model intact while reducing adapter boilerplate.
4. Simplify `Makefile` to retain only meaningful build, test, verification, deployment, and primary dev entrypoints.
5. Remove all CLI-oriented make targets and compatibility aliases.

## Non-Goals

1. Rework repository, persistence, or packaging domain boundaries.
2. Redesign auth or audit policy semantics.
3. Introduce a new dependency injection framework.
4. Change CLI behavior or add new backend features as part of this refactor.

## Problems In The Current Structure

The current backend adapter is still centered on `net/http` plus custom glue:

- `routes.go` manually registers every endpoint and contains package/version path adaptation logic.
- `middleware.go` reimplements middleware composition and request context plumbing.
- handler files repeatedly parse query strings, decode request bodies, and perform method checks.
- `common/params.go` contains parsing code that partially overlaps with what a framework binder can already do.
- `Makefile` mixes real entrypoints with many fine-grained dev aliases and CLI convenience commands that are not needed for the backend workflow.

This makes the adapter layer larger than it needs to be and obscures feature boundaries.

## Chosen Approach

Use a gradual HTTP-adapter refactor:

1. keep the existing `Server` composition and business modules
2. replace `http.ServeMux` routing with `gin.Engine`
3. move handler code into feature-oriented directories
4. move generic HTTP concerns into gin middleware
5. keep existing catalog error generation and response payloads

This captures most of the maintainability gain without forcing a domain rewrite.

## Rejected Alternatives

### 1. Keep `net/http` and only move files

This lowers immediate change risk but preserves the custom routing, binding, and middleware code that caused the current adapter sprawl.

### 2. Full architectural rewrite across app/business/foundation

This would exceed the request. The current business seams are already usable; the main duplication sits in the HTTP adapter layer.

## Target Structure

`app/catalogapi` will be reorganized around HTTP responsibilities:

- `app/catalogapi/router`
  - engine construction
  - route group registration by feature
- `app/catalogapi/middleware`
  - CORS
  - request and trace metadata
  - observability
  - auth/session principal injection
  - action guard helpers
- `app/catalogapi/handlers/catalog`
  - skills list/detail/version/install-target endpoints
- `app/catalogapi/handlers/governance`
  - governance and catalog mutation endpoints
- `app/catalogapi/handlers/packaging`
  - packaging prepare/validate/pack/publish/cleanup/bundle endpoints
- `app/catalogapi/handlers/session`
  - session current/login/logout endpoints
- `app/catalogapi/handlers/system`
  - health, readiness, metrics
- `app/catalogapi/binding`
  - request DTOs and validation helpers that remain useful beyond basic gin binding
- `app/catalogapi/response`
  - JSON and catalog error writing helpers adapted for gin

The exact file count can stay pragmatic. The important boundary is responsibility by feature and HTTP concern, not splitting into tiny files for its own sake.

## Gin Migration Rules

### Routing

- Use `gin.Engine` and `gin.RouterGroup` for route registration.
- Group routes by capability:
  - system
  - session
  - catalog read
  - catalog write
  - packaging
- Preserve the current endpoint paths and methods.
- Preserve mode-based route exposure (`read-only`, `write`, etc.).

### Binding

- Replace manual path extraction with `c.Param(...)`.
- Replace manual JSON decoding with `ShouldBindJSON` where the payload maps cleanly to a request struct.
- Replace simple query parsing with `ShouldBindQuery` where only scalar binding is needed.
- Keep explicit helper validation for domain-specific rules such as:
  - cursor token semantics
  - exact semver validation
  - governance filter enums
  - package access checks

The rule is: let gin handle transport-level binding, keep domain validation in local helpers.

### Middleware

Rebuild existing wrappers as gin middleware:

- `withCORS`
- request id / trace id injection
- observability/status capture
- auth principal resolution
- action guards for protected routes

These middlewares must continue to populate the same downstream context values expected by existing modules.

### Error Handling

- Keep `catalog.Error` as the canonical API error model.
- Adapt current response helpers to gin so handlers can return the same payload contract and status codes.
- Avoid introducing a generic gin error envelope that would break clients.

### Streaming And Raw Responses

Endpoints that already need raw `http.ResponseWriter` behavior, especially bundle download and metrics, may use gin APIs that expose the underlying writer where necessary. The migration should not force JSON-only abstractions onto streaming responses.

## Server Construction

`catalogapi.NewServer` will continue to own dependency assembly for workflows and modules.

The main change is:

- `Server.Handler()` returns a gin-backed `http.Handler`
- route setup moves out of the main server file into router-oriented code

`cmd/skill-registry/main.go` should need only minimal updates to work with the new handler construction.

## Makefile Simplification

Keep only high-signal targets for backend and repo development:

- `make dev`
- `make deploy` or the existing primary deployment entrypoint
- `make build`
- `make verify`
- `make test-go`
- `make test-web`
- `make build-web`

Optional supporting targets are acceptable only if they are directly used by the retained top-level commands.

Remove:

- all `cli*` targets
- compatibility aliases for CLI flows
- extra fine-grained dev lifecycle commands that only expose implementation details of `dev`
- verbose generated help text describing deleted commands

If a removed target still contains useful logic, fold that logic under `dev`, `deploy`, `build`, or `verify` instead of keeping a public alias.

## Testing Strategy

The refactor should be guarded with focused adapter-level tests first, then broader verification:

1. route registration tests for representative read, write, session, and packaging endpoints
2. handler tests covering:
   - path param extraction
   - query binding
   - JSON binding
   - auth and package guard behavior
3. middleware tests for:
   - CORS
   - request id and trace id propagation
   - unauthorized responses
   - observability capture
4. regression runs:
   - `go test ./app/catalogapi`
   - `go test ./...`

`Makefile` changes should be validated by ensuring retained targets still map to working commands.

## Risks

1. Gin migration can subtly change request parsing or status behavior if handlers rely on low-level `net/http` edge cases.
2. Middleware ordering mistakes could break principal injection, observability, or audit logging.
3. Overusing binding tags could blur transport validation and domain validation; the implementation must keep that boundary explicit.
4. Aggressive `Makefile` deletion could remove commands still used by existing docs or local workflows; retained targets should cover the practical developer path.

## Rollout Notes

This refactor should be delivered in small internal steps:

1. add gin dependency and router shell
2. migrate middleware
3. migrate one feature group at a time
4. remove obsolete `net/http` adapter helpers once replacement coverage exists
5. simplify `Makefile` after the new backend entrypoints are stable

That keeps diffs reviewable and limits regression surface during the transition.
