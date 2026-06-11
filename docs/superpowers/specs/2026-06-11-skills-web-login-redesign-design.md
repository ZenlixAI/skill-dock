# Skills Web Login Redesign

## Scope

This design covers the existing login experience under:

- `apps/skills-web/src/pages/LoginPage.tsx`
- `apps/skills-web/src/styles.css`
- related frontend tests that assert login rendering and behavior

Out of scope:

- login API behavior, session handling, or redirect rules
- global shell information architecture outside the login route
- introducing new frontend dependencies, fonts, or UI libraries
- adding role walkthroughs, environment banners, or troubleshooting content to the login page

## Goals

1. Replace the current plain single-card login page with a more intentional console entry experience.
2. Keep the page minimal and operational, with the form still acting as the clear primary action.
3. Align the login route more tightly with the existing warm dark console theme without creating a second visual language.
4. Improve input, button, loading, focus, and error states so the page feels finished rather than merely functional.
5. Preserve existing navigation behavior, session redirects, and testable semantics.

## Non-Goals

1. Rework authentication flows or add new login options.
2. Turn the login route into a marketing or onboarding page.
3. Add environment documentation, sample accounts, or role explanations to the page body.
4. Redesign unrelated portal or console screens as part of this task.

## Design

### 1. Page structure

The login page will move from a centered single-column stack to an asymmetric two-column layout on desktop, collapsing to one column on narrow screens.

The left column is a restrained context block rather than a feature panel. It will contain:

- eyebrow text indicating this is the console login entry
- a stronger page title that frames the route as the entrance to the skill registry console
- one short sentence explaining that import, authoring, governance, and audit workflows live behind this login

The right column remains the dominant region and contains the actual login surface. This keeps the form central while preventing the page from feeling like an unstyled fallback screen.

On mobile, the page will stack vertically with the context block first and the form second. No horizontal overflow or split-screen treatment is allowed.

### 2. Surface and layout treatment

The left column will not use a full card container. It should rely on spacing, typography, and one subtle structural accent such as a quiet divider line or compact stateless callout.

The right column will use a strengthened console surface derived from the existing theme tokens:

- slightly denser panel treatment than a generic content card
- clearer edge definition through border, inner highlight, and tinted shadow
- tighter spacing rhythm around the header and fields so the page reads as a focused control surface

The page container should feel intentionally offset rather than mathematically centered. The columns should not be equal width. The form column should carry more visual and spatial weight.

### 3. Typography and copy tone

The copy should stay concise and operational. The page must not drift into product marketing language.

Typography changes:

- the page title should carry more weight and tighter tracking than the current generic heading
- supporting text should stay narrow and readable, with restrained muted contrast
- labels should remain explicit and familiar: `用户名`, `密码`, `登录`

The left-side description should be one short sentence. The form panel may include a compact title and helper line, but it should not repeat the full same message as the left column.

### 4. Form interactions and states

The implementation must preserve the existing controlled form and submit flow, while improving the visible states.

Required behavior and presentation:

- inputs have a calm default state, a subtle hover lift, and a clear focus-visible ring
- the submit button has hover and pressed feedback using transform and shadow rather than abrupt color jumps
- the loading state keeps button width stable so the layout does not shift between `登录` and `登录中…`
- the error message remains inline and sits close to the submit action
- the error treatment should read like a system response banner rather than a plain red rectangle

Accessibility requirements:

- preserve proper label-to-input association
- expose the error region through `aria-live`
- keep keyboard focus visibility obvious on all interactive elements

### 5. Implementation boundaries

The redesign should stay inside the existing React and CSS architecture.

Implementation expectations:

- update `LoginPage.tsx` structure semantically rather than rebuilding route logic
- add login-specific layout and state classes in `styles.css`
- reuse existing tokens where possible and only add new tokens when the page cannot be expressed cleanly with the current set
- keep `SectionHeading` and `SurfaceCard` usage only if they still fit the redesigned structure; if they force an awkward generic layout, use route-local markup instead

The redesign must not change:

- the `next` query param behavior
- the force-logout landing behavior
- mock or token-auth redirects
- the actual login API call flow

## Testing Strategy

1. Run the `apps/skills-web` unit tests, especially login and shell coverage.
2. Run the frontend build to verify type checking and CSS usage remain valid.
3. Manually review the login route at desktop and narrow mobile widths.
4. During manual review, verify default, hover, focus, loading, and error states.
5. Confirm the page still redirects correctly after successful login and after logout-triggered return to `/login`.

## Risks

1. Reusing the generic page components too literally could keep the page looking like a content screen instead of an entry screen.
2. Adding too much visual treatment to the right panel could conflict with the minimal console-first direction.
3. Responsive imbalance could make the left context block feel redundant on smaller laptops if spacing is not tuned carefully.
