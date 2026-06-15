# Backend Reorganization Design

Date: 2026-06-15
Status: Approved in conversation, pending written-spec review

## Context

The backend currently exposes one stable HTTP product surface to the Web UI and CLI, but the implementation is spread across several top-level areas:

- `app/catalogapi`
- `business/catalog`
- `business/packaging`
- `foundation/persist`
- `foundation/artifact`
- `app/auth`
- `app/observability`

The main friction is not raw file count alone. The bigger issue is that the current seams make navigation expensive:

- one `catalogapi` module owns routing, handlers, middleware, workflow assembly, cache helpers, and shared HTTP concerns
- the `app` / `business` / `foundation` split is technology-shaped, so understanding one business concern requires bouncing across many directories
- several modules are shallow: their interface is close to their implementation and they mainly forward calls between neighboring layers

This design restructures the backend to improve locality first, increase leverage second, and reduce change/test radius third.

## Goals

- Make backend navigation start from business concepts first: Catalog, Packaging, Governance, System
- Keep the external HTTP API contract stable for the Web UI and CLI
- Deepen modules so each interface hides more implementation detail behind a smaller seam
- Separate HTTP concerns, application orchestration, domain rules, and infrastructure adapters inside each business area
- Reduce the amount of backend behavior that depends on the `catalogapi` top-level module as a central coordinator

## Non-Goals

- No intentional product behavior changes
- No intentional changes to HTTP routes, request payloads, response payloads, or auth semantics
- No frontend or CLI redesign
- No database schema redesign unless required as a mechanical consequence of moving code

## Hard Constraints

- External HTTP API paths remain stable
- Request and response formats remain stable
- Auth behavior remains stable
- Web and CLI integrations must continue to work without consumer-facing changes
- Internal package paths may change
- Key business modules may change package names
- Shared platform modules should remain as stable as practical once introduced

## Design Principles

- Top-level structure follows business domains, because that is the fastest navigation path for maintainers
- Inside a domain, modules are grouped by role: `http`, `application`, `domain`, `infra`
- A module belongs in `domain` only if its implementation is independent of Gin, SQL, file system, npm registry access, or process environment
- A module belongs in `application` if it coordinates multiple seams, enforces workflow ordering, or owns a transaction-like orchestration boundary
- A module belongs in `infra` if it is a concrete adapter for persistence, network, packaging workspace management, or other external systems
- Cross-domain technical concerns live under `internal/platform`
- Shallow pass-through modules should be deleted or merged into a deeper module that earns its interface

## Target Structure

```text
cmd/
  skill-registry/

internal/
  platform/
    auth/
    http/
    persistence/
    artifact/
    observability/

  catalog/
    http/
    application/
    domain/
    infra/

  packaging/
    http/
    application/
    domain/
    infra/

  governance/
    http/
    application/

  system/
    http/
    application/
```

### Why this structure

- It improves locality: a maintainer investigating Catalog can stay under `internal/catalog/...`
- It improves leverage: a domain-facing interface can stay compact while hiding HTTP, SQL, and artifact details behind adapters
- It preserves useful separation between business modules and shared platform modules without forcing a technology-first navigation model

## Module Responsibilities

### `internal/<domain>/http`

Owns:

- route registration
- Gin handlers
- request binding and validation mapping
- response mapping
- domain-specific HTTP middleware, if any

Must not own:

- business rule evaluation
- repository implementation
- direct SQL, npm, or workspace file manipulation

### `internal/<domain>/application`

Owns:

- workflow orchestration
- use-case entry points
- coordination between domain interfaces and adapters
- transaction-like sequencing
- auth collaboration when the workflow needs explicit policy checks

Must not own:

- Gin request/response specifics
- raw persistence details

### `internal/<domain>/domain`

Owns:

- core types
- invariants
- error models
- repository or store interfaces
- business rules

Must not own:

- Gin dependencies
- SQL driver details
- file system or network implementation details

### `internal/<domain>/infra`

Owns:

- SQL repositories
- store implementations
- artifact registry adapters tied to the domain
- workspace and lock adapters tied to the domain

Must not own:

- product HTTP contract
- cross-domain workflow assembly

### `internal/platform/*`

Owns:

- shared auth adapter and session machinery
- shared HTTP concerns such as common middleware, JSON writing helpers, request metadata, observability helpers, rate limiting, and caches used across domains
- shared persistence adapters not owned by one domain
- shared artifact adapters
- shared observability adapters

## File Mapping From Current Structure

### Current `app/catalogapi`

Split by responsibility:

- `handlers/*`, `binding/*`, and `router/*` move into the matching domain `http` module
- `middleware/*`, `common/*`, `response/*`, `request_limiter.go`, and shared cache helpers move primarily into `internal/platform/http`
- `catalog_read_module.go`, `catalog_mutation_module.go`, `read_api_module.go`, `governance_module.go`, `packaging_module.go`, `packaging_access_module.go`, `session_module.go`, and `system/module.go` are reorganized into domain `application` modules
- `server.go`, `server_lifecycle.go`, and `server_mode.go` stop acting as a broad business coordinator and become thin composition utilities near the runtime assembly seam

### Current `business/catalog`

Reorganize into:

- `internal/catalog/domain`
  - core catalog types
  - catalog error model
  - repository and store interfaces
  - business rules that do not depend on transport or storage details
- `internal/catalog/application`
  - catalog read and write workflows
  - orchestration that coordinates repository interfaces and policy seams
- `internal/catalog/infra`
  - SQL repository implementation
  - SQL state store implementation
  - in-memory adapters used for tests and local seams

If a current `read_module` or `write_module` mostly acts as workflow orchestration rather than pure domain rules, it belongs in `application`, not `domain`.

### Current `business/packaging`

Reorganize into:

- `internal/packaging/domain`
  - packaging types
  - warnings
  - bundle rules
  - workspace invariants
- `internal/packaging/application`
  - packaging service orchestration
  - published release workflow
  - prepare, validate, pack, and cleanup coordination
- `internal/packaging/infra`
  - workspace persistence
  - workspace locking
  - file system adapters
  - concrete publish adapters and bridges to artifact infrastructure

### Current `foundation/persist`

Move to `internal/platform/persistence` because these are shared persistence adapters, not an independent product domain.

### Current `foundation/artifact`

Move to `internal/platform/artifact` because these adapters support multiple domains and should present a stable infrastructure seam.

### Current `app/auth`

Move to `internal/platform/auth` because auth is a shared platform module.

### Current `app/observability`

Move to `internal/platform/observability`.

## Runtime Composition

`cmd/skill-registry/main.go` becomes a thin composition root:

- loads environment configuration
- opens shared infrastructure
- constructs platform adapters
- constructs domain adapters
- constructs application modules
- mounts HTTP modules
- starts the HTTP server

It should not embed domain behavior. Its interface should stay narrow and explicit.

## Migration Plan

The user approved a two-step migration strategy.

### Step 1: Restructure Skeleton Without Changing External Contract

- create the new `internal/platform`, `internal/catalog`, `internal/packaging`, `internal/governance`, and `internal/system` layout
- move files into their target modules
- preserve current external HTTP routes, payloads, and auth behavior
- allow short-lived compatibility adapters where needed to keep the code compiling during the move

Exit criteria:

- all Go packages compile
- existing backend tests pass or are mechanically updated for package moves only
- Web and CLI integration points remain unchanged

### Step 2: Deepen Seams and Delete Shallow Modules

- remove broad pass-through `catalogapi` workflow wrappers
- make handlers depend on domain-specific application interfaces instead of a large central server coordinator
- reduce package-level cross-talk
- tighten the composition root so assembly is explicit
- reorganize tests to sit closer to the modules they verify

Exit criteria:

- no broad `catalogapi`-style coordination module remains as the default place to add business behavior
- cross-domain dependencies are intentional and explicit
- test ownership aligns with the new module seams

## Recommended Execution Order

1. Extract `internal/platform/auth`
2. Extract `internal/platform/http`
3. Extract `internal/platform/persistence`
4. Reorganize `internal/catalog`
5. Reorganize `internal/packaging`
6. Reorganize `internal/system` and `internal/governance`
7. Simplify runtime composition in `cmd/skill-registry`

This order reduces risk because shared platform seams are stabilized before the largest business modules move.

## Risks

### Middleware regression risk

The current Gin middleware chain and request context behavior are easy to break during moves. This is the highest regression risk because auth, tracing, canonical redirects, and metadata all pass through this seam.

Mitigation:

- keep dedicated HTTP seam tests for middleware order and context effects
- preserve existing route-level behavior before deleting transitional code

### Over-centralized composition risk

`app/catalogapi/server.go` currently acts as a broad composition and behavior hub. Splitting it too aggressively without replacing it with explicit application interfaces could scatter behavior instead of deepening modules.

Mitigation:

- move orchestration into domain `application` modules before thinning the runtime assembly seam

### Misplaced packaging workflow risk

Some packaging modules are named like services but behave like workflow coordinators. If they are placed into `domain`, the domain interface will stay shallow and pull infrastructure concerns inward.

Mitigation:

- classify packaging code by dependency profile, not by filename

## Verification Strategy

Verification stays at three levels:

### Domain tests

- keep business rule and invariant coverage close to `catalog` and `packaging` domain modules

### HTTP seam tests

- preserve coverage for routing, auth, status codes, redirect behavior, and critical JSON contracts

### Top-level workflow tests

- keep only a small number of server-level or composition-level tests
- do not rebuild a giant integration test bucket around the new layout

## Acceptance Criteria

This design is complete when:

- backend navigation starts from business domains instead of `app` / `business` / `foundation`
- `catalogapi` no longer acts as the default catch-all module
- shared technical adapters are discoverable under `internal/platform`
- the Web UI and CLI continue to work against unchanged external HTTP contracts
- test ownership is closer to the modules under test

## Out of Scope Follow-Up

After the reorganization is complete, a separate review may decide whether any remaining seams justify ADRs or further interface consolidation. That work is not part of this design.
