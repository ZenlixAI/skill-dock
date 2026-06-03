# Skill Registry Web Cult UI Refresh Design

## Document Status

- Date: 2026-06-03
- Status: Approved in conversation, pending final user review of written spec
- Scope: `apps/skills-web`

## Goal

Refresh the web UI so it feels closer to the visual energy and component rhythm of `cult-ui`, while preserving the registry product's operational clarity.

The selected direction is:

- `frontstage strong style + backstage restrained`

This means:

- `Overview`, `Catalog`, `Skill Detail`, and `Version Detail` become more expressive, editorial, and product-like
- `Import`, `Governance`, `Audit`, and `Authoring Flow` remain denser and calmer, optimized for task completion and scanability

## Why This Change

The current UI already maps the right backend contracts, but it treats most routes as variations of the same glass-card layout. That causes two problems:

1. The discovery-facing routes do not establish enough product identity or hierarchy.
2. The workflow-facing routes do not get enough visual restraint or structural discipline to behave like operator consoles.

This refresh should fix both without changing API ownership, backend semantics, or business rules.

## Non-Goals

- No backend API changes
- No new aggregate endpoints
- No front-end reimplementation of packaging, governance, or install policy logic
- No rewrite onto a large UI framework
- No attempt to copy `cult-ui` source patterns literally
- No fake dashboard behavior that suggests data the backend does not provide

## Constraints

### Product Constraints

- The registry remains a single product shell, not two separate applications.
- Frontstage and backstage routes must share core design tokens so the product still feels unified.
- Detail pages must bridge discovery and action instead of becoming static record views.

### Technical Constraints

- The app stays in the existing `Vite + React` setup.
- Small UI dependencies are allowed only when they materially improve motion or visual polish.
- Large component-system adoption is out of scope.
- Existing route structure and API integration remain intact unless a local front-end-only reorganization is needed.

### Data Constraints

- Errors returned by backend endpoints must still surface directly in the UI.
- The front end continues to render `warningDetails`, `importReport`, governance records, and audit records from server contracts without inventing derived truth.

## Design Direction

### Core Principle

Use one shell with two visual intensities:

- `frontstage`: expressive, layered, editorial, motion-aware
- `backstage`: compact, stable, precise, operator-oriented

The distinction comes from composition and component treatment, not from divergent branding.

### What We Borrow From Cult UI

- stronger editorial composition on key pages
- more deliberate card depth and border rhythm
- selective entrance/reveal motion
- bolder CTA clusters and spotlight modules
- clearer component personality on key discovery surfaces

### What We Do Not Borrow Literally

- heavy all-page animation
- over-stylized form controls in workflow pages
- a large dependency footprint just to mimic example code
- decorative interaction that obscures status, warnings, or audit semantics

## Shared Visual System

### Typography

- Use an editorial serif for high-level headlines and page hero moments.
- Use a restrained sans-serif for product content and dense UI text.
- Use mono accents selectively for registry data such as digests, registries, package names, and system-oriented values.

### Color

- Base direction remains warm and tactile rather than generic white or dark UI.
- Keep an ink-like text color for primary readability.
- Use an ember accent for primary emphasis.
- Use a moss/utility support color for secondary positive states.
- Preserve stronger reds and warning tones for operational risk states.

### Surface Language

- Replace blanket glassmorphism with more deliberate surfaces.
- Use thicker borders, directional gradients, subtle texture/noise, and selective glow.
- Frontstage cards can carry richer depth and visual drama.
- Backstage cards should be flatter, calmer, and easier to scan in sequence.

### Motion

- Frontstage routes can use short route fades, staggered reveals, and stronger hover states.
- Backstage routes should use minimal transitions focused on clarity: hover, focus, open/close, loading, and state change feedback.

## Component Strategy

Create two component families on top of shared tokens.

### Spotlight Components

Used on:

- `Overview`
- `Catalog`
- selected panels in `Skill Detail`
- selected panels in `Version Detail`

Characteristics:

- larger typographic contrast
- richer hover and depth behavior
- more narrative copy hierarchy
- more prominent CTA grouping

### Console Components

Used on:

- `Import`
- `Governance`
- `Audit`
- `Authoring Flow`

Characteristics:

- tighter spacing discipline
- calmer backgrounds
- stronger status grouping
- clearer left-right control/review separation

### Shared Support Components

Unify the treatment of:

- pills
- status indicators
- rails
- metric summaries
- timeline rows
- property lists
- warnings/errors/empty states

These should feel like one coherent system even when the page intensity changes.

## Information Architecture

Keep the product as one shared shell with route-level treatment.

### Frontstage Routes

- `/`
- `/catalog`
- `/skills/:packageName`
- `/skills/:packageName/versions/:version`

### Backstage Routes

- `/import`
- `/governance`
- `/audit`
- `/authoring`

### Shared Shell Requirements

- stronger brand lockup
- clearer primary navigation
- environment/mode context remains visible
- route-level visual intensity is controlled below the shell

## Page-by-Page Plan

### Overview

Role:

- primary product entry
- strongest expression of the refreshed visual language

Changes:

- replace the current balanced hero/rail with a more editorial hero composition
- group CTAs more intentionally around discovery and workflow entry points
- keep curated skills, metrics, governance, and audit, but stage them with clearer hierarchy
- make operator context feel like a supporting rail rather than competing with the hero

Must preserve:

- existing data sources
- recent governance/audit visibility
- direct entry to authoring/import/catalog

### Catalog

Role:

- high-signal discovery surface

Changes:

- promote search into a more prominent command band
- give filters a stronger and calmer structural lane
- make result cards more differentiated and actionable
- improve state hierarchy across `packageName`, `sourceType`, `latestVersion`, and status
- make "what to do next" more obvious from each result row/card

Must preserve:

- current filter semantics
- no front-end governance inference
- current route transitions into skill and version detail

### Skill Detail

Role:

- bridge between discovery and action

Changes:

- keep a frontstage tone, but make the operational next steps more visible
- present identity, install posture, versions, and governance with clearer task flow
- use a stronger secondary action rail for install preview, governance summary, and next-step guidance

Must preserve:

- install target visibility
- version entry points
- skill-level governance action

### Version Detail

Role:

- precise review and execution page for one version

Changes:

- stay visually connected to frontstage pages, but increase operational focus
- make handoff, install impact, release-record action, and governance action feel like one decision surface
- improve separation between passive state reading and active mutation controls

Must preserve:

- release handoff behavior
- release-record form
- governance action semantics
- backend error transparency

### Import

Role:

- backstage workflow console

Changes:

- preserve the explicit `prepare -> validate -> pack` structure
- improve step emphasis and current-stage readability
- stabilize the left-right workflow/review split
- make warnings, import normalization, and error review more scannable
- keep form controls calmer and more utilitarian than discovery-facing surfaces

Must preserve:

- packaging API mapping
- explicit step workflow
- direct rendering of `warningDetails`, `importReport`, `errors`, and bundle metadata

### Governance

Role:

- backstage control surface for governance records and impact reading

Changes:

- retain the layered interpretation approach
- tighten filters, status grouping, and timeline readability
- reduce decorative noise so blocking vs migration semantics are easier to scan

Must preserve:

- target/action filtering
- record layering
- replacement recommendation visibility

### Audit

Role:

- backstage investigation surface

Changes:

- retain the failures/mutations/reads separation
- improve line-item density and readability
- make denied, failed, mutating, and request metadata easier to compare quickly

Must preserve:

- filter semantics
- audit layering logic
- visibility of request metadata and endpoint/action context

### Authoring Flow

Role:

- backstage support route that should visually align with `Import`

Changes:

- align it to the calmer console family
- ensure it inherits the same left-right workflow/review rhythm as the import experience where appropriate

Must preserve:

- current workflow semantics

## Dependency Strategy

Allow only small dependencies that materially improve the target feel.

Acceptable categories:

- motion primitives
- iconography
- minimal utility helpers for transitions or class composition if needed

Not acceptable:

- adopting a large third-party UI system
- replacing the styling architecture with a framework-heavy abstraction
- bringing in dependencies that force large-scale component rewrites unrelated to this refresh

## Implementation Shape

The implementation will likely center on:

1. shared shell and navigation refinements
2. design token refresh in CSS
3. component family split for spotlight vs console surfaces
4. route-by-route layout rewrites
5. light motion enhancement where appropriate

This is a presentation and interaction-clarity refactor, not a behavior rewrite.

## Validation Plan

### Hard Gates

- `npm run type-check`
- `npm run build`

### Manual Verification Focus

- `Overview` and `Catalog` feel visually coherent and meaningfully upgraded
- `Skill Detail` and `Version Detail` still bridge reading and action cleanly
- `Import`, `Governance`, and `Audit` remain easy to operate and do not lose readability
- error, warning, and empty states remain obvious
- desktop and mobile layouts remain usable

## Risks

### Risk: Two-product feel

If frontstage and backstage styling diverge too much, the product may feel split apart.

Mitigation:

- share tokens, shell structure, and support components
- vary intensity, not identity

### Risk: Over-styling workflow pages

If backstage pages inherit too much decorative treatment, operational clarity will regress.

Mitigation:

- use the console component family
- keep denser spacing and lower motion

### Risk: Too little payoff

If the refresh stays too conservative, it will not meaningfully realize the chosen direction.

Mitigation:

- concentrate visual ambition on `Overview` and `Catalog`
- let detail pages act as the transition bridge

## Success Criteria

- The app feels intentionally designed rather than uniformly card-based.
- Frontstage routes have stronger product identity and clearer discovery hierarchy.
- Backstage routes feel calmer and more trustworthy for operational work.
- Existing API-backed workflows remain intact.
- No new backend contract is required to ship the refresh.
