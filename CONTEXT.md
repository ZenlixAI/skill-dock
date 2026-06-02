# Agent Skill Registry

Defines the language for a lightweight registry that lets teams publish, discover, and install reusable agent skills. It exists to keep product-facing skill concepts distinct from distribution and governance mechanics.

Current authoritative docs live in:

- `README.md`
- `docs/adr/`
- `docs/superpowers/specs/2026-05-28-agent-skill-registry-overview.zh-CN.md`
- `docs/superpowers/specs/2026-05-28-agent-skill-registry-api-cli-contracts.zh-CN.md`
- `docs/superpowers/specs/2026-05-28-agent-skill-registry-catalog-api-detailed-design.zh-CN.md`
- `docs/superpowers/specs/2026-05-28-agent-skill-registry-skill-packager-importer.zh-CN.md`
- `docs/superpowers/specs/2026-05-28-agent-skill-registry-publish-release-governance.zh-CN.md`
- `docs/superpowers/specs/2026-05-28-agent-skill-registry-local-installer-adapter.zh-CN.md`
- `docs/superpowers/specs/2026-05-28-agent-skill-registry-skills-sh-compatibility.zh-CN.md`
- `docs/superpowers/specs/2026-05-28-agent-skill-registry-catalog-api.openapi.yaml`

## Language

**Agent Skill**:
A reusable capability unit whose public identity is its **Package Name**.
_Avoid_: Bundle, plugin, asset

**Package Name**:
The npm-compatible identifier that uniquely names an **Agent Skill** in V1.
_Avoid_: Skill ID, external ID

**Package Scope**:
The npm namespace segment of a **Package Name**, such as `@platform.skill`.
_Avoid_: Installation Scope, registry name

**Skill Package Scope**:
A reserved **Package Scope** used to identify publishable skill packages in V1.
_Avoid_: Generic npm scope, optional naming hint

**Display Name**:
The human-readable name shown to users for an **Agent Skill**.
_Avoid_: Stable identity, package coordinate

**Search Result**:
The catalog listing entry shown to users when discovering an **Agent Skill**.
_Avoid_: Raw package record, version row

**Skill Search Index**:
The search view whose primary entries are **Agent Skills** rather than individual versions.
_Avoid_: Version search table

**Read Compatibility**:
The decision to align discovery-oriented read APIs with skills.sh without inheriting its write-side or identity model.
_Avoid_: Full protocol cloning

**CLI Compatibility**:
The decision to reuse skills.sh command language and user mental models without reusing its source-first install target format.
_Avoid_: Source-first protocol reuse

**Package-Only CLI Input**:
The rule that V1 CLI commands accept only **Package Name** as the positional skill identifier.
_Avoid_: Repo path guessing, bare name guessing

**Catalog Database**:
The primary relational store that holds skill records, version records, and governance changes in V1.
_Avoid_: Pluggable search stack, package repository

**Agent Skill Aggregate**:
The full catalog state for one **Agent Skill**, loaded and saved as one concurrency unit, including skill detail, all **Skill Versions**, lifecycle state, and **Governance Changes** needed for install resolution.
_Avoid_: Raw row collection, partial version slice

**State Store**:
The persistence seam beneath the catalog repository that loads and saves one **Agent Skill Aggregate** and provides read indexes for listing and filtering.
_Avoid_: Business-state owner, artifact intake module

**Source Export**:
An optional conversion that renders an internal skill package into a source-first layout for ecosystems such as skills.sh.
_Avoid_: Native publish path

**Lossy Export**:
An export that preserves author-facing source content but does not preserve full catalog governance state for round-trip recovery.
_Avoid_: Full-fidelity backup

**Author Metadata**:
The minimal, author-maintained fields in `SKILL.md` that describe an **Agent Skill** for humans.
_Avoid_: Catalog state, distribution identity

**Derived Components**:
The bundle capabilities inferred by catalog-side scanning from the published artifact contents.
_Avoid_: Author-declared component list

**Skill Version**:
An immutable published release of one **Agent Skill**.
_Avoid_: Revision, mutable release

**Published Version**:
A **Skill Version** that is eligible for normal install resolution.
_Avoid_: Draft, hidden version

**Deprecated Version**:
A **Skill Version** that remains explicitly installable but is excluded from default `latest` resolution.
_Avoid_: Default latest, hard removed version

**Yanked Version**:
A **Skill Version** that is excluded from normal install resolution and normal installation flows.
_Avoid_: Deprecated version

**Retired Skill**:
An **Agent Skill** that no longer participates in default install resolution as a live catalog entry.
_Avoid_: Yanked version

**Skill Bundle**:
The packaged distribution artifact that carries one **Agent Skill Version**.
_Avoid_: Agent Skill, source directory

**Single-Skill Bundle**:
A **Skill Bundle** that distributes exactly one **Agent Skill** in V1.
_Avoid_: Skill collection, multi-skill pack

**Compatibility Attachment**:
A bundle file kept for ecosystem interoperability without becoming a first-class catalog concept.
_Avoid_: Core domain object

**Published Release**:
A **Skill Version** that has both a stored **Skill Bundle** and a successful catalog registration.
_Avoid_: Tarball exists, npm-only publish

**Release Record**:
The catalog-side registration of a published artifact that already exists in the package repository.
_Avoid_: Artifact upload, package storage

**Orphaned Artifact**:
A stored **Skill Bundle** that exists in the package repository without a successful catalog registration.
_Avoid_: Published release

**Idempotent Registration**:
A repeated **Release Record** request for the same **Package Name** and **Skill Version** that succeeds only when the artifact content matches the existing record.
_Avoid_: Conflicting re-publish

**Installed Skill**:
An **Agent Skill Version** that has been placed in a target scope and is visible to local discovery.
_Avoid_: Downloaded archive, unpacked but undiscoverable files

**Update Resolution**:
The act of resolving an already installed skill to the catalog's current default installable version for the same package.
_Avoid_: Semver-range upgrade, index-only refresh

**Installation Scope**:
The local target boundary, such as global, project, or agent, within which installed skills are discovered.
_Avoid_: Registry scope, package scope

**Local Removal**:
The act of uninstalling an **Installed Skill** from one local **Installation Scope** without changing catalog state.
_Avoid_: Remote retirement, catalog governance

**Scope Precedence**:
The fixed order used to choose between multiple installed copies of the same package across local installation scopes.
_Avoid_: Filesystem-dependent lookup

**Default Install Scope**:
The local installation scope chosen by the CLI when the user does not explicitly select one.
_Avoid_: Implicit global install, context guessing

**Installed View**:
The CLI listing of locally installed skills across one or more installation scopes.
_Avoid_: Effective-only hidden state

**Source Type**:
The factual origin classification of an **Agent Skill**, such as first-party, imported, or community.
_Avoid_: Install permission, policy outcome

**Install Policy**:
The catalog rule that determines whether a **Skill Version** participates in normal install resolution.
_Avoid_: Source Type, recommendation label

**Replacement Recommendation**:
An optional catalog hint that points from an unavailable **Agent Skill** to a suggested alternative.
_Avoid_: Required retirement dependency

**Install Resolution Failure**:
A structured catalog response that explains why an install target could not be resolved.
_Avoid_: Generic not found, opaque error

**Install Target**:
The resolved package repository coordinates and verification data needed to fetch a skill artifact.
_Avoid_: Mandatory tarball direct link

**Registry Entry Point**:
The single catalog-facing endpoint that a CLI user configures for discovery and installation in V1.
_Avoid_: Separate user-managed catalog and npm registry endpoints

**Default Package Repository**:
The single npm-compatible repository that a V1 catalog resolves against for published skill artifacts.
_Avoid_: Multi-repository routing layer

**Governance Change**:
A catalog action that changes lifecycle or installability state for a skill or version.
_Avoid_: Silent status flip

**Governance Reason**:
The recorded explanation for a **Governance Change**.
_Avoid_: Missing rationale

**Authenticated Read**:
A read operation that requires a recognized caller identity unless the deployment explicitly chooses to open it.
_Avoid_: Anonymous-by-default private access

**Reader Role**:
The minimal V1 role allowed to search, inspect, and resolve installs from the catalog.
_Avoid_: Governance operator

**Publisher Role**:
The minimal V1 role allowed to publish releases and perform governance changes, while also retaining read capabilities.
_Avoid_: Read-only user

**Version Request**:
The version selector supplied to install resolution, which in V1 is either empty for default latest resolution or an exact version.
_Avoid_: Semver range, floating constraint

**Upstream Source**:
The external origin from which an **Agent Skill** may be imported or rebundled.
_Avoid_: Alternate identity, second package name

**Upstream Version**:
The version identifier from an external source that may be preserved when creating an internal **Agent Skill Version**.
_Avoid_: Mutable internal revision

## Relationships

- One **Agent Skill** is identified by exactly one **Package Name** in V1
- One publishable **Agent Skill** in V1 must use a reserved **Skill Package Scope**
- One **Agent Skill** may present one mutable **Display Name**
- One **Agent Skill** has one set of minimal **Author Metadata** in `SKILL.md`
- One **Derived Components** value is produced by catalog-side scanning rather than author declaration
- One **Search Result** presents **Display Name** as the primary label and **Package Name** as the stable reference
- One **Skill Search Index** returns one result per **Agent Skill**, not one result per **Skill Version**
- One **Read Compatibility** scope aligns only discovery-style read APIs with skills.sh
- One **Read Compatibility** scope does not introduce a second public identity beside **Package Name**
- One **CLI Compatibility** scope reuses command language such as add/find/list/remove/update without adopting source-first package identity
- One **Package-Only CLI Input** requires positional install and update targets to use **Package Name**
- One **Catalog Database** is the default V1 source for metadata storage and minimal search behavior
- One **State Store** persists one **Agent Skill Aggregate** as the catalog repository's concurrency unit
- One **Source Export** is an optional downstream adaptation, not the native registry publish model
- One **Source Export** is a **Lossy Export** rather than a full round-trip representation of catalog state
- One **Agent Skill** has one or more **Skill Versions**
- One version listing for an **Agent Skill** is sourced from catalog records, not raw package-repository enumeration
- One **Skill Version** is distributed as one **Single-Skill Bundle**
- One **Single-Skill Bundle** contains exactly one root `SKILL.md` and no nested `SKILL.md` files
- One `plugin.json` in V1 is a **Compatibility Attachment**, not a first-class catalog concept
- One **Agent Skill** may reference zero or one **Upstream Source**
- One **Published Release** requires both bundle storage and catalog registration
- One **Release Record** registers an existing artifact rather than uploading it through the catalog
- One **Idempotent Registration** is allowed only when repeated registration points to the same artifact content
- One **Orphaned Artifact** is not searchable or installable through normal catalog flows
- One **Installed Skill** requires both local placement and successful discovery in its target scope
- One **Update Resolution** targets the current default installable version for an already installed **Package Name**
- One **Installation Scope** contains at most one installed version for a given **Package Name**
- One **Local Removal** affects only local installation state within the chosen **Installation Scope**
- One **Scope Precedence** prefers agent scope over project scope over global scope
- One **Default Install Scope** is project scope unless the user explicitly selects agent or global
- One **Installed View** includes installed copies across scopes and marks which copy currently wins by precedence
- One **Source Type** describes origin only and does not decide installability
- One **Install Policy** decides whether a **Skill Version** is normally resolvable
- One **Replacement Recommendation** may accompany an unavailable **Agent Skill**, but is not required to retire it
- One **Install Resolution Failure** distinguishes governance and lookup failures instead of collapsing them into a generic missing record
- One **Install Target** is primarily defined by package name, version, registry, and integrity rather than a required tarball direct URL
- One **Registry Entry Point** is the only registry endpoint a CLI user must configure in V1
- One **Default Package Repository** is the single package repository a V1 catalog resolves against
- One **Governance Change** records at least its new state, **Governance Reason**, actor, and timestamp
- One **Governance Change** is a unified action family across skill-level and version-level lifecycle updates
- One **Authenticated Read** is the default V1 posture for catalog queries and install resolution in private deployments
- One **Reader Role** covers catalog discovery and install resolution
- One **Publisher Role** covers release registration and governance changes in addition to reader capabilities
- One **Version Request** in V1 is either empty or an exact version
- One **Skill Version** must represent immutable content for its **Package Name**
- One imported **Upstream Version** may be preserved only when the internal release keeps version-content immutability intact
- Default `latest` resolution selects the highest semver among **Published Versions** only
- A **Deprecated Version** may be installed only when explicitly requested
- A **Yanked Version** is outside normal installation flows
- A **Yanked Version** remains visible in version history with explicit status
- A **Retired Skill** remains queryable in detail and version history even though it is outside default install resolution
- A **Retired Skill** is excluded from default search results unless explicitly requested

## Example dialogue

> **Dev:** "If `@community/pdf-skill` is repackaged as `@platform/pdf-skill`, is that still the same **Agent Skill**?"
> **Domain expert:** "No. In V1 those are two **Agent Skills** with different **Package Names**; the relationship is captured as an **Upstream Source**, not a shared identity."

> **Dev:** "Can we rename the catalog entry from `PDF Tooling` to `PDF Workflow Toolkit` without changing installs?"
> **Domain expert:** "Yes. **Display Name** may change, but **Package Name** remains the install identity."

## Flagged ambiguities

- "`id`" and "`package_name`" were both used as public skill identity — resolved: V1 exposes **Package Name** as the only public identity; any database `id` stays internal.
- "`scope`" could mean npm namespace or local install boundary — resolved: use **Package Scope** for names like `@xxx.skill/` and **Installation Scope** for global/project/agent targets.
- "`@xxx.skill/`" could be treated as a loose convention — resolved: V1 treats it as a required **Skill Package Scope** for publishable skill packages.
- "`name`" in `SKILL.md` could be mistaken for install identity — resolved: it is the **Display Name** snapshot, not the public identity.
- "search results" could default to raw package coordinates — resolved: **Search Results** lead with **Display Name** and include **Package Name** as the stable identifier.
- "search" could return version rows — resolved: the **Skill Search Index** is skill-level; version details are expanded separately.
- "`skills.sh compatibility`" could imply copying its full identity and write protocol — resolved: V1 keeps only **Read Compatibility** with its discovery APIs.
- "`skills.sh compatibility`" could justify a parallel `{source}/{skill}` identity — resolved: V1 does not add any second public identity beyond **Package Name**.
- "`skills.sh CLI reuse`" could imply reusing `owner/repo` install targets and source-first distribution — resolved: V1 keeps only **CLI Compatibility** at the command-language level.
- "`skills add`" could accept repo paths or bare names for convenience — resolved: V1 enforces **Package-Only CLI Input** and rejects source-first identifiers.
- "`Catalog Index`" could imply a pluggable search architecture from day one — resolved: V1 prefers one relational **Catalog Database** with minimal built-in search over multiple backend abstractions.
- "`skills.sh compatibility`" could imply direct publishability — resolved: internal packages require an optional **Source Export** step before entering source-first ecosystems like skills.sh.
- "`source export`" could be mistaken for a full backup format — resolved: exported source is lossy and does not guarantee round-trip recovery of catalog governance state.
- "version listing" could be read directly from npm storage — resolved: visible versions come from catalog records, excluding orphaned artifacts while retaining governed states like yanked.
- "`SKILL.md`" could grow a catalog-specific extension block — resolved: V1 keeps only minimal **Author Metadata** there and does not define `x-catalog`.
- "`components`" could be treated as author-entered metadata — resolved: V1 stores **Derived Components** from catalog-side scanning only.
- "`deprecated`" was described as installable without saying whether it remains the default latest — resolved: default resolution ignores **Deprecated Versions** and selects only **Published Versions**.
- "`publish`" could mean npm storage or catalog visibility — resolved: a release is published only after catalog registration succeeds; npm-only artifacts are **Orphaned Artifacts**.
- "`publish` API" could imply artifact upload through the catalog — resolved: catalog publish is a **Release Record** for an artifact already stored in the package repository.
- "publish retry" could be treated as either duplicate success or duplicate corruption — resolved: **Idempotent Registration** succeeds only for content-identical retries and rejects conflicting content under the same package/version.
- "`bundle`" could imply a pack containing multiple public skills — resolved: V1 allows only **Single-Skill Bundles**; multi-skill packs are invalid.
- "`skills/`" inside a bundle could hide extra skill definitions — resolved: V1 rejects nested `SKILL.md` files and any bundle shape that implies more than one public skill.
- "`plugin.json`" could pull plugin semantics into the catalog model — resolved: V1 treats it only as a **Compatibility Attachment** and does not model its internals as first-class registry concepts.
- "`add`" could mean only downloading files — resolved: installation succeeds only when the **Installed Skill** is locally discoverable in the target scope.
- "`update`" could imply semver negotiation or index refresh only — resolved: it re-resolves an installed package to the catalog's current default installable version and replaces the local installation when needed.
- "`update`" could silently do nothing when governance blocks progress — resolved: update surfaces structured failure reasons such as retired or no resolvable target instead of silently succeeding.
- "`remove`" could imply remote governance changes — resolved: it performs only **Local Removal** and never changes catalog lifecycle state.
- "multiple local installs" could resolve by accident — resolved: **Scope Precedence** is fixed as agent, then project, then global.
- "`add`" without an explicit scope could guess too broadly — resolved: the **Default Install Scope** is project, not global and not agent inference.
- "`list`" could hide overridden installs — resolved: the **Installed View** shows installed copies across scopes and indicates the effective winner.
- "installed versions" could imply npm-style coexistence — resolved: each **Installation Scope** permits only one installed version per **Package Name**.
- "`source_type`" could be misused as an install gate — resolved: **Source Type** records origin, while **Install Policy** controls normal install resolution.
- "`version`" input could imply npm-style semver ranges — resolved: V1 accepts only an exact version or no version at all.
- "imported version" could be reused after internal content changes — resolved: version immutability is preserved per **Package Name**; repackaged imports use a different internal **Package Name** rather than mutating version meaning.
- "`yanked`" could imply hidden history — resolved: yanked versions stay visible in version listings with explicit status, while remaining outside normal install resolution.
- "`retired`" could imply logical deletion — resolved: retired skills remain visible in detail and version history, but stay outside default install resolution.
- "searchable history" could imply inclusion in normal discovery — resolved: retired skills are hidden from default search and shown only when explicitly requested.
- "retire" could require a 1:1 successor — resolved: **Replacement Recommendations** are optional guidance, not a prerequisite for retirement.
- "install failure" could be reported as a generic 404 — resolved: install resolution returns structured failure reasons such as retired, yanked, exact-version-missing, or policy-blocked.
- "`tarballUrl`" could be treated as the required install contract — resolved: the stable **Install Target** is package name, version, registry, and integrity; direct tarball URLs are optional.
- "`registry`" could require separate user-facing catalog and npm endpoints — resolved: V1 exposes one **Registry Entry Point** and derives package-repository details through install resolution.
- "`registry`" could imply a catalog routing across many package repositories — resolved: V1 uses one **Default Package Repository** and handles other sources through import/sync rather than runtime multi-repository resolution.
- "status change" could be stored without explanation — resolved: every **Governance Change** records a **Governance Reason**, actor, and timestamp.
- "deprecate/yank/retire" could be modeled as unrelated operations — resolved: they are all **Governance Changes** within one lifecycle governance model.
- "read API" could be anonymous by default even in private registry deployments — resolved: V1 defaults to **Authenticated Read**, with anonymous access only as an explicit deployment choice.
- "permissions" could expand into fine-grained workflow roles in V1 — resolved: V1 keeps only **Reader Role** and **Publisher Role**.
- "`x-catalog`" could be used as a substitute for registry state — resolved: V1 has no `x-catalog`; catalog-specific data comes from publish input and catalog-side scanning/governance.
