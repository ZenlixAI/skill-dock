# Skills Web Login Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the `skills-web` login route into a minimal console-style two-column entry page without changing session logic, redirect behavior, or authentication APIs.

**Architecture:** Keep the existing route wiring and session flow intact, and concentrate the redesign in the login route component plus route-local CSS classes inside the shared stylesheet. Add focused page-level tests around the new structure and error behavior so the visual refresh does not weaken login semantics.

**Tech Stack:** React 19, React Router 7, Vitest, Testing Library, Vite, CSS custom properties

---

## File Map

- `apps/skills-web/src/pages/LoginPage.tsx`
  - Owns the login route structure, controlled form state, submit lifecycle, redirect rules, and inline error rendering.
- `apps/skills-web/src/pages/LoginPage.test.tsx`
  - New focused route test file for the redesigned login layout, redirect behavior, and inline error handling.
- `apps/skills-web/src/styles.css`
  - Owns the new login-specific layout, surface, field, button, and responsive rules while reusing existing theme tokens.
- `apps/skills-web/package.json`
  - No edits expected; use the existing `test` and `build` scripts for verification.

### Task 1: Add Focused Login Page Regression Tests

**Files:**
- Create: `apps/skills-web/src/pages/LoginPage.test.tsx`

- [ ] **Step 1: Add a page-level render helper and default mocks**

Create `apps/skills-web/src/pages/LoginPage.test.tsx` with the shared render helper and client mocks so login behavior can be tested without going through unrelated shell tests:

```tsx
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import * as client from '../api/client';
import { LoginPage } from './LoginPage';
import { SessionProvider } from '../session/SessionProvider';

vi.mock('../api/client', () => ({
  getRegistryClientConfig: vi.fn(() => ({
    mode: 'real',
    baseUrl: null,
    role: null,
    actor: null,
    hasToken: false,
  })),
  getSession: vi.fn(),
  loginSession: vi.fn(),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function renderLoginPage(initialEntry = '/login?next=%2Fcatalog') {
  return render(
    <SessionProvider>
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route path="/catalog" element={<div>catalog destination</div>} />
          <Route path="/login" element={<LoginPage />} />
        </Routes>
      </MemoryRouter>
    </SessionProvider>
  );
}
```

- [ ] **Step 2: Lock down the redesigned two-zone layout copy**

Add a render test that asserts the new context block and form region both appear, while keeping the existing core field labels and action label:

```tsx
it('renders the console login context and form controls', async () => {
  vi.mocked(client.getSession).mockResolvedValue(null);

  renderLoginPage();

  expect(await screen.findByText('控制台登录')).toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 1, name: '进入技能注册表控制台' })).toBeInTheDocument();
  expect(
    screen.getByText('导入、编写、治理和审计流程统一收口到这个控制台入口。')
  ).toBeInTheDocument();
  expect(screen.getByLabelText('用户名')).toBeInTheDocument();
  expect(screen.getByLabelText('密码')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: '登录' })).toBeInTheDocument();
});
```

- [ ] **Step 3: Add an inline error-state test with accessible announcement**

Add a submit failure test that proves login errors stay inline and are exposed through a live region:

```tsx
it('announces login errors inline when submission fails', async () => {
  vi.mocked(client.getSession).mockResolvedValue(null);
  vi.mocked(client.loginSession).mockRejectedValue(new Error('用户名或密码不正确'));

  renderLoginPage();

  fireEvent.change(screen.getByLabelText('用户名'), { target: { value: 'reader' } });
  fireEvent.change(screen.getByLabelText('密码'), { target: { value: 'wrong-secret' } });
  fireEvent.click(screen.getByRole('button', { name: '登录' }));

  const error = await screen.findByRole('status');
  expect(error).toHaveTextContent('用户名或密码不正确');
  expect(error).toHaveAttribute('aria-live', 'polite');
  expect(screen.getByRole('button', { name: '登录' })).not.toBeDisabled();
});
```

- [ ] **Step 4: Keep redirect behavior covered**

Add a success-path test that confirms the page still redirects to `next` after a successful login:

```tsx
it('redirects to next after a successful login', async () => {
  vi.mocked(client.getSession)
    .mockResolvedValueOnce(null)
    .mockResolvedValueOnce({
      role: 'reader',
      actor: 'dev@company',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });
  vi.mocked(client.loginSession).mockResolvedValue(undefined);

  renderLoginPage();

  fireEvent.change(screen.getByLabelText('用户名'), { target: { value: 'reader' } });
  fireEvent.change(screen.getByLabelText('密码'), { target: { value: 'reader-secret' } });
  fireEvent.click(screen.getByRole('button', { name: '登录' }));

  expect(await screen.findByText('catalog destination')).toBeInTheDocument();
});
```

- [ ] **Step 5: Run the new login page tests**

Run:

```bash
cd apps/skills-web
npm test -- src/pages/LoginPage.test.tsx
```

Expected: `FAIL` because the current login page does not yet render the new context block or accessible inline error region.

### Task 2: Rebuild The Login Route Markup Around The Approved Two-Column Design

**Files:**
- Modify: `apps/skills-web/src/pages/LoginPage.tsx`
- Test: `apps/skills-web/src/pages/LoginPage.test.tsx`

- [ ] **Step 1: Replace the generic page-grid composition with route-local login sections**

Update `apps/skills-web/src/pages/LoginPage.tsx` so the rendered structure becomes a dedicated login page instead of a `SectionHeading` + `SurfaceCard` content stack:

```tsx
return (
  <main className="login-page" id="main-content">
    <section className="login-layout" aria-labelledby="login-title">
      <div className="login-context">
        <span className="login-context__eyebrow">控制台登录</span>
        <p className="login-context__kicker">技能注册表</p>
        <h1 id="login-title">进入技能注册表控制台</h1>
        <p className="login-context__description">
          导入、编写、治理和审计流程统一收口到这个控制台入口。
        </p>
      </div>

      <section className="login-panel" aria-label="登录表单区域">
        <div className="login-panel__header">
          <p className="login-panel__eyebrow">身份验证</p>
          <h2>登录</h2>
          <p>使用你的仓库账号继续访问控制台。</p>
        </div>
```

- [ ] **Step 2: Keep the existing controlled form flow, but strengthen semantics**

Retain the current state variables and submit handler, while updating the form subtree to use explicit ids, a live error region, and a full-width submit action:

```tsx
        <form className="login-form" onSubmit={handleSubmit}>
          <label className="field login-form__field" htmlFor="login-username">
            <span>用户名</span>
            <input
              id="login-username"
              autoComplete="username"
              onChange={(event) => setUsername(event.target.value)}
              value={username}
            />
          </label>
          <label className="field login-form__field" htmlFor="login-password">
            <span>密码</span>
            <input
              id="login-password"
              autoComplete="current-password"
              onChange={(event) => setPassword(event.target.value)}
              type="password"
              value={password}
            />
          </label>
          {error ? (
            <div aria-live="polite" className="error-box login-form__error" role="status">
              {error}
            </div>
          ) : null}
          <button className="button button--primary login-form__submit" disabled={submitting} type="submit">
            {submitting ? '登录中…' : '登录'}
          </button>
        </form>
      </section>
    </section>
  </main>
);
```

- [ ] **Step 3: Remove no-longer-used generic imports**

Trim the `LoginPage.tsx` imports so only the route and session utilities remain:

```tsx
import { FormEvent, useState } from 'react';
import { Navigate, useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { getRegistryClientConfig, loginSession } from '../api/client';
import { useRegistrySession } from '../session/SessionProvider';
```

- [ ] **Step 4: Run the login page tests after the markup change**

Run:

```bash
cd apps/skills-web
npm test -- src/pages/LoginPage.test.tsx
```

Expected: `PASS` for the new page tests.

### Task 3: Add Login-Specific Layout, Surface, And Interaction Styling

**Files:**
- Modify: `apps/skills-web/src/styles.css`
- Test: `apps/skills-web/src/pages/LoginPage.test.tsx`

- [ ] **Step 1: Add a dedicated login layout block near the shared page layout rules**

Insert login-specific layout classes after `.page-grid` so the route gets its own two-column composition:

```css
.login-page {
  min-height: 100dvh;
  padding: 32px 24px 40px;
}

.login-layout {
  max-width: 1280px;
  min-height: calc(100dvh - 72px);
  margin: 0 auto;
  display: grid;
  grid-template-columns: minmax(18rem, 0.88fr) minmax(22rem, 1.12fr);
  gap: clamp(1.5rem, 3vw, 3.5rem);
  align-items: center;
}

.login-context {
  position: relative;
  display: grid;
  gap: 18px;
  max-width: 28rem;
  padding: 1.25rem 0 1.25rem 1.5rem;
}
```

- [ ] **Step 2: Add the context column typography and subtle structural accent**

Add the left-column styling so the page reads like a console entry instead of a landing page:

```css
.login-context::before {
  content: "";
  position: absolute;
  inset: 0 auto 0 0;
  width: 1px;
  background: linear-gradient(180deg, transparent, rgba(210, 138, 82, 0.44), transparent);
}

.login-context__eyebrow,
.login-panel__eyebrow {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  padding: 0.45rem 0.7rem;
  border-radius: 999px;
  border: 1px solid var(--surface-quiet-border);
  background: rgba(48, 34, 25, 0.54);
  color: var(--accent);
  font-size: 0.77rem;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.login-context h1 {
  margin: 0;
  max-width: 10ch;
  font-size: clamp(2.8rem, 5vw, 4.8rem);
  line-height: 0.9;
  letter-spacing: -0.05em;
}
```

- [ ] **Step 3: Add the strengthened login panel, field, button, and error styles**

Add the right-column console surface and its interaction rules using the existing token system:

```css
.login-panel {
  padding: clamp(1.5rem, 3vw, 2.25rem);
  border: 1px solid var(--border-strong);
  border-radius: 2rem;
  background:
    linear-gradient(180deg, rgba(45, 32, 25, 0.96), rgba(28, 20, 15, 0.92)),
    radial-gradient(circle at top left, rgba(210, 138, 82, 0.18), transparent 30%);
  box-shadow:
    inset 0 1px 0 rgba(246, 237, 227, 0.06),
    var(--shadow-strong);
  backdrop-filter: blur(20px);
}

.login-form__field input:hover {
  border-color: var(--accent-border-emphasis);
  background:
    linear-gradient(180deg, rgba(55, 39, 29, 0.96), rgba(34, 24, 18, 0.9)),
    rgba(33, 24, 18, 0.88);
}

.login-form__submit {
  width: 100%;
  min-height: 3.5rem;
  margin-top: 0.5rem;
}

.login-form__error {
  padding: 0.95rem 1rem;
  border-style: solid;
}
```

- [ ] **Step 4: Add responsive collapse rules for tablet and mobile**

Extend the existing media queries so the login layout collapses cleanly on smaller screens:

```css
@media (max-width: 1180px) {
  .login-layout {
    min-height: auto;
    grid-template-columns: 1fr;
    align-items: start;
    padding-block: 2rem;
  }

  .login-context {
    max-width: none;
    padding-left: 1.25rem;
  }
}

@media (max-width: 820px) {
  .login-page {
    padding-inline: 18px;
  }

  .login-context {
    padding-left: 1rem;
  }

  .login-context h1 {
    max-width: none;
    font-size: clamp(2.3rem, 11vw, 3.4rem);
  }
}
```

- [ ] **Step 5: Run tests and a production build**

Run:

```bash
cd apps/skills-web
npm test -- src/pages/LoginPage.test.tsx src/components/AppShell.test.tsx
npm run build
```

Expected:

- tests: `PASS`
- build: Vite completes successfully and writes the production bundle

- [ ] **Step 6: Commit the login redesign**

Run:

```bash
git add apps/skills-web/src/pages/LoginPage.tsx apps/skills-web/src/pages/LoginPage.test.tsx apps/skills-web/src/styles.css
git commit -m "feat: redesign skills web login page"
```
