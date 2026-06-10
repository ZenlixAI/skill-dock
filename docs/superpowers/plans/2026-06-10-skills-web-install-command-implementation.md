# Skills Web Install Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Update the web install command panel to generate `npx --registry ... skills add ... --version ...` commands instead of `npm install ...`.

**Architecture:** The change is isolated to the frontend install command builder in `apps/skills-web/src/lib/installCommand.ts`. Existing UI components keep using the same interface shape, while tests are updated first to lock the new command contract before the implementation changes.

**Tech Stack:** React, TypeScript, Vitest, Testing Library

---

### Task 1: Update install command tests

**Files:**
- Modify: `apps/skills-web/src/components/InstallCommandPanel.test.tsx`
- Modify: `apps/skills-web/src/pages/VersionDetailPage.test.tsx`

- [ ] **Step 1: Write the failing test expectations**

Replace each existing `npm install ...` assertion with:

```ts
'npx --registry https://npm.internal.example.com skills add @platform.skill/agent-notes --version 1.2.3'
```

and the fallback expectation with:

```ts
'npx --registry <待补充 npm 仓库地址> skills add @platform.skill/agent-notes --version 1.2.3'
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
npm test -- --run apps/skills-web/src/components/InstallCommandPanel.test.tsx apps/skills-web/src/pages/VersionDetailPage.test.tsx
```

Expected: assertions fail because the UI still renders `npm install ...`.

### Task 2: Implement the new command builder

**Files:**
- Modify: `apps/skills-web/src/lib/installCommand.ts`

- [ ] **Step 1: Change the command builder**

Update `buildInstallCommand` to return:

```ts
return `npx --registry ${commandRegistry} skills add ${packageName} --version ${version}`;
```

- [ ] **Step 2: Re-run the targeted tests**

Run:

```bash
npm test -- --run apps/skills-web/src/components/InstallCommandPanel.test.tsx apps/skills-web/src/pages/VersionDetailPage.test.tsx
```

Expected: all targeted tests pass.

### Task 3: Verify no unintended display regressions

**Files:**
- Re-check: `apps/skills-web/src/components/InstallCommandPanel.tsx`

- [ ] **Step 1: Confirm component contract stays unchanged**

Verify the panel still:

```ts
buildInstallCommand({
  packageName,
  version,
  npmRegistry: registryLabel,
});
```

and still copies the generated command via `navigator.clipboard.writeText(command)`.

- [ ] **Step 2: Run the same targeted tests as final verification**

Run:

```bash
npm test -- --run apps/skills-web/src/components/InstallCommandPanel.test.tsx apps/skills-web/src/pages/VersionDetailPage.test.tsx
```

Expected: passing test run with no assertion failures in either file.
