# Platform Authoring Import Web Upgrade Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade `platform_authoring` in the web import flow from raw JSON entry to a template-driven multi-file authoring panel.

**Architecture:** Keep the backend `source.type=platform_authoring` contract unchanged. Replace the current inline JSON textarea in `ImportPage` with a local file-model editor that can generate `Record<string,string>` payloads, while keeping an advanced raw JSON escape hatch for debugging.

**Tech Stack:** React, TypeScript, Vitest, Testing Library, existing `skills-web` CSS surface

---

## File Map

- Modify: `apps/skills-web/src/pages/ImportPage.tsx`
  - Add authoring template definitions
  - Add file-list state and conversion helpers
  - Replace the platform authoring textarea UI
- Modify: `apps/skills-web/src/pages/ImportPage.test.tsx`
  - Add tests for template defaults, file editing, file deletion, and `prepare` request payloads
- Modify: `apps/skills-web/src/styles.css`
  - Add layout styles for template bar, file list, editor, and advanced JSON area

### Task 1: Cover Current Gap With Tests

**Files:**
- Modify: `apps/skills-web/src/pages/ImportPage.test.tsx`
- Test: `apps/skills-web/src/pages/ImportPage.test.tsx`

- [ ] **Step 1: Write a failing test for platform authoring default UI**

```tsx
it('shows the platform authoring template editor instead of only a raw json textarea', async () => {
  render(
    <MemoryRouter>
      <ImportPage />
    </MemoryRouter>
  );

  fireEvent.change(await screen.findByLabelText('来源类型'), {
    target: { value: 'platform_authoring' },
  });

  expect(screen.getByText('模板')).toBeInTheDocument();
  expect(screen.getByDisplayValue('commands/run.txt')).toBeInTheDocument();
});
```

- [ ] **Step 2: Run the test to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/pages/ImportPage.test.tsx
```

Expected: FAIL because the page still renders only the inline JSON textarea.

- [ ] **Step 3: Add failing tests for template switching and prepare payload**

```tsx
it('switches templates and sends the generated files in prepare payload', async () => {
  vi.mocked(client.preparePackaging).mockResolvedValue({
    workspacePath: '/tmp/skill-packaging/workspace',
    packageName: '@company.skill/zoom-out',
    version: '1.2.0',
    warnings: [],
    warningDetails: [],
    errors: [],
  });

  render(
    <MemoryRouter>
      <ImportPage />
    </MemoryRouter>
  );

  fireEvent.change(await screen.findByLabelText('来源类型'), {
    target: { value: 'platform_authoring' },
  });

  fireEvent.change(screen.getByLabelText('模板'), {
    target: { value: 'workflow' },
  });
  fireEvent.change(screen.getByLabelText('当前文件内容'), {
    target: { value: 'step 1\nstep 2\n' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Prepare' }));

  expect(client.preparePackaging).toHaveBeenCalledWith(
    expect.objectContaining({
      source: expect.objectContaining({
        type: 'platform_authoring',
        files: {
          'workflows/main.md': 'step 1\nstep 2\n',
        },
      }),
    })
  );
});
```

- [ ] **Step 4: Add a failing test for add/delete file behavior**

```tsx
it('allows adding and deleting non-locked authoring files', async () => {
  render(
    <MemoryRouter>
      <ImportPage />
    </MemoryRouter>
  );

  fireEvent.change(await screen.findByLabelText('来源类型'), {
    target: { value: 'platform_authoring' },
  });

  fireEvent.change(screen.getByLabelText('新文件路径'), {
    target: { value: 'docs/README.md' },
  });
  fireEvent.click(screen.getByRole('button', { name: '添加文件' }));

  expect(screen.getByDisplayValue('docs/README.md')).toBeInTheDocument();

  fireEvent.click(screen.getByRole('button', { name: '删除 docs/README.md' }));

  expect(screen.queryByDisplayValue('docs/README.md')).not.toBeInTheDocument();
});
```

- [ ] **Step 5: Run the test file again to confirm RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/pages/ImportPage.test.tsx
```

Expected: FAIL on the new platform authoring expectations.

### Task 2: Implement Authoring State and Request Conversion

**Files:**
- Modify: `apps/skills-web/src/pages/ImportPage.tsx`

- [ ] **Step 1: Add local authoring model and template helpers near the top of `ImportPage.tsx`**

Add:

```ts
type AuthoringTemplateId = 'prompt' | 'command' | 'workflow';

type AuthoringFile = {
  path: string;
  content: string;
  locked: boolean;
};
```

and a template map:

```ts
const AUTHORING_TEMPLATES: Record<
  AuthoringTemplateId,
  {
    label: string;
    description: string;
    files: Array<{ path: string; content: string; locked?: boolean }>;
  }
> = {
  prompt: {
    label: 'prompt',
    description: '单 prompt skill',
    files: [{ path: 'prompts/main.md', content: 'Write the prompt here.\n', locked: true }],
  },
  command: {
    label: 'command',
    description: '单命令 skill',
    files: [{ path: 'commands/run.txt', content: 'echo authored\n', locked: true }],
  },
  workflow: {
    label: 'workflow',
    description: '步骤型 workflow skill',
    files: [{ path: 'workflows/main.md', content: '1. First step\n2. Second step\n', locked: true }],
  },
};
```

- [ ] **Step 2: Add pure helpers for file conversion and validation**

Add helpers:

```ts
function createAuthoringFiles(template: AuthoringTemplateId): AuthoringFile[] { ... }
function filesToRecord(files: AuthoringFile[]): Record<string, string> { ... }
function validateAuthoringPath(path: string, existing: AuthoringFile[]): string | null { ... }
function recordToFiles(files: Record<string, string>): AuthoringFile[] { ... }
```

Rules:
- reject empty path
- reject duplicates
- reject `..`
- reject absolute path
- reject `SKILL.md`

- [ ] **Step 3: Replace `inlineFiles`-only state with authoring panel state**

Introduce state:

```ts
const [authoringTemplate, setAuthoringTemplate] = useState<AuthoringTemplateId>('command');
const [authoringFiles, setAuthoringFiles] = useState<AuthoringFile[]>(() => createAuthoringFiles('command'));
const [selectedAuthoringPath, setSelectedAuthoringPath] = useState('commands/run.txt');
const [newAuthoringPath, setNewAuthoringPath] = useState('');
const [authoringError, setAuthoringError] = useState<string | null>(null);
const [showAdvancedFiles, setShowAdvancedFiles] = useState(false);
const [rawInlineFiles, setRawInlineFiles] = useState('');
const [rawInlineFilesDirty, setRawInlineFilesDirty] = useState(false);
```

- [ ] **Step 4: Update `buildPrepareRequest()` to serialize authoring files**

Use:

```ts
const platformFiles =
  rawInlineFilesDirty && rawInlineFiles.trim() !== ''
    ? parseInlineFiles(rawInlineFiles)
    : filesToRecord(authoringFiles);
```

and send:

```ts
files: platformFiles,
```

- [ ] **Step 5: Run the tests to verify GREEN for request payload behavior**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/pages/ImportPage.test.tsx
```

Expected: template/payload tests move closer to passing; UI expectations may still fail until rendering is updated.

### Task 3: Replace Platform Authoring UI

**Files:**
- Modify: `apps/skills-web/src/pages/ImportPage.tsx`
- Modify: `apps/skills-web/src/styles.css`

- [ ] **Step 1: Replace the platform authoring JSON textarea block with a multi-panel editor**

Render:

```tsx
<div className="authoring-editor">
  <label className="field">
    <span>模板</span>
    <select ... />
  </label>
  <div className="authoring-editor__layout">
    <div className="authoring-editor__files">...</div>
    <div className="authoring-editor__content">...</div>
  </div>
</div>
```

Include:
- template picker
- file list buttons
- add-file input/button
- current file content textarea
- delete button for non-locked files

- [ ] **Step 2: Add event handlers for template reset, file add, file select, file delete, and file content edit**

Implement small handlers:

```ts
function resetAuthoringTemplate(nextTemplate: AuthoringTemplateId) { ... }
function handleAddAuthoringFile() { ... }
function handleDeleteAuthoringFile(path: string) { ... }
function handleAuthoringContentChange(value: string) { ... }
```

- [ ] **Step 3: Add advanced JSON toggle below the visual editor**

Render a collapsible section with:

```tsx
<button type="button">高级模式 / 原始 files JSON</button>
```

and a textarea bound to `rawInlineFiles`.

- [ ] **Step 4: Add styles for the new authoring editor**

Extend `styles.css` with:

```css
.authoring-editor { display: grid; gap: 16px; }
.authoring-editor__layout { display: grid; grid-template-columns: 240px minmax(0, 1fr); gap: 16px; }
.authoring-editor__files,
.authoring-editor__content,
.authoring-editor__advanced { ... }
.authoring-file-row,
.authoring-template-row { ... }
```

Keep mobile behavior simple by collapsing to one column under the existing responsive breakpoint.

- [ ] **Step 5: Run tests again to verify GREEN**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/pages/ImportPage.test.tsx
```

Expected: PASS for the new `platform_authoring` tests and existing `git` tests.

### Task 4: Verify and Close Out

**Files:**
- Modify if needed: `apps/skills-web/src/pages/ImportPage.tsx`
- Modify if needed: `apps/skills-web/src/styles.css`

- [ ] **Step 1: Run focused web typecheck**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && npm --prefix apps/skills-web run typecheck
```

Expected: PASS

- [ ] **Step 2: Run the focused test suite one more time**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/pages/ImportPage.test.tsx
```

Expected: PASS

- [ ] **Step 3: Commit only the implementation files**

```bash
git -C /Users/ultraman/workspace/skill_registry add \
  apps/skills-web/src/pages/ImportPage.tsx \
  apps/skills-web/src/pages/ImportPage.test.tsx \
  apps/skills-web/src/styles.css \
  docs/superpowers/plans/2026-06-05-platform-authoring-import-web-upgrade.md

git -C /Users/ultraman/workspace/skill_registry commit -m "feat: improve platform authoring import flow"
```

- [ ] **Step 4: Move task #40 to `in_review` and summarize verification**

Report:
- tests run
- typecheck result
- any remaining UX gaps intentionally deferred
