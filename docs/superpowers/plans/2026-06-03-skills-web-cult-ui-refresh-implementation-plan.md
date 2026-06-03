# Skills Web Cult UI Refresh Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refresh `apps/skills-web` into the approved `frontstage strong style + backstage restrained` design without changing backend contracts.

**Architecture:** Keep one shared product shell, split shared UI primitives into spotlight and console variants, and recompose each route around those variants. Add only light client-side dependencies for motion/icons plus a minimal test setup for shell-level route treatment.

**Tech Stack:** React 19, React Router 7, TypeScript, Vite, CSS, Vitest, Testing Library, Framer Motion, Lucide React

---

## File Structure

### Create

- `apps/skills-web/src/lib/routeTheme.ts` - route metadata for frontstage/backstage shell treatment
- `apps/skills-web/src/lib/routeTheme.test.ts` - tests for route theme classification
- `apps/skills-web/src/test/setup.ts` - DOM test setup
- `docs/superpowers/plans/2026-06-03-skills-web-cult-ui-refresh-implementation-plan.md` - this plan

### Modify

- `apps/skills-web/package.json` - add light UI/test dependencies
- `apps/skills-web/vite.config.ts` - enable Vitest
- `apps/skills-web/src/app.tsx` - centralize route declarations if needed for shell metadata
- `apps/skills-web/src/components/AppShell.tsx` - new shell, nav, environment rail, route intensity
- `apps/skills-web/src/components/SectionHeading.tsx` - richer heading variants
- `apps/skills-web/src/components/SurfaceCard.tsx` - spotlight/console variants
- `apps/skills-web/src/components/StatusPill.tsx` - unified pill styling hooks
- `apps/skills-web/src/components/TimelineList.tsx` - cleaner timeline rows
- `apps/skills-web/src/components/WarningStack.tsx` - console-friendly warnings
- `apps/skills-web/src/components/ImportReportCard.tsx` - structured import summary styling
- `apps/skills-web/src/pages/OverviewPage.tsx` - frontstage landing redesign
- `apps/skills-web/src/pages/CatalogPage.tsx` - frontstage discovery redesign
- `apps/skills-web/src/pages/SkillDetailPage.tsx` - bridge discovery-to-action redesign
- `apps/skills-web/src/pages/VersionDetailPage.tsx` - operational frontstage detail redesign
- `apps/skills-web/src/pages/ImportPage.tsx` - restrained workflow redesign
- `apps/skills-web/src/pages/GovernancePage.tsx` - restrained console redesign
- `apps/skills-web/src/pages/AuditPage.tsx` - restrained console redesign
- `apps/skills-web/src/pages/AuthoringFlowPage.tsx` - backstage alignment
- `apps/skills-web/src/styles.css` - token refresh and new layout system

## Task 1: Add Route Theme and Test Harness

**Files:**
- Create: `apps/skills-web/src/lib/routeTheme.ts`
- Create: `apps/skills-web/src/lib/routeTheme.test.ts`
- Create: `apps/skills-web/src/test/setup.ts`
- Modify: `apps/skills-web/package.json`
- Modify: `apps/skills-web/vite.config.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { describe, expect, it } from 'vitest';
import { getRouteTheme } from './routeTheme';

describe('getRouteTheme', () => {
  it('marks overview and catalog as frontstage routes', () => {
    expect(getRouteTheme('/').tone).toBe('frontstage');
    expect(getRouteTheme('/catalog').tone).toBe('frontstage');
  });

  it('marks import and audit as backstage routes', () => {
    expect(getRouteTheme('/import').tone).toBe('backstage');
    expect(getRouteTheme('/audit').tone).toBe('backstage');
  });

  it('treats skill detail routes as frontstage detail routes', () => {
    expect(getRouteTheme('/skills/%40platform.skill%2Fpdf-skill').tone).toBe('frontstage');
    expect(getRouteTheme('/skills/%40platform.skill%2Fpdf-skill/versions/1.2.0').accent).toBe('detail');
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npm run test -- src/lib/routeTheme.test.ts`
Expected: FAIL because `vitest` and `routeTheme.ts` do not exist yet.

- [ ] **Step 3: Add the minimal test setup and implementation**

```ts
// apps/skills-web/src/lib/routeTheme.ts
export type RouteTone = 'frontstage' | 'backstage';
export type RouteAccent = 'landing' | 'detail' | 'console';

export type RouteTheme = {
  tone: RouteTone;
  accent: RouteAccent;
};

const skillDetailPattern = /^\/skills\/[^/]+$/;
const versionDetailPattern = /^\/skills\/[^/]+\/versions\/[^/]+$/;

export function getRouteTheme(pathname: string): RouteTheme {
  if (pathname === '/' || pathname === '/catalog' || skillDetailPattern.test(pathname)) {
    return {
      tone: 'frontstage',
      accent: pathname === '/' ? 'landing' : 'detail',
    };
  }

  if (versionDetailPattern.test(pathname)) {
    return {
      tone: 'frontstage',
      accent: 'detail',
    };
  }

  return {
    tone: 'backstage',
    accent: 'console',
  };
}
```

```ts
// apps/skills-web/src/test/setup.ts
import '@testing-library/jest-dom/vitest';
```

```ts
// apps/skills-web/vite.config.ts
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
  },
});
```

- [ ] **Step 4: Add light dependencies**

```json
{
  "dependencies": {
    "framer-motion": "^12.0.0",
    "lucide-react": "^0.539.0"
  },
  "devDependencies": {
    "@testing-library/jest-dom": "^6.6.3",
    "@testing-library/react": "^16.1.0",
    "jsdom": "^26.1.0",
    "vitest": "^2.1.8"
  },
  "scripts": {
    "test": "vitest run"
  }
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `npm run test -- src/lib/routeTheme.test.ts`
Expected: PASS with `3 passed`.

## Task 2: Rebuild Shared Shell and Primitive Variants

**Files:**
- Modify: `apps/skills-web/src/components/AppShell.tsx`
- Modify: `apps/skills-web/src/components/SectionHeading.tsx`
- Modify: `apps/skills-web/src/components/SurfaceCard.tsx`
- Modify: `apps/skills-web/src/components/StatusPill.tsx`
- Modify: `apps/skills-web/src/components/TimelineList.tsx`
- Modify: `apps/skills-web/src/components/WarningStack.tsx`
- Modify: `apps/skills-web/src/components/ImportReportCard.tsx`
- Modify: `apps/skills-web/src/styles.css`

- [ ] **Step 1: Update shared primitives**

Implement:
- shell reads `useLocation()` and `getRouteTheme()`
- shell applies `frontstage` / `backstage` / `detail` classes
- `SurfaceCard` supports variants like `spotlight`, `console`, `rail`, `hero`
- `SectionHeading` supports layout classes for landing vs console pages
- `StatusPill`, `TimelineList`, `WarningStack`, `ImportReportCard` use the new shared token system

- [ ] **Step 2: Add shared styles**

Implement in `apps/skills-web/src/styles.css`:
- new root tokens for typography, surfaces, shadows, and route-specific intensity
- shell/nav/environment rail redesign
- spotlight card family
- console card family
- mobile-safe responsive rules

- [ ] **Step 3: Run targeted checks**

Run:
- `npm run test -- src/lib/routeTheme.test.ts`
- `npm run type-check`

Expected:
- tests still pass
- TypeScript remains clean

## Task 3: Redesign Frontstage Pages

**Files:**
- Modify: `apps/skills-web/src/pages/OverviewPage.tsx`
- Modify: `apps/skills-web/src/pages/CatalogPage.tsx`
- Modify: `apps/skills-web/src/pages/SkillDetailPage.tsx`
- Modify: `apps/skills-web/src/pages/VersionDetailPage.tsx`
- Modify: `apps/skills-web/src/styles.css`

- [ ] **Step 1: Recompose overview and catalog**

Implement:
- editorial overview hero with stronger CTA cluster
- operator rail separated from curated/discovery content
- catalog search band, stronger filters lane, and richer result cards

- [ ] **Step 2: Recompose skill and version detail**

Implement:
- stronger identity/install posture split
- action rails and next-step guidance
- better handoff/governance grouping on version detail

- [ ] **Step 3: Verify frontstage routes compile**

Run:
- `npm run type-check`
- `npm run build`

Expected:
- build succeeds
- no type regressions

## Task 4: Redesign Backstage Pages

**Files:**
- Modify: `apps/skills-web/src/pages/ImportPage.tsx`
- Modify: `apps/skills-web/src/pages/GovernancePage.tsx`
- Modify: `apps/skills-web/src/pages/AuditPage.tsx`
- Modify: `apps/skills-web/src/pages/AuthoringFlowPage.tsx`
- Modify: `apps/skills-web/src/styles.css`

- [ ] **Step 1: Recompose the console pages**

Implement:
- calmer page headers
- denser but clearer workflow/filter sidebars
- more stable review panes and timeline sections
- stronger stage/status grouping for import flow

- [ ] **Step 2: Tighten responsive behavior**

Implement:
- desktop split layouts collapse cleanly on mobile
- form controls and timeline rows remain readable at narrow widths

- [ ] **Step 3: Run full verification**

Run:
- `npm run test`
- `npm run type-check`
- `npm run build`

Expected:
- tests pass
- type-check passes
- production build passes

## Self-Review

- Spec coverage: covered shell split, shared token system, frontstage routes, backstage routes, light dependency boundary, and verification gates.
- Placeholder scan: no `TBD`, `TODO`, or undefined file references remain.
- Type consistency: `getRouteTheme()` defines the route intensity contract that `AppShell` consumes throughout the implementation.
