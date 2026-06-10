# Skills Web Warm Theme Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert `apps/skills-web` from the current mixed cool-accent theme to one unified warm dark theme without changing routes, component APIs, or page structure.

**Architecture:** Keep all existing React and routing behavior intact and concentrate the work in `apps/skills-web/src/styles.css`, with lightweight regression coverage in `AppShell` and route-theme tests so theme-related shell classes remain stable while the color system changes. The rollout is intentionally split into global tokens first and high-exposure components second so the visual diff stays reviewable.

**Tech Stack:** React 19, React Router 7, Vitest, Vite, CSS custom properties

---

## File Map

- `apps/skills-web/src/styles.css`
  - Owns the global color tokens, shell backgrounds, halos, panel surfaces, form controls, button styles, status treatments, and most component-level visual states.
- `apps/skills-web/src/components/AppShell.test.tsx`
  - Best place to lock down route shell class names and confirm the shell still renders the portal-first navigation, legal links, and stable route wrappers after theme work.
- `apps/skills-web/src/lib/routeTheme.test.ts`
  - Keeps the tone/accent mapping contract stable so palette unification does not accidentally mutate route semantics.
- `apps/skills-web/package.json`
  - No edits expected, but use its existing `test` and `build` scripts for verification.

### Task 1: Add Theme Regression Coverage Around Route Shells

**Files:**
- Modify: `apps/skills-web/src/components/AppShell.test.tsx`
- Modify: `apps/skills-web/src/lib/routeTheme.test.ts`

- [ ] **Step 1: Extend `AppShell` coverage with explicit shell class assertions**

Add a second test in `apps/skills-web/src/components/AppShell.test.tsx` that renders representative routes and asserts the shell keeps the expected tone/accent classes while preserving footer/nav chrome:

```tsx
it('applies stable route shell classes for portal and console routes', async () => {
  vi.mocked(client.getSession).mockResolvedValue({
    role: 'reader',
    actor: 'dev@company',
    authMechanism: 'session',
    allowedActions: ['session.read'],
    allowedPackages: [],
  });

  const portal = render(
    <SessionProvider>
      <MemoryRouter initialEntries={['/catalog']}>
        <AppShell>
          <div>content</div>
        </AppShell>
      </MemoryRouter>
    </SessionProvider>
  );

  expect(portal.container.firstElementChild).toHaveClass('app-shell', 'app-shell--frontstage', 'app-shell--detail');
  portal.unmount();

  const consoleShell = render(
    <SessionProvider>
      <MemoryRouter initialEntries={['/login']}>
        <AppShell>
          <div>content</div>
        </AppShell>
      </MemoryRouter>
    </SessionProvider>
  );

  expect(consoleShell.container.firstElementChild).toHaveClass('app-shell', 'app-shell--backstage', 'app-shell--console');
  expect(screen.getByRole('link', { name: '隐私' })).toHaveAttribute('href', '/privacy');
});
```

- [ ] **Step 2: Tighten `routeTheme` expectations for the routes used in visual review**

Expand `apps/skills-web/src/lib/routeTheme.test.ts` so the routes that will be manually reviewed are locked to their existing semantic buckets:

```ts
it('keeps legal and login routes on their current semantic accents', () => {
  expect(getRouteTheme('/privacy')).toEqual({ tone: 'frontstage', accent: 'legal' });
  expect(getRouteTheme('/terms')).toEqual({ tone: 'frontstage', accent: 'legal' });
  expect(getRouteTheme('/login')).toEqual({ tone: 'backstage', accent: 'console' });
  expect(getRouteTheme('/workbench/import')).toEqual({ tone: 'backstage', accent: 'console' });
});
```

- [ ] **Step 3: Run the targeted regression tests**

Run:

```bash
cd apps/skills-web
npm test -- src/components/AppShell.test.tsx src/lib/routeTheme.test.ts
```

Expected: `PASS` for both test files.

- [ ] **Step 4: Commit the regression coverage**

Run:

```bash
git add apps/skills-web/src/components/AppShell.test.tsx apps/skills-web/src/lib/routeTheme.test.ts
git commit -m "test: cover skills web shell theme routes"
```

### Task 2: Replace The Global Cool Palette With Warm Dark Tokens

**Files:**
- Modify: `apps/skills-web/src/styles.css`

- [ ] **Step 1: Replace the root color tokens with one warm palette**

Update the `:root` block near the top of `apps/skills-web/src/styles.css` to remove the cool cyan/moss emphasis and replace it with a single warm dark palette:

```css
:root {
  --bg: #17110d;
  --bg-deep: #100b08;
  --panel: rgba(31, 22, 18, 0.84);
  --panel-strong: rgba(37, 27, 22, 0.96);
  --panel-muted: rgba(27, 20, 16, 0.72);
  --panel-console: rgba(35, 26, 21, 0.9);
  --border: rgba(214, 181, 145, 0.16);
  --border-strong: rgba(214, 181, 145, 0.28);
  --text: #f5ede5;
  --muted: #c8b4a3;
  --accent: #cf8a4a;
  --accent-soft: rgba(207, 138, 74, 0.16);
  --good: #88b294;
  --warn: #d8a35f;
  --bad: #d5826b;
  --shadow-soft: 0 22px 60px rgba(14, 8, 4, 0.3);
  --shadow-strong: 0 36px 100px rgba(14, 8, 4, 0.42);
  --radius-xl: 30px;
  --radius-lg: 22px;
  --radius-md: 18px;
  --radius-sm: 14px;
}
```

- [ ] **Step 2: Rebuild the page background and shell halos around the warm palette**

Change the `body`, shell background, and halo rules so the page atmosphere stays layered but loses the blue-tech cast:

```css
body {
  margin: 0;
  color: var(--text);
  background:
    radial-gradient(circle at top left, rgba(207, 138, 74, 0.18), transparent 30%),
    radial-gradient(circle at top right, rgba(122, 77, 44, 0.14), transparent 34%),
    linear-gradient(180deg, #17110d 0%, #100b08 100%);
  font-family: "IBM Plex Sans", "Segoe UI Variable", "Source Sans 3", sans-serif;
}

.app-shell--frontstage {
  background: linear-gradient(180deg, rgba(33, 24, 19, 0.78), transparent 28%);
}

.app-shell--backstage {
  background: linear-gradient(180deg, rgba(25, 18, 14, 0.84), transparent 24%);
}

.app-shell--legal {
  background: linear-gradient(180deg, rgba(42, 28, 22, 0.76), transparent 24%);
}

.app-shell--neutral {
  background: linear-gradient(180deg, rgba(29, 21, 17, 0.76), transparent 24%);
}

.app-shell__halo--ember {
  background: rgba(207, 138, 74, 0.2);
}

.app-shell__halo--moss {
  background: rgba(128, 82, 48, 0.15);
}
```

- [ ] **Step 3: Warm up the shell chrome and quiet neutrals**

Update the shell framing and neutral utility surfaces so headers, footers, nav rails, labels, and empty/error scaffolding all read from the same family:

```css
.shell-nav {
  border: 1px solid var(--border);
  background: rgba(23, 17, 13, 0.74);
  box-shadow: var(--shadow-soft);
}

.shell-footer__inner {
  border: 1px solid var(--border);
  background: rgba(23, 17, 13, 0.68);
}

.env-banner span,
.eyebrow,
.hero-kicker,
.signal-row strong {
  background: rgba(52, 34, 24, 0.26);
  border-color: rgba(214, 181, 145, 0.12);
}

.empty-box {
  border-color: rgba(214, 181, 145, 0.2);
  background: rgba(29, 21, 17, 0.68);
}

.error-box {
  border-color: rgba(213, 130, 107, 0.28);
  background: rgba(65, 28, 24, 0.86);
}
```

- [ ] **Step 4: Run the focused tests and a production build**

Run:

```bash
cd apps/skills-web
npm test -- src/components/AppShell.test.tsx src/lib/routeTheme.test.ts
npm run build
```

Expected:

- tests: `PASS`
- build: Vite completes successfully and writes the production bundle

- [ ] **Step 5: Commit the global warm token pass**

Run:

```bash
git add apps/skills-web/src/styles.css
git commit -m "style: warm skills web global theme tokens"
```

### Task 3: Recolor High-Exposure Controls And Surface Variants

**Files:**
- Modify: `apps/skills-web/src/styles.css`

- [ ] **Step 1: Recolor navigation, buttons, and focus states to use the new accent**

Update the high-traffic interaction selectors so active, hover, and keyboard focus states all derive from the warm accent rather than lingering cool tints:

```css
.nav-chip {
  border: 1px solid rgba(214, 181, 145, 0.1);
  color: var(--muted);
  background: rgba(37, 27, 22, 0.56);
}

.nav-chip:hover,
.nav-chip--active {
  color: var(--text);
  background: rgba(55, 38, 28, 0.92);
  border-color: var(--border-strong);
  box-shadow: 0 10px 24px rgba(24, 12, 6, 0.24);
}

.field input:focus,
.field select:focus,
.field textarea:focus {
  border-color: rgba(207, 138, 74, 0.42);
  box-shadow: 0 0 0 4px rgba(207, 138, 74, 0.12);
}

.button--primary {
  background: linear-gradient(135deg, #cf8a4a, #e0a76d);
  color: #1b120c;
  box-shadow: 0 10px 22px rgba(24, 12, 6, 0.3);
}

.button--ghost {
  background: rgba(37, 27, 22, 0.76);
  border-color: rgba(214, 181, 145, 0.14);
}
```

- [ ] **Step 2: Rebuild card and panel variants so portal and console surfaces share one warm language**

Update the major surface selectors that still carry cool gradients or teal/cyan atmospheric washes:

```css
.skill-summary-card {
  background:
    linear-gradient(180deg, rgba(33, 24, 19, 0.96), rgba(29, 21, 17, 0.88)),
    radial-gradient(circle at top left, rgba(207, 138, 74, 0.12), transparent 36%);
  border: 1px solid var(--border);
  box-shadow: var(--shadow-soft);
}

.console-entry-card {
  background:
    linear-gradient(180deg, rgba(27, 20, 16, 0.96), rgba(23, 17, 13, 0.84)),
    radial-gradient(circle at top right, rgba(150, 95, 56, 0.12), transparent 28%);
}

.surface-card--subtle {
  background:
    linear-gradient(180deg, rgba(29, 21, 17, 0.96), rgba(37, 27, 22, 0.82)),
    radial-gradient(circle at top left, rgba(207, 138, 74, 0.1), transparent 26%);
}

.surface-card--hero {
  background:
    linear-gradient(145deg, rgba(255, 247, 239, 0.98), rgba(255, 252, 247, 0.82)),
    radial-gradient(circle at top left, rgba(207, 138, 74, 0.18), transparent 34%),
    radial-gradient(circle at bottom right, rgba(168, 111, 70, 0.12), transparent 30%);
}

.surface-card--rail,
.surface-card--cool,
.surface-card--console {
  background:
    linear-gradient(180deg, rgba(250, 245, 238, 0.96), rgba(255, 251, 246, 0.82)),
    radial-gradient(circle at top right, rgba(168, 111, 70, 0.1), transparent 26%);
}
```

- [ ] **Step 3: Normalize semantic pills, rows, and workflow highlights to fit the new palette**

Finish the recolor pass on supporting UI states so workflow and data-heavy screens still feel cohesive:

```css
.status-pill--good {
  background: rgba(136, 178, 148, 0.14);
  color: var(--good);
}

.status-pill--warn {
  background: rgba(216, 163, 95, 0.14);
  color: var(--warn);
}

.status-pill--bad {
  background: rgba(213, 130, 107, 0.14);
  color: var(--bad);
}

.status-pill--accent {
  background: var(--accent-soft);
  color: var(--accent);
}

.step-card--active,
.flow-node--active,
.list-card:hover {
  border-color: rgba(207, 138, 74, 0.24);
}

.timeline-item__bar {
  background: linear-gradient(180deg, rgba(207, 138, 74, 0.82), rgba(120, 75, 45, 0.42));
}
```

- [ ] **Step 4: Search for old cool-theme literals and remove any prominent remnants**

Run:

```bash
rg -n "6fd3ff|49bba0|111, 211, 255|73, 187, 160|150, 178, 215" apps/skills-web/src/styles.css
```

Expected: no matches in `apps/skills-web/src/styles.css`.

- [ ] **Step 5: Re-run focused tests and build**

Run:

```bash
cd apps/skills-web
npm test -- src/components/AppShell.test.tsx src/lib/routeTheme.test.ts
npm run build
```

Expected:

- tests: `PASS`
- build: Vite completes successfully

- [ ] **Step 6: Commit the surface and interaction recolor**

Run:

```bash
git add apps/skills-web/src/styles.css
git commit -m "style: unify skills web warm accents"
```

### Task 4: Full Verification And Visual Review

**Files:**
- Verify: `apps/skills-web/src/styles.css`
- Verify: `apps/skills-web/src/components/AppShell.test.tsx`
- Verify: `apps/skills-web/src/lib/routeTheme.test.ts`

- [ ] **Step 1: Run the full frontend test suite**

Run:

```bash
cd apps/skills-web
npm test
```

Expected: all Vitest suites pass.

- [ ] **Step 2: Run the production build one more time**

Run:

```bash
cd apps/skills-web
npm run build
```

Expected: production build succeeds without type-check or bundling errors.

- [ ] **Step 3: Run a local visual review server**

Run:

```bash
cd apps/skills-web
npm run dev -- --host 127.0.0.1 --port 4173
```

Review these routes in the browser:

- `http://127.0.0.1:4173/`
- `http://127.0.0.1:4173/catalog`
- `http://127.0.0.1:4173/login`
- `http://127.0.0.1:4173/workbench`
- `http://127.0.0.1:4173/workbench/import`
- `http://127.0.0.1:4173/skills/%40platform.skill%2Fpdf-skill`

Confirm:

- no obvious cyan/teal accent remnants remain in halo glows, active nav, focus rings, or CTA buttons
- portal and console screens still feel distinct by density and opacity, not by color family
- primary text, muted text, form controls, and semantic pills remain readable

- [ ] **Step 4: Review the diff for scope control**

Run:

```bash
git diff --stat HEAD~3..HEAD
git diff -- apps/skills-web/src/styles.css apps/skills-web/src/components/AppShell.test.tsx apps/skills-web/src/lib/routeTheme.test.ts
```

Expected: only the planned theme/test files changed; no route logic, component API, or content scope drift.
