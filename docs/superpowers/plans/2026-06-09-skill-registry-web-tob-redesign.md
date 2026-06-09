# Skill Registry Web ToB Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reshape `apps/skills-web` into a developer-first `Portal` plus a secondary `Console`, with install-first detail pages, stronger product completeness, and a real ToB registry visual system.

**Architecture:** Keep the existing React Router routes, API contracts, and session model, but add a clearer product shell: route metadata, Portal/Console navigation groups, legal and not-found pages, and a shared install command panel. `Overview` and `Catalog` become discovery-first pages, `SkillDetail` and `VersionDetail` share install-oriented action surfaces, and the backend pages stay in `/workbench/*` while presenting themselves as a secondary operations console. Because the current Catalog contracts do not expose explicit owner/team fields, the redesign uses package namespace plus `updatedAt` as the tertiary provenance row until the backend adds first-class ownership metadata.

**Tech Stack:** React 19, TypeScript, React Router 7, Vitest, Testing Library, existing `skills-web` CSS

---

## File Map

- Create: `apps/skills-web/src/lib/navigation.ts`
  - Portal and Console nav item definitions, grouped by section
- Create: `apps/skills-web/src/lib/routeMeta.ts`
  - Route title/description/section metadata used by the shell and document head
- Create: `apps/skills-web/src/components/DocumentMeta.tsx`
  - Lightweight `<title>` and `<meta name="description">` synchronizer
- Create: `apps/skills-web/src/pages/NotFoundPage.tsx`
  - Branded 404 page with back paths into Portal
- Create: `apps/skills-web/src/pages/PrivacyPage.tsx`
  - Internal privacy notice page linked from the footer
- Create: `apps/skills-web/src/pages/TermsPage.tsx`
  - Internal terms-of-use page linked from the footer
- Create: `apps/skills-web/src/components/AppShell.test.tsx`
  - Shell coverage for skip link, Portal/Console grouping, and footer links
- Create: `apps/skills-web/src/pages/NotFoundPage.test.tsx`
  - Route-level 404 coverage

- Create: `apps/skills-web/src/components/SkillSummaryCard.tsx`
  - Shared discovery card used by Overview and Catalog
- Create: `apps/skills-web/src/pages/OverviewPage.test.tsx`
  - Overview coverage for the search hero and curated/recent sections
- Create: `apps/skills-web/src/pages/CatalogPage.test.tsx`
  - Catalog coverage for query hydration, discovery card content order, and empty guidance

- Create: `apps/skills-web/src/lib/installCommand.ts`
  - Shared install command builder
- Create: `apps/skills-web/src/lib/installCommand.test.ts`
  - Unit tests for install command generation
- Create: `apps/skills-web/src/components/InstallCommandPanel.tsx`
  - Shared install/copy surface for detail pages
- Create: `apps/skills-web/src/components/InstallCommandPanel.test.tsx`
  - Coverage for copy behavior and disabled/error messaging

- Create: `apps/skills-web/src/pages/WorkbenchHomePage.test.tsx`
  - Coverage for Console home entry grouping
- Create: `apps/skills-web/src/pages/AuditPage.test.tsx`
  - Coverage for audit summary-first layout and empty state guidance

- Modify: `apps/skills-web/src/app.tsx`
  - Add legal pages and a real `*` route for not found
- Modify: `apps/skills-web/src/components/AppShell.tsx`
  - Build the Portal/Console shell, skip link, footer, and metadata hooks
- Modify: `apps/skills-web/src/components/SectionHeading.tsx`
  - Support the new Portal hero and Console heading layouts without forcing the old rhythm
- Modify: `apps/skills-web/src/components/SurfaceCard.tsx`
  - Add the new discovery/install surface variants without rewriting call sites
- Modify: `apps/skills-web/src/lib/routeTheme.ts`
  - Distinguish Portal, Console, auth, legal, and not-found tones
- Modify: `apps/skills-web/src/pages/OverviewPage.tsx`
  - Replace the dashboard-like layout with a search-first developer portal
- Modify: `apps/skills-web/src/pages/CatalogPage.tsx`
  - Support query-string hydration, a results-first layout, and better empty/error guidance
- Modify: `apps/skills-web/src/pages/SkillDetailPage.tsx`
  - Move install actions into the primary above-the-fold surface
- Modify: `apps/skills-web/src/pages/VersionDetailPage.tsx`
  - Move version install/release data into a primary action surface
- Modify: `apps/skills-web/src/pages/WorkbenchHomePage.tsx`
  - Reframe `/workbench` as `Console Home`
- Modify: `apps/skills-web/src/pages/GovernancePage.tsx`
  - Make the page read like an operations console rather than a raw timeline
- Modify: `apps/skills-web/src/pages/AuditPage.tsx`
  - Make the page answer “what happened” before rendering raw rows
- Modify: `apps/skills-web/src/styles.css`
  - Replace warm workbench tokens with the new dark Portal/Console system and page layouts
- Modify: `apps/skills-web/index.html`
  - Add durable base meta tags for the product shell

## Task 1: Build the Portal/Console Shell and Product Completeness Foundation

**Files:**
- Create: `apps/skills-web/src/lib/navigation.ts`
- Create: `apps/skills-web/src/lib/routeMeta.ts`
- Create: `apps/skills-web/src/components/DocumentMeta.tsx`
- Create: `apps/skills-web/src/pages/NotFoundPage.tsx`
- Create: `apps/skills-web/src/pages/PrivacyPage.tsx`
- Create: `apps/skills-web/src/pages/TermsPage.tsx`
- Create: `apps/skills-web/src/components/AppShell.test.tsx`
- Create: `apps/skills-web/src/pages/NotFoundPage.test.tsx`
- Modify: `apps/skills-web/src/app.tsx`
- Modify: `apps/skills-web/src/components/AppShell.tsx`
- Modify: `apps/skills-web/src/components/SectionHeading.tsx`
- Modify: `apps/skills-web/src/components/SurfaceCard.tsx`
- Modify: `apps/skills-web/src/lib/routeTheme.ts`
- Modify: `apps/skills-web/src/styles.css`
- Modify: `apps/skills-web/index.html`
- Test: `apps/skills-web/src/components/AppShell.test.tsx`
- Test: `apps/skills-web/src/pages/NotFoundPage.test.tsx`

- [ ] **Step 1: Write the failing shell and not-found tests**

Create `apps/skills-web/src/components/AppShell.test.tsx` with:

```tsx
import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import * as client from '../api/client';
import { SessionProvider } from '../session/SessionProvider';
import { AppShell } from './AppShell';

vi.mock('../api/client', () => ({
  getRegistryClientConfig: vi.fn(() => ({
    mode: 'real',
    baseUrl: null,
    role: null,
    actor: null,
    hasToken: false,
  })),
  getSession: vi.fn(),
  logoutSession: vi.fn(),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('AppShell', () => {
  it('renders a skip link, portal-first nav, console entry, and legal footer links', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'reader',
      actor: 'dev@company',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });

    render(
      <SessionProvider>
        <MemoryRouter initialEntries={['/catalog']}>
          <AppShell>
            <div>content</div>
          </AppShell>
        </MemoryRouter>
      </SessionProvider>
    );

    expect(await screen.findByRole('link', { name: 'Skip to content' })).toHaveAttribute('href', '#main-content');
    expect(screen.getByRole('link', { name: 'Overview' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Catalog' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Console' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Privacy' })).toHaveAttribute('href', '/privacy');
    expect(screen.getByRole('link', { name: 'Terms' })).toHaveAttribute('href', '/terms');
  });
});
```

Create `apps/skills-web/src/pages/NotFoundPage.test.tsx` with:

```tsx
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { NotFoundPage } from './NotFoundPage';

describe('NotFoundPage', () => {
  it('offers a clear way back into the portal', () => {
    render(
      <MemoryRouter>
        <NotFoundPage />
      </MemoryRouter>
    );

    expect(screen.getByRole('heading', { name: 'Page not found' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Back to overview' })).toHaveAttribute('href', '/');
    expect(screen.getByRole('link', { name: 'Browse catalog' })).toHaveAttribute('href', '/catalog');
  });
});
```

- [ ] **Step 2: Run the new tests to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/components/AppShell.test.tsx src/pages/NotFoundPage.test.tsx
```

Expected: FAIL because the shell does not expose the new Portal/Console navigation labels, skip link, or footer links, and `NotFoundPage.tsx` does not exist yet.

- [ ] **Step 3: Implement the shell, route metadata, legal pages, and 404 route**

Create `apps/skills-web/src/lib/navigation.ts` with:

```ts
import { AppWindow, Blocks, FileClock, Search, ShieldAlert, Sparkles, Workflow } from 'lucide-react';

export const portalNavItems = [
  { to: '/', label: 'Overview', icon: Sparkles },
  { to: '/catalog', label: 'Catalog', icon: Search },
] as const;

export const consoleNavItems = [
  { to: '/workbench', label: 'Console', icon: AppWindow },
  { to: '/workbench/import', label: 'Import', icon: Workflow, requiredAction: 'packaging.prepare' },
  { to: '/workbench/authoring', label: 'Authoring', icon: Blocks, requiredAction: 'packaging.prepare' },
  { to: '/workbench/governance', label: 'Governance', icon: ShieldAlert, requiredAction: 'catalog.governance.skill.apply' },
  { to: '/workbench/audit', label: 'Audit', icon: FileClock },
] as const;
```

Create `apps/skills-web/src/lib/routeMeta.ts` with:

```ts
export type RouteSection = 'portal' | 'console' | 'auth' | 'legal' | 'not-found';

export interface RouteMeta {
  title: string;
  description: string;
  section: RouteSection;
}

export function getRouteMeta(pathname: string): RouteMeta {
  if (pathname === '/') {
    return {
      title: 'Skill Registry Portal',
      description: 'Search, evaluate, and install skills from the company registry.',
      section: 'portal',
    };
  }
  if (pathname === '/catalog') {
    return {
      title: 'Catalog · Skill Registry',
      description: 'Browse curated and searchable skills with install-ready metadata.',
      section: 'portal',
    };
  }
  if (pathname.startsWith('/workbench')) {
    return {
      title: 'Console · Skill Registry',
      description: 'Operate import, authoring, governance, and audit workflows.',
      section: 'console',
    };
  }
  if (pathname === '/login') {
    return {
      title: 'Login · Skill Registry',
      description: 'Sign in to the internal skill registry.',
      section: 'auth',
    };
  }
  if (pathname === '/privacy') {
    return {
      title: 'Privacy · Skill Registry',
      description: 'How the internal registry handles operator and catalog activity data.',
      section: 'legal',
    };
  }
  if (pathname === '/terms') {
    return {
      title: 'Terms · Skill Registry',
      description: 'Rules for using the internal skill registry and Console workflows.',
      section: 'legal',
    };
  }
  return {
    title: 'Page not found · Skill Registry',
    description: 'The requested registry page does not exist.',
    section: 'not-found',
  };
}
```

Create `apps/skills-web/src/components/DocumentMeta.tsx` with:

```tsx
import { useEffect } from 'react';

interface DocumentMetaProps {
  title: string;
  description: string;
}

export function DocumentMeta({ title, description }: DocumentMetaProps) {
  useEffect(() => {
    document.title = title;
    const tag = document.querySelector('meta[name="description"]');
    if (tag) {
      tag.setAttribute('content', description);
    }
  }, [description, title]);

  return null;
}
```

Create `apps/skills-web/src/pages/NotFoundPage.tsx` with:

```tsx
import { Link } from 'react-router-dom';
import { SectionHeading } from '../components/SectionHeading';
import { SurfaceCard } from '../components/SurfaceCard';

export function NotFoundPage() {
  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="404"
        title="Page not found"
        description="The page you opened is not part of the registry portal or Console."
        tone="detail"
      />
      <SurfaceCard title="Try one of these paths" variant="hero">
        <div className="action-row">
          <Link className="button button--primary" to="/">
            Back to overview
          </Link>
          <Link className="button button--secondary" to="/catalog">
            Browse catalog
          </Link>
        </div>
      </SurfaceCard>
    </div>
  );
}
```

Create `apps/skills-web/src/pages/PrivacyPage.tsx` and `apps/skills-web/src/pages/TermsPage.tsx` with:

```tsx
import { SectionHeading } from '../components/SectionHeading';
import { SurfaceCard } from '../components/SurfaceCard';

export function PrivacyPage() {
  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="Privacy"
        title="Privacy notice"
        description="This registry stores session actor, package, version, and workflow audit data for operational tracing."
        tone="detail"
      />
      <SurfaceCard title="What we store">
        <p>Session identity, allowed actions, package and version targets, and workflow results are recorded for traceability.</p>
      </SurfaceCard>
    </div>
  );
}
```

```tsx
import { SectionHeading } from '../components/SectionHeading';
import { SurfaceCard } from '../components/SurfaceCard';

export function TermsPage() {
  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="Terms"
        title="Terms of use"
        description="Use the registry for approved internal package discovery, release, and governance workflows."
        tone="detail"
      />
      <SurfaceCard title="Allowed use">
        <p>Only publish, retire, deprecate, or yank packages that fall within your granted package scope and operational responsibility.</p>
      </SurfaceCard>
    </div>
  );
}
```

Update `apps/skills-web/src/components/AppShell.tsx` so the shell uses the new metadata and nav groups:

```tsx
import { NavLink, useLocation, useNavigate } from 'react-router-dom';
import { DocumentMeta } from './DocumentMeta';
import { consoleNavItems, portalNavItems } from '../lib/navigation';
import { getRouteMeta } from '../lib/routeMeta';

const meta = getRouteMeta(location.pathname);
const visibleConsoleNav = consoleNavItems.filter((item) =>
  item.requiredAction ? hasAction(session.data, item.requiredAction) : true
);
```

and in the JSX:

```tsx
<DocumentMeta description={meta.description} title={meta.title} />
<a className="skip-link" href="#main-content">
  Skip to content
</a>
<header className="shell-header">
  <div className="brand-lockup__copy">
    <div className="brand-lockup__eyebrow">{meta.section === 'console' ? 'Console' : 'Registry Portal'}</div>
    <h1>Skill Registry</h1>
    <p>Search, evaluate, install, and operate internal skills.</p>
  </div>
  ...
</header>
<div className="shell-nav">
  <nav className="shell-nav__group" aria-label="Portal">
    {portalNavItems.map(...)}
  </nav>
  <nav className="shell-nav__group shell-nav__group--console" aria-label="Console">
    {visibleConsoleNav.map(...)}
  </nav>
</div>
<main id="main-content" className="page-shell">
  {children}
</main>
<footer className="shell-footer">
  <span>Internal registry for discoverable, governed skill packages.</span>
  <div className="shell-footer__links">
    <NavLink to="/privacy">Privacy</NavLink>
    <NavLink to="/terms">Terms</NavLink>
  </div>
</footer>
```

Update `apps/skills-web/src/app.tsx` so the legal pages and 404 route are real:

```tsx
import { NotFoundPage } from './pages/NotFoundPage';
import { PrivacyPage } from './pages/PrivacyPage';
import { TermsPage } from './pages/TermsPage';

<Route path="/privacy" element={<PrivacyPage />} />
<Route path="/terms" element={<TermsPage />} />
...
<Route path="*" element={<NotFoundPage />} />
```

Update `apps/skills-web/index.html` with:

```html
<meta name="description" content="Internal skill registry for discovery, install, release, and governance workflows." />
<meta property="og:title" content="Skill Registry" />
<meta property="og:description" content="Internal skill registry for discovery, install, release, and governance workflows." />
```

Update `apps/skills-web/src/styles.css` to establish the new shell foundation:

```css
:root {
  --bg: #08111b;
  --bg-elevated: #0d1724;
  --panel: rgba(13, 23, 36, 0.88);
  --panel-strong: rgba(17, 28, 42, 0.96);
  --panel-muted: rgba(12, 21, 34, 0.72);
  --border: rgba(163, 184, 208, 0.14);
  --border-strong: rgba(163, 184, 208, 0.24);
  --text: #edf4ff;
  --muted: #9db0c9;
  --accent: #5b8cff;
  --accent-soft: rgba(91, 140, 255, 0.16);
  --good: #45b887;
  --warn: #d3a347;
  --bad: #d06b63;
  --shadow-soft: 0 20px 60px rgba(2, 8, 16, 0.36);
  --shadow-strong: 0 28px 90px rgba(2, 8, 16, 0.52);
}

.skip-link {
  position: absolute;
  top: 16px;
  left: 16px;
  transform: translateY(-150%);
}

.skip-link:focus {
  transform: translateY(0);
}

.shell-nav__group {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.shell-footer {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  max-width: 1480px;
  margin: 0 auto;
  padding: 0 24px 40px;
  color: var(--muted);
}
```

- [ ] **Step 4: Run the shell tests and the type-safe build**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/components/AppShell.test.tsx src/pages/NotFoundPage.test.tsx && npm run build
```

Expected: PASS. The shell tests should pass and the build should emit the Vite production bundle without route/type errors.

- [ ] **Step 5: Commit the shell foundation**

```bash
cd /Users/ultraman/workspace/skill_registry
git add apps/skills-web/src/lib/navigation.ts apps/skills-web/src/lib/routeMeta.ts apps/skills-web/src/components/DocumentMeta.tsx apps/skills-web/src/pages/NotFoundPage.tsx apps/skills-web/src/pages/PrivacyPage.tsx apps/skills-web/src/pages/TermsPage.tsx apps/skills-web/src/components/AppShell.test.tsx apps/skills-web/src/pages/NotFoundPage.test.tsx apps/skills-web/src/app.tsx apps/skills-web/src/components/AppShell.tsx apps/skills-web/src/components/SectionHeading.tsx apps/skills-web/src/components/SurfaceCard.tsx apps/skills-web/src/lib/routeTheme.ts apps/skills-web/src/styles.css apps/skills-web/index.html
git commit -m "feat: add portal console shell foundation"
```

## Task 2: Redesign Overview and Catalog Around Discovery

**Files:**
- Create: `apps/skills-web/src/components/SkillSummaryCard.tsx`
- Create: `apps/skills-web/src/pages/OverviewPage.test.tsx`
- Create: `apps/skills-web/src/pages/CatalogPage.test.tsx`
- Modify: `apps/skills-web/src/pages/OverviewPage.tsx`
- Modify: `apps/skills-web/src/pages/CatalogPage.tsx`
- Modify: `apps/skills-web/src/styles.css`
- Test: `apps/skills-web/src/pages/OverviewPage.test.tsx`
- Test: `apps/skills-web/src/pages/CatalogPage.test.tsx`

- [ ] **Step 1: Write the failing discovery-page tests**

Create `apps/skills-web/src/pages/OverviewPage.test.tsx` with:

```tsx
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import * as client from '../api/client';
import { OverviewPage } from './OverviewPage';

vi.mock('../api/client', () => ({
  listAuditEvents: vi.fn(),
  listCuratedSkills: vi.fn(),
  listGovernanceChanges: vi.fn(),
  listSkills: vi.fn(),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function CatalogTarget() {
  const location = useLocation();
  return <div>{`catalog target${location.search}`}</div>;
}

describe('OverviewPage', () => {
  it('submits the hero search into the catalog query string', async () => {
    vi.mocked(client.listSkills).mockResolvedValue({ items: [], pageInfo: { hasNextPage: false, nextCursor: null } });
    vi.mocked(client.listCuratedSkills).mockResolvedValue([]);
    vi.mocked(client.listGovernanceChanges).mockResolvedValue({ items: [], pageInfo: { hasNextPage: false, nextCursor: null } });
    vi.mocked(client.listAuditEvents).mockResolvedValue({ items: [], pageInfo: { hasNextPage: false, nextCursor: null } });

    render(
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route path="/" element={<OverviewPage />} />
          <Route path="/catalog" element={<CatalogTarget />} />
        </Routes>
      </MemoryRouter>
    );

    fireEvent.change(await screen.findByLabelText('Search skills'), { target: { value: 'review' } });
    fireEvent.click(screen.getByRole('button', { name: 'Search catalog' }));

    expect(await screen.findByText('catalog target?q=review')).toBeInTheDocument();
  });
});
```

Create `apps/skills-web/src/pages/CatalogPage.test.tsx` with:

```tsx
import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import * as client from '../api/client';
import { CatalogPage } from './CatalogPage';

vi.mock('../api/client', () => ({
  listSkills: vi.fn(),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('CatalogPage', () => {
  it('hydrates the search query from the URL and shows discovery-first empty guidance', async () => {
    vi.mocked(client.listSkills).mockResolvedValue({
      items: [],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });

    render(
      <MemoryRouter initialEntries={['/catalog?q=pdf']}>
        <CatalogPage />
      </MemoryRouter>
    );

    expect(await screen.findByDisplayValue('pdf')).toBeInTheDocument();
    expect(screen.getByText('Try a broader capability, package scope, or source filter.')).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run the discovery-page tests to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/pages/OverviewPage.test.tsx src/pages/CatalogPage.test.tsx
```

Expected: FAIL because `OverviewPage` has no searchable hero form and `CatalogPage` does not hydrate `q` from the URL or show the new empty guidance.

- [ ] **Step 3: Implement the shared discovery card and the new Overview/Catalog layouts**

Create `apps/skills-web/src/components/SkillSummaryCard.tsx` with:

```tsx
import { Link } from 'react-router-dom';
import type { CuratedSkillSummary, SkillSummary } from '../api/types';
import { formatDate } from '../lib/format';
import { StatusPill } from './StatusPill';

type SkillSummaryCardProps = {
  skill: SkillSummary | CuratedSkillSummary;
  curationReason?: string;
};

export function SkillSummaryCard({ skill, curationReason }: SkillSummaryCardProps) {
  const namespace = skill.packageName.split('/')[0] || skill.packageName;

  return (
    <article className="skill-summary-card">
      <div className="skill-summary-card__header">
        <div>
          <strong>{skill.displayName}</strong>
          <span>{skill.packageName}</span>
        </div>
        <StatusPill tone={skill.latestVersionStatus === 'published' ? 'good' : skill.latestVersionStatus === 'deprecated' ? 'warn' : 'bad'}>
          {skill.latestVersionStatus || 'unknown'}
        </StatusPill>
      </div>
      <p className="skill-summary-card__description">{skill.description}</p>
      <div className="skill-summary-card__meta">
        <StatusPill tone="accent">{skill.sourceType}</StatusPill>
        {skill.latestVersion ? <StatusPill>latest {skill.latestVersion}</StatusPill> : null}
        <span className="skill-summary-card__provenance">{namespace} · updated {formatDate(skill.updatedAt)}</span>
      </div>
      {curationReason ? <p className="skill-summary-card__reason">{curationReason}</p> : null}
      <div className="skill-summary-card__actions">
        <Link className="button button--primary" to={`/skills/${encodeURIComponent(skill.packageName)}`}>
          Open skill
        </Link>
      </div>
    </article>
  );
}
```

Update `apps/skills-web/src/pages/OverviewPage.tsx` so the hero submits to Catalog:

```tsx
import { startTransition, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { SkillSummaryCard } from '../components/SkillSummaryCard';

const [query, setQuery] = useState('');
const navigate = useNavigate();

function handleSearchSubmit(event: React.FormEvent<HTMLFormElement>) {
  event.preventDefault();
  const next = query.trim();
  startTransition(() => {
    navigate(next ? `/catalog?q=${encodeURIComponent(next)}` : '/catalog');
  });
}
```

and replace the top section with:

```tsx
<SurfaceCard variant="hero" className="portal-hero">
  <form className="portal-hero__search" onSubmit={handleSearchSubmit}>
    <label className="field">
      <span>Search skills</span>
      <input
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        placeholder="review, pdf, governance, import"
      />
    </label>
    <button className="button button--primary" type="submit">
      Search catalog
    </button>
  </form>
</SurfaceCard>
```

Use `SkillSummaryCard` for curated items and add a small `Console` entry card near the recent-change column.

Add discovery-shaped loading placeholders to `OverviewPage.tsx`:

```tsx
{skills.loading || curated.loading ? (
  <div className="portal-skeleton-grid" aria-hidden="true">
    <div className="portal-skeleton portal-skeleton--hero" />
    <div className="portal-skeleton" />
    <div className="portal-skeleton" />
  </div>
) : null}
```

Update `apps/skills-web/src/pages/CatalogPage.tsx` so it hydrates `q` from the URL and preserves it:

```tsx
import { startTransition, useDeferredValue, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { SkillSummaryCard } from '../components/SkillSummaryCard';

const [searchParams, setSearchParams] = useSearchParams();
const [query, setQuery] = useState(searchParams.get('q') ?? '');

useEffect(() => {
  setQuery(searchParams.get('q') ?? '');
}, [searchParams]);

function updateQuery(next: string) {
  setQuery(next);
  startTransition(() => {
    const params = new URLSearchParams(searchParams);
    if (next.trim()) params.set('q', next.trim());
    else params.delete('q');
    setSearchParams(params, { replace: true });
  });
}
```

and replace the empty state with:

```tsx
{skills.loading && resultItems.length === 0 ? (
  <div className="results-skeleton" aria-hidden="true">
    <div className="portal-skeleton" />
    <div className="portal-skeleton" />
    <div className="portal-skeleton" />
  </div>
) : null}

{!skills.loading && !skills.errorDetail && resultItems.length === 0 ? (
  <div className="empty-box">
    <strong>No skills matched this view.</strong>
    <p>Try a broader capability, package scope, or source filter.</p>
  </div>
) : null}
```

Update `apps/skills-web/src/styles.css` with discovery-specific layout classes:

```css
.portal-hero {
  display: grid;
  gap: 18px;
  padding: 28px;
}

.portal-hero__search {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 14px;
  align-items: end;
}

.skill-summary-card {
  display: grid;
  gap: 14px;
  padding: 20px;
  border-radius: 22px;
  background: var(--panel-strong);
  border: 1px solid var(--border);
}

.skill-summary-card__header,
.skill-summary-card__meta,
.skill-summary-card__actions {
  display: flex;
  gap: 10px;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
}

.skill-summary-card__provenance {
  color: var(--muted);
  font-size: 0.92rem;
}

.portal-skeleton-grid,
.results-skeleton {
  display: grid;
  gap: 16px;
}

.portal-skeleton {
  min-height: 140px;
  border-radius: 22px;
  background: linear-gradient(90deg, rgba(157, 176, 201, 0.08), rgba(157, 176, 201, 0.16), rgba(157, 176, 201, 0.08));
  background-size: 200% 100%;
  animation: portalSkeleton 1.4s ease infinite;
}

.portal-skeleton--hero {
  min-height: 220px;
}

@keyframes portalSkeleton {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
```

- [ ] **Step 4: Run the discovery tests and one existing detail-page smoke test**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/pages/OverviewPage.test.tsx src/pages/CatalogPage.test.tsx src/pages/SkillDetailPage.test.tsx
```

Expected: PASS. The new page tests should pass and `SkillDetailPage.test.tsx` should remain green, proving the discovery-page changes did not break shared session/test setup.

- [ ] **Step 5: Commit the discovery-first pages**

```bash
cd /Users/ultraman/workspace/skill_registry
git add apps/skills-web/src/components/SkillSummaryCard.tsx apps/skills-web/src/pages/OverviewPage.test.tsx apps/skills-web/src/pages/CatalogPage.test.tsx apps/skills-web/src/pages/OverviewPage.tsx apps/skills-web/src/pages/CatalogPage.tsx apps/skills-web/src/styles.css
git commit -m "feat: redesign portal overview and catalog"
```

## Task 3: Make Skill and Version Detail Pages Install-First

**Files:**
- Create: `apps/skills-web/src/lib/installCommand.ts`
- Create: `apps/skills-web/src/lib/installCommand.test.ts`
- Create: `apps/skills-web/src/components/InstallCommandPanel.tsx`
- Create: `apps/skills-web/src/components/InstallCommandPanel.test.tsx`
- Modify: `apps/skills-web/src/pages/SkillDetailPage.tsx`
- Modify: `apps/skills-web/src/pages/SkillDetailPage.test.tsx`
- Modify: `apps/skills-web/src/pages/VersionDetailPage.tsx`
- Modify: `apps/skills-web/src/pages/VersionDetailPage.test.tsx`
- Modify: `apps/skills-web/src/styles.css`
- Test: `apps/skills-web/src/lib/installCommand.test.ts`
- Test: `apps/skills-web/src/components/InstallCommandPanel.test.tsx`
- Test: `apps/skills-web/src/pages/SkillDetailPage.test.tsx`
- Test: `apps/skills-web/src/pages/VersionDetailPage.test.tsx`

- [ ] **Step 1: Write the failing install-panel tests**

Create `apps/skills-web/src/lib/installCommand.test.ts` with:

```ts
import { describe, expect, it } from 'vitest';
import { buildInstallCommand } from './installCommand';

describe('buildInstallCommand', () => {
  it('builds an npm install command pinned to version and registry', () => {
    expect(
      buildInstallCommand({
        packageName: '@platform.skill/agent-notes',
        version: '1.2.3',
        npmRegistry: 'https://npm.internal.example.com',
      })
    ).toBe('npm install @platform.skill/agent-notes@1.2.3 --registry=https://npm.internal.example.com');
  });
});
```

Create `apps/skills-web/src/components/InstallCommandPanel.test.tsx` with:

```tsx
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { InstallCommandPanel } from './InstallCommandPanel';

describe('InstallCommandPanel', () => {
  it('copies the generated install command', async () => {
    Object.defineProperty(window.navigator, 'clipboard', {
      value: { writeText: vi.fn().mockResolvedValue(undefined) },
      configurable: true,
    });

    render(
      <InstallCommandPanel
        title="Install this skill"
        packageName="@platform.skill/agent-notes"
        version="1.2.3"
        npmRegistry="https://npm.internal.example.com"
        installPolicy="resolvable"
      />
    );

    fireEvent.click(screen.getByRole('button', { name: 'Copy install command' }));

    await waitFor(() =>
      expect(window.navigator.clipboard.writeText).toHaveBeenCalledWith(
        'npm install @platform.skill/agent-notes@1.2.3 --registry=https://npm.internal.example.com'
      )
    );
  });
});
```

Extend `apps/skills-web/src/pages/SkillDetailPage.test.tsx` with:

```tsx
it('shows the install command panel above the version list when install target data is present', async () => {
  ...
  expect(await screen.findByText('Install this skill')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Copy install command' })).toBeInTheDocument();
});
```

Extend `apps/skills-web/src/pages/VersionDetailPage.test.tsx` with:

```tsx
it('shows the install command panel ahead of release and governance forms', async () => {
  ...
  expect(await screen.findByText('Install this version')).toBeInTheDocument();
  expect(screen.getByText('npm install @platform.skill/agent-notes@1.0.0 --registry=https://npm.internal.example.com')).toBeInTheDocument();
});
```

- [ ] **Step 2: Run the install-focused tests to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/lib/installCommand.test.ts src/components/InstallCommandPanel.test.tsx src/pages/SkillDetailPage.test.tsx src/pages/VersionDetailPage.test.tsx
```

Expected: FAIL because the install helper and panel do not exist, and the detail pages do not render the new install-first section.

- [ ] **Step 3: Implement the shared install helper/panel and reflow the detail pages**

Create `apps/skills-web/src/lib/installCommand.ts` with:

```ts
export function buildInstallCommand(input: {
  packageName: string;
  version: string;
  npmRegistry: string;
}) {
  return `npm install ${input.packageName}@${input.version} --registry=${input.npmRegistry}`;
}
```

Create `apps/skills-web/src/components/InstallCommandPanel.tsx` with:

```tsx
import { useState } from 'react';
import type { InstallPolicy } from '../api/types';
import { buildInstallCommand } from '../lib/installCommand';
import { StatusPill } from './StatusPill';
import { SurfaceCard } from './SurfaceCard';

interface InstallCommandPanelProps {
  title: string;
  packageName: string;
  version: string;
  npmRegistry: string;
  installPolicy: InstallPolicy;
  tarballUrl?: string;
  integrity?: string;
}

export function InstallCommandPanel(props: InstallCommandPanelProps) {
  const [copied, setCopied] = useState(false);
  const command = buildInstallCommand(props);
  const installBlocked = props.installPolicy === 'blocked';

  async function handleCopy() {
    await navigator.clipboard.writeText(command);
    setCopied(true);
  }

  return (
    <SurfaceCard title={props.title} variant="hero">
      <div className="install-command-panel">
        <code>{command}</code>
        <div className="pill-row">
          <StatusPill tone={installBlocked ? 'bad' : props.installPolicy === 'catalog_only' ? 'warn' : 'good'}>
            {props.installPolicy}
          </StatusPill>
          <StatusPill>{props.npmRegistry}</StatusPill>
        </div>
        <div className="list-card__actions">
          <button className="button button--primary" disabled={installBlocked} onClick={handleCopy} type="button">
            Copy install command
          </button>
          {copied ? <span className="install-command-panel__copied">Copied</span> : null}
        </div>
      </div>
    </SurfaceCard>
  );
}
```

Update `apps/skills-web/src/pages/SkillDetailPage.tsx` to render the install panel before version/governance sections:

```tsx
import { InstallCommandPanel } from '../components/InstallCommandPanel';

const installVersion = installTarget.data?.version || detail.data?.latestVersion || '';
const installRegistry = installTarget.data?.npmRegistry || '';

{installTarget.data ? (
  <InstallCommandPanel
    title="Install this skill"
    packageName={decodedPackage}
    version={installVersion}
    npmRegistry={installRegistry}
    installPolicy={latestVersionItem?.installPolicy || 'catalog_only'}
    tarballUrl={installTarget.data.tarballUrl}
    integrity={installTarget.data.integrity}
  />
) : (
  <SurfaceCard title="Install status" variant="hero">
    <p>No install target is available yet. Review the latest version status before proceeding.</p>
  </SurfaceCard>
)}
```

Update `apps/skills-web/src/pages/VersionDetailPage.tsx` similarly, above release/governance forms:

```tsx
<InstallCommandPanel
  title="Install this version"
  packageName={decodedPackage}
  version={version}
  npmRegistry={detail.data?.registry.npmRegistry || 'registry unavailable'}
  installPolicy={detail.data?.installPolicy || 'catalog_only'}
  tarballUrl={detail.data?.registry.tarballUrl}
  integrity={detail.data?.registry.integrity}
/>
```

Update `apps/skills-web/src/styles.css` with:

```css
.install-command-panel {
  display: grid;
  gap: 14px;
}

.install-command-panel code {
  display: block;
  padding: 16px 18px;
  border-radius: 16px;
  background: rgba(3, 9, 18, 0.78);
  color: var(--text);
  overflow-x: auto;
  font-variant-numeric: tabular-nums;
}

.install-command-panel__copied {
  color: var(--good);
  font-size: 0.92rem;
}
```

- [ ] **Step 4: Run the install tests and a focused build**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/lib/installCommand.test.ts src/components/InstallCommandPanel.test.tsx src/pages/SkillDetailPage.test.tsx src/pages/VersionDetailPage.test.tsx && npm run build
```

Expected: PASS. The helper and panel tests should pass, both detail-page tests should remain green, and the build should succeed.

- [ ] **Step 5: Commit the install-first detail pages**

```bash
cd /Users/ultraman/workspace/skill_registry
git add apps/skills-web/src/lib/installCommand.ts apps/skills-web/src/lib/installCommand.test.ts apps/skills-web/src/components/InstallCommandPanel.tsx apps/skills-web/src/components/InstallCommandPanel.test.tsx apps/skills-web/src/pages/SkillDetailPage.tsx apps/skills-web/src/pages/SkillDetailPage.test.tsx apps/skills-web/src/pages/VersionDetailPage.tsx apps/skills-web/src/pages/VersionDetailPage.test.tsx apps/skills-web/src/styles.css
git commit -m "feat: make detail pages install first"
```

## Task 4: Reframe Workbench Pages as a Secondary Operations Console and Verify the Whole App

**Files:**
- Create: `apps/skills-web/src/pages/WorkbenchHomePage.test.tsx`
- Create: `apps/skills-web/src/pages/AuditPage.test.tsx`
- Modify: `apps/skills-web/src/pages/WorkbenchHomePage.tsx`
- Modify: `apps/skills-web/src/pages/GovernancePage.tsx`
- Modify: `apps/skills-web/src/pages/AuditPage.tsx`
- Modify: `apps/skills-web/src/styles.css`
- Test: `apps/skills-web/src/pages/WorkbenchHomePage.test.tsx`
- Test: `apps/skills-web/src/pages/GovernancePage.test.tsx`
- Test: `apps/skills-web/src/pages/AuditPage.test.tsx`

- [ ] **Step 1: Write the failing Console-page tests**

Create `apps/skills-web/src/pages/WorkbenchHomePage.test.tsx` with:

```tsx
import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import * as client from '../api/client';
import { WorkbenchHomePage } from './WorkbenchHomePage';

vi.mock('../api/client', () => ({
  listAuditEvents: vi.fn(),
  listGovernanceChanges: vi.fn(),
  listSkills: vi.fn(),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('WorkbenchHomePage', () => {
  it('frames workbench as Console Home with action-oriented entry cards', async () => {
    vi.mocked(client.listSkills).mockResolvedValue({ items: [], pageInfo: { hasNextPage: false, nextCursor: null } });
    vi.mocked(client.listGovernanceChanges).mockResolvedValue({ items: [], pageInfo: { hasNextPage: false, nextCursor: null } });
    vi.mocked(client.listAuditEvents).mockResolvedValue({ items: [], pageInfo: { hasNextPage: false, nextCursor: null } });

    render(
      <MemoryRouter>
        <WorkbenchHomePage />
      </MemoryRouter>
    );

    expect(await screen.findByRole('heading', { name: 'Console Home' })).toBeInTheDocument();
    expect(screen.getByText('Import and package')).toBeInTheDocument();
    expect(screen.getByText('Governance and audit')).toBeInTheDocument();
  });
});
```

Create `apps/skills-web/src/pages/AuditPage.test.tsx` with:

```tsx
import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import * as client from '../api/client';
import { SessionProvider } from '../session/SessionProvider';
import { AuditPage } from './AuditPage';

vi.mock('../api/client', () => ({
  getRegistryClientConfig: vi.fn(() => ({
    mode: 'real',
    baseUrl: null,
    role: null,
    actor: null,
    hasToken: false,
  })),
  getSession: vi.fn(),
  listAuditEvents: vi.fn(),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('AuditPage', () => {
  it('shows summary-first empty guidance when there are no audit rows', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });
    vi.mocked(client.listAuditEvents).mockResolvedValue({
      items: [],
      pageInfo: { hasNextPage: false, nextCursor: null },
    });

    render(
      <SessionProvider>
        <MemoryRouter>
          <AuditPage />
        </MemoryRouter>
      </SessionProvider>
    );

    expect(await screen.findByText('No audit events matched this view.')).toBeInTheDocument();
    expect(screen.getByText('Adjust actor, package, or outcome filters to widen the trace window.')).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run the Console-page tests to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/pages/WorkbenchHomePage.test.tsx src/pages/GovernancePage.test.tsx src/pages/AuditPage.test.tsx
```

Expected: FAIL because `WorkbenchHomePage` still presents itself as a generic workbench and `AuditPage` does not yet show the new empty guidance.

- [ ] **Step 3: Reframe Console home, governance, and audit around operations workflows**

Update `apps/skills-web/src/pages/WorkbenchHomePage.tsx` so the page title and cards match the new Console language:

```tsx
<SectionHeading
  eyebrow="Console"
  title="Console Home"
  description="Run import, authoring, governance, and audit workflows without crowding the Portal."
  tone="backstage"
/>
```

Replace the old list with grouped cards:

```tsx
<div className="console-home-grid">
  <SurfaceCard title="Import and package" description="Prepare, validate, and pack external or authored skills." variant="console">
    <div className="list-card__actions">
      <Link className="button button--primary" to="/workbench/import">Open import</Link>
      <Link className="button button--secondary" to="/workbench/authoring">Open authoring</Link>
    </div>
  </SurfaceCard>
  <SurfaceCard title="Governance and audit" description="Review package-impacting actions and operator traces." variant="console">
    <div className="list-card__actions">
      <Link className="button button--secondary" to="/workbench/governance">Open governance</Link>
      <Link className="button button--secondary" to="/workbench/audit">Open audit</Link>
    </div>
  </SurfaceCard>
</div>
```

Add Console-shaped loading placeholders to `WorkbenchHomePage.tsx`:

```tsx
{skills.loading || governance.loading || audit.loading ? (
  <div className="console-skeleton-grid" aria-hidden="true">
    <div className="portal-skeleton" />
    <div className="portal-skeleton" />
  </div>
) : null}
```

Update `apps/skills-web/src/pages/GovernancePage.tsx` to lead with impact summary language:

```tsx
<SectionHeading
  eyebrow="Console / Governance"
  title="Governance impact"
  description="Review package-wide or version-wide actions before drilling into the event stream."
  tone="backstage"
/>
```

Update `apps/skills-web/src/pages/AuditPage.tsx` similarly:

```tsx
<SectionHeading
  eyebrow="Console / Audit"
  title="Audit trace"
  description="Start with scope and outcomes, then inspect the underlying event rows."
  tone="backstage"
/>
```

and replace the empty state with:

```tsx
{!audit.loading && !audit.error && filteredItems.length === 0 ? (
  <div className="empty-box">
    <strong>No audit events matched this view.</strong>
    <p>Adjust actor, package, or outcome filters to widen the trace window.</p>
  </div>
) : null}
```

Update `apps/skills-web/src/styles.css` with:

```css
.console-home-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px;
}

.console-skeleton-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px;
}

@media (max-width: 960px) {
  .portal-hero__search,
  .console-home-grid,
  .console-skeleton-grid {
    grid-template-columns: 1fr;
  }
}
```

- [ ] **Step 4: Run the full unit suite, build, and do a manual route sweep**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test && npm run build
```

Then run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm run dev
```

Manual checks in the browser:

- `/`
- `/catalog?q=review`
- open the first skill card from `/catalog`
- open the latest version link from that skill detail page
- `/workbench`
- `/workbench/governance`
- `/workbench/audit`
- `/privacy`
- `/terms`
- `/does-not-exist`

Expected: the unit suite passes, the build succeeds, and each manual route reads as either `Portal` or `Console` without dead ends or broken navigation.

- [ ] **Step 5: Commit the Console framing and final verification**

```bash
cd /Users/ultraman/workspace/skill_registry
git add apps/skills-web/src/pages/WorkbenchHomePage.test.tsx apps/skills-web/src/pages/AuditPage.test.tsx apps/skills-web/src/pages/WorkbenchHomePage.tsx apps/skills-web/src/pages/GovernancePage.tsx apps/skills-web/src/pages/AuditPage.tsx apps/skills-web/src/styles.css
git commit -m "feat: align workbench pages to console"
```
