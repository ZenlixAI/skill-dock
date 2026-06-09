# Catalog K8s HA Design For Read-Heavy Traffic

## Summary

This design optimizes the current `catalog` backend for a read-heavy Kubernetes deployment where:

- search and catalog reads are the dominant traffic shape
- `1-5s` eventual consistency is acceptable for newly published content
- the deployment surface is Helm-based and currently lives in `/Users/ultraman/workspace/skill-registry-helm-chart`
- the system should remain a single binary, but reads and packaging/write traffic should be split at deployment and routing level
- `RWX` shared storage is available for write-side packaging workspaces

The recommended phase-1 architecture is:

- `catalog-read` Deployment:
  - horizontally scalable
  - serves read-only catalog APIs
  - uses short-lived in-memory caches
  - does not mount shared packaging workspace storage
- `catalog-write` Deployment:
  - small replica count
  - serves packaging, publish, cleanup, audit, and mutation APIs
  - mounts `RWX` shared workspace storage
  - enforces stricter concurrency limits
- both workloads use the same PostgreSQL database
- both workloads run the same binary with different mode configuration

## Goals

- Make read traffic scale independently from packaging and publish traffic.
- Prevent heavy packaging requests from reducing read throughput or latency.
- Make `catalog-read` safe for multi-replica Kubernetes deployment.
- Keep the implementation inside one binary and one repository.
- Preserve current PostgreSQL-backed persistence.
- Use Helm as the deployment control surface.

## Non-Goals

- No queue-based asynchronous packaging in phase 1.
- No Redis or external cache in phase 1.
- No read replica database topology in phase 1.
- No frontend or `skills-web` redesign work.
- No eventual-consistency window larger than `5s`.

## Current Constraints

### Runtime

- The server currently runs all routes in one process.
- The entrypoint uses `http.ListenAndServe` directly.
- There is no explicit graceful shutdown path.
- There is no explicit `database/sql` pool tuning.

### Packaging

- The codebase now supports shared workspace persistence across instances when:
  - the database is shared
  - the shared workspace root is shared
  - that root resolves to the same underlying storage across pods
- The runtime already supports a configurable shared workspace root through `SKILL_REGISTRY_SHARED_WORKSPACE_ROOT`.

### Helm

- The current Helm chart already allows `catalog.replicaCount > 1`.
- The chart does not currently:
  - split read and write workloads
  - mount `RWX` shared packaging storage for `catalog`
  - inject `SKILL_REGISTRY_SHARED_WORKSPACE_ROOT`
  - expose separate Services for read vs write routes

This means the chart currently scales the whole API surface together, but does not yet provide a safe operational boundary for multi-replica packaging flows.

## Chosen Approach

### Single Binary, Dual Workload

Keep one binary and introduce an explicit server mode:

- `SKILL_REGISTRY_SERVER_MODE=read`
- `SKILL_REGISTRY_SERVER_MODE=write`
- `SKILL_REGISTRY_SERVER_MODE=all`

`all` preserves local-dev compatibility. Helm production should use `read` and `write`.

This avoids a source-level split into separate services while still creating a real runtime seam between read traffic and heavy write traffic.

### Why This Approach

This is the best fit for the chosen constraints:

- lower implementation cost than a fully separate service split
- materially better fault isolation than a single mixed Deployment
- preserves the current repo and binary shape
- lets Helm become the operational control point
- avoids queueing infrastructure before it is necessary

## Route Surface

### Read Mode

`catalog-read` should register only:

- `/healthz`
- `/readyz`
- `/metrics`
- `/api/v1/skills`
- `/api/v1/skills/search`
- `/api/v1/skills/curated`
- `/api/v1/skills/*` read/detail/version/install routes
- `/api/v1/governance-changes`
- session endpoints only if they are required for the chosen auth mode

`catalog-read` must not serve:

- `/api/v1/skill-packaging/*`
- publish workflows
- audit mutation surfaces
- governance mutation surfaces

If a disallowed route is hit on a read pod, it should fail fast with a clear error instead of attempting partial execution.

### Write Mode

`catalog-write` should register:

- `/healthz`
- `/readyz`
- `/metrics`
- `/api/v1/skill-packaging/*`
- publish flows
- release record mutation flows
- governance mutation flows
- audit endpoints if required operationally
- optionally read APIs for operational simplicity, but they should not be the primary routed target

Operational routing should still direct read traffic away from `catalog-write`.

## Traffic Routing

### Helm-Level Split

The Helm chart should render:

- `catalog-read` Deployment
- `catalog-read` Service
- `catalog-write` Deployment
- `catalog-write` Service

Traefik routing should send:

- read routes to `catalog-read`
- packaging and mutation routes to `catalog-write`

### Route Mapping

Read service:

- `/api/v1/skills`
- `/api/v1/skills/search`
- `/api/v1/skills/curated`
- `/api/v1/skills/`
- `/api/v1/governance-changes`

Write service:

- `/api/v1/skill-packaging/prepare`
- `/api/v1/skill-packaging/validate`
- `/api/v1/skill-packaging/pack`
- `/api/v1/skill-packaging/publish`
- `/api/v1/skill-packaging/cleanup`
- `/api/v1/skill-packaging/bundle`
- any mutation endpoints under `/api/v1/skills/*` that create or change state

## Consistency Model

### Target

The accepted model is low-latency eventual consistency with a `1-5s` visibility window after successful writes.

### Read Cache Strategy

Phase 1 should use per-pod in-memory caches on `catalog-read` only.

Cache TTL guidance:

- search/list/curated pages: `2s`
- skill detail/version detail/install-target: `3-5s`

Write pods do not need the same read cache strategy.

### Cache Rules

Only cache:

- successful read responses
- responses from read-only handlers

Do not cache:

- auth failures
- validation failures
- `5xx`
- packaging responses

Cache keys must include:

- normalized query parameters
- authenticated principal scope relevant to package filtering
- path parameters where applicable

This prevents cross-scope leakage when package authorization differs by caller.

### Invalidation Model

Phase 1 does not use cross-pod push invalidation.

Instead:

- writes commit to PostgreSQL
- read pods continue serving cached results until TTL expiry
- cache convergence happens naturally within the accepted `1-5s` window

This is intentionally simple and aligned with the accepted consistency budget.

## Read-Side Protection

### Concurrency Limits

`catalog-read` should have explicit query concurrency protection so spikes degrade in a bounded way rather than saturating CPU or PostgreSQL connections.

Recommended shape:

- cap concurrent expensive read handlers with a semaphore
- optionally distinguish cheap detail reads from heavier search/list reads
- fail fast with `429` or brief queueing when saturation is reached

### Connection Pooling

Explicit `database/sql` tuning is required.

The application should expose configuration for:

- `max_open_conns`
- `max_idle_conns`
- `conn_max_lifetime`
- `conn_max_idle_time`

Recommended policy:

- `catalog-read`: higher pool ceiling than write
- `catalog-write`: lower ceiling, more conservative

These values should be configurable independently for read and write workloads through env vars and Helm values.

### Metrics

Expose and monitor:

- request rate
- request latency by route family
- cache hit rate and miss rate
- in-flight read concurrency
- database pool stats
- rejection count for read concurrency limiter

## Write-Side Protection

### Shared Workspace Root

`catalog-write` must mount `RWX` shared storage and set:

- `SKILL_REGISTRY_SHARED_WORKSPACE_ROOT=<mounted path>`

`catalog-read` must not mount this shared workspace volume.

### Write Replica Count

Phase 1 recommendation:

- start with `replicas=1`
- allow `replicas=2` only when RWX behavior, locking, and operational characteristics are verified in-cluster

The code is capable of shared workspace persistence, but the operational risk still concentrates on write workloads. This is where caution belongs.

### Packaging Concurrency

`catalog-write` should cap concurrent packaging operations explicitly.

Recommended protection:

- semaphore or bounded worker gate around `prepare/validate/pack/publish`
- lower concurrency target than read side
- clear overload responses under saturation

This protects:

- RWX storage
- CPU and memory on write pods
- PostgreSQL
- Nexus / upstream registry interactions

## Kubernetes HA Strategy

### Read Deployment

Recommended `catalog-read` policy:

- `minReplicas=3`
- HPA enabled
- PDB enabled with `minAvailable: 2`
- topology spread across nodes
- when applicable, spread across zones as well

This makes read traffic resilient to:

- a single pod failure
- node drain
- rolling deployment
- modest burst traffic

### Write Deployment

Recommended `catalog-write` policy:

- start at `replicas=1`
- optional `replicas=2` only after cluster verification
- no aggressive HPA in phase 1
- stronger resource requests and limits than read pods
- separate PDB if multi-replica is enabled

### Probes

Both workloads should use:

- `startupProbe`
- `readinessProbe`
- `livenessProbe`

The readiness contract should ensure pods leave service before shutdown completes.

### Pod Lifecycle

The server must move from raw `http.ListenAndServe` to an explicit `http.Server` with graceful shutdown.

Required behavior:

- trap termination signal
- mark pod unavailable through readiness behavior
- call `Server.Shutdown`
- wait for in-flight requests within `terminationGracePeriodSeconds`

Helm should pair this with:

- `terminationGracePeriodSeconds`
- optional `preStop` delay or drain hook

## Helm Changes

The chart should evolve from one catalog workload to two catalog workloads.

### New Values Shape

Recommended values structure:

- `catalogRead.replicaCount`
- `catalogRead.resources`
- `catalogRead.hpa`
- `catalogRead.pdb`
- `catalogRead.affinity`
- `catalogRead.topologySpreadConstraints`
- `catalogRead.dbPool`
- `catalogRead.cache`
- `catalogRead.concurrency`
- `catalogWrite.replicaCount`
- `catalogWrite.resources`
- `catalogWrite.rwxWorkspace`
- `catalogWrite.dbPool`
- `catalogWrite.concurrency`

The existing `catalog.*` block can be kept temporarily for compatibility, but the long-term target should be explicit `catalogRead` and `catalogWrite` sections.

### Secrets And Env

Both workloads share:

- database URL
- auth tokens
- session config where needed
- npm registry config

Write-only env:

- `SKILL_REGISTRY_SHARED_WORKSPACE_ROOT`
- any packaging concurrency settings

Read-only env:

- cache TTLs
- read concurrency settings

## Rollout Plan

### Phase 1

- add server mode
- gate route registration by mode
- introduce explicit `http.Server` lifecycle
- add configurable DB pool settings
- add read cache and read concurrency limiter
- add write concurrency limiter
- split Helm into `catalog-read` and `catalog-write`
- mount `RWX` only on write

### Phase 2

- tune cache TTLs and pool sizes with production metrics
- evaluate `catalog-write` replica count `2`
- add active cache invalidation only if the `1-5s` window proves too loose

### Deferred

- queue-backed async packaging
- Redis or shared cache
- PostgreSQL read replicas
- separate binaries or repos

## Validation

The implementation is not complete unless all of the following are verified:

- read requests continue succeeding while write pods are saturated
- read traffic scales with `catalog-read` replica count
- write traffic does not require read pods to mount RWX storage
- rolling restart of one read pod does not interrupt read availability
- node drain with PDB leaves read service available
- newly published content becomes visible within the `1-5s` TTL window
- cross-pod packaging flows work correctly on shared RWX storage

## Risks

- RWX storage latency may dominate packaging behavior.
- Short TTL caches may still be too chatty for PostgreSQL under large read bursts if concurrency limits are omitted.
- `catalog-write` at `replicas=2` needs real cluster verification because shared storage correctness and lock timing become operational, not just code-level, concerns.
- Session/auth route placement must be decided carefully if browser login remains in use.

## Recommendation

Implement phase 1 now and stop there before considering queue-backed packaging.

That yields the highest leverage for the chosen traffic shape:

- real read-path horizontal scaling
- better Kubernetes availability
- bounded consistency semantics
- minimal architectural expansion beyond the current codebase shape
