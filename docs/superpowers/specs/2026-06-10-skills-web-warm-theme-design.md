# Skills Web Warm Theme Design

## Scope

This design covers the existing web frontend under:

- `apps/skills-web`

Out of scope:

- route behavior, page structure, or component APIs
- copy changes unrelated to theme presentation
- typography or layout redesign beyond what existing styles already require
- introducing new frontend dependencies

## Goals

1. Replace the current mixed cool-accent palette with a single warm-accent theme across the full web experience.
2. Keep the existing dark interface model while shifting the base surfaces, borders, shadows, and emphasis colors to a warmer visual language.
3. Ensure the portal, detail pages, legal pages, and console read as one system rather than separate color families.
4. Preserve readability, focus visibility, and semantic status signaling after the palette change.
5. Keep the implementation reviewable by concentrating it in theme tokens and existing style consumers.

## Non-Goals

1. Rebuild page layouts or change route-specific information architecture.
2. Redesign the component library or add a separate theming system.
3. Remove semantic success, warning, and error colors entirely.
4. Perform a broader visual refresh covering fonts, spacing, or content tone.

## Design

### 1. Global palette consolidation

The root theme in `apps/skills-web/src/styles.css` will move from blue-black with cyan and moss accents to a unified warm-dark palette.

The new structure is:

- deep background colors shift from navy-blue to charcoal with a subtle brown undertone
- panel and border colors become warm neutrals derived from the same base family
- a single copper/amber accent becomes the primary interactive highlight
- shadow colors become brown-tinted rather than blue-black

The current secondary accent family (`--moss`, matching gradients, and similar cool tints) will stop acting as a peer visual accent. If a non-primary tint remains for status semantics or atmospheric depth, it must stay subordinate to the warm accent and never compete with it at the page level.

### 2. Shared accent behavior across routes

Route themes will continue to distinguish page contexts structurally, but they will no longer imply different color families.

This means:

- portal pages, console pages, detail pages, and legal pages all inherit the same warm base palette
- route-specific shells can still vary in opacity and intensity
- route-specific backgrounds must stay within the same charcoal-to-warm-neutral range
- no route should reintroduce a cold blue or teal halo as its dominant highlight

This preserves the existing route model without fragmenting the brand language.

### 3. High-exposure component recoloring

The palette shift must be carried through the components that most visibly expose theme decisions. The implementation will keep the same class names and component interfaces, while updating their style rules to consume the new tokens consistently.

Primary targets:

- shell framing: `app-shell`, `shell-nav`, `shell-footer`, halo layers, and grain overlays
- navigation states: `nav-chip`, active states, hover states, and focus-visible treatment
- action controls: `button` variants, especially `button--primary`, `button--secondary`, and `button--ghost`
- form controls: `input`, `select`, `textarea`, and `field` focus outlines
- content surfaces: `surface-card` variants, `skill-summary-card`, `console-entry-card`, `list-card`, and metric-style surfaces
- status treatments: `status-pill--accent` plus any neutral pill backgrounds that currently skew cool

The rule is that warm emphasis is reserved for interaction, active selection, and visual guidance. Large passive surfaces should remain restrained so the console does not become visually noisy.

### 4. Atmospheric background cleanup

The current design uses halos, radial gradients, and translucent panels to create depth. That structure stays, but the color story changes.

Implementation requirements:

- replace cyan or teal radial glows with warm amber or copper glows
- keep grain and blur layers, but ensure they support a dark neutral base instead of a “blue tech” look
- keep gradients asymmetrical and soft so the page still has dimensionality without feeling saturated
- maintain tinted shadows and translucent surfaces so the interface remains layered rather than flat

The target aesthetic is a disciplined warm console, not a sepia or retro treatment.

### 5. Contrast and accessibility guardrails

The palette change cannot trade away clarity.

Required guardrails:

- primary text remains high-contrast on all dark backgrounds
- muted text stays visibly secondary but readable for long-form descriptions
- interactive focus states remain obvious on keyboard navigation
- primary button, active nav, and selected states remain distinguishable from nearby passive surfaces
- semantic `good`, `warn`, and `bad` treatments remain recognizable even after saturation is reduced to fit the warmer palette

If a warm token reduces readability or state recognition, contrast wins over palette purity.

## Testing Strategy

1. Run the existing `apps/skills-web` unit test suite to catch regressions in class-driven rendering and route theme expectations.
2. Run the frontend build to ensure no style edits accidentally break compilation or type checking.
3. Manually review representative routes after the style change:
   - portal home or catalog
   - a skill or version detail page
   - login or workbench home
   - a console workflow page such as import, authoring, governance, or audit
4. During review, confirm that no prominent cool-accent remnants remain in navigation, buttons, focus rings, or hero/background halos.

## Risks

1. Some components may contain hardcoded cool tint values rather than tokenized values, so a token-only pass may leave visual inconsistencies behind.
2. Route-specific shell variants may look too similar after unification unless opacity and surface contrast still provide enough contextual separation.
3. Reducing saturation too aggressively could make interactive states feel passive, especially in dense console screens.
