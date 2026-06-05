# Skills Web Architecture Deepening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deepen `apps/skills-web` into clear session, packaging, mutation, and adapter modules while preserving current user-visible behavior.

**Architecture:** Keep the current routes, backend contracts, and UI flows intact, but move the real behavior behind smaller interfaces. `SessionProvider` becomes the single session seam, `ImportPage` consumes authoring and packaging hooks instead of owning the whole workflow, detail pages consume a shared Catalog mutation hook, and the wide API adapter is split internally by domain with a dedicated mock state store.

**Tech Stack:** React 19, TypeScript, React Router, Vitest, Testing Library, existing `skills-web` CSS surface

---

## File Map

- Create: `apps/skills-web/src/session/posture.ts`
  - Session posture and capability helpers used by provider, guards, and pages
- Create: `apps/skills-web/src/session/posture.test.ts`
  - Unit tests for posture and capability behavior
- Modify: `apps/skills-web/src/session/SessionProvider.tsx`
  - Expose richer session seam helpers and stop pages from bypassing it
- Modify: `apps/skills-web/src/components/RouteGuards.tsx`
  - Consume the strengthened session seam
- Modify: `apps/skills-web/src/components/PermissionGate.tsx`
  - Consume session seam posture/capability helpers
- Modify: `apps/skills-web/src/components/PermissionGate.test.tsx`
  - Keep gate behavior covered after seam changes

- Create: `apps/skills-web/src/features/packaging/useAuthoringDraft.ts`
  - Own template/file/JSON authoring state for `platform_authoring`
- Create: `apps/skills-web/src/features/packaging/useAuthoringDraft.test.ts`
  - Unit tests for authoring draft rules
- Create: `apps/skills-web/src/features/packaging/usePackagingWorkflow.ts`
  - Own prepare/validate/pack/cleanup workflow, mutation result persistence, and handoff generation
- Create: `apps/skills-web/src/features/packaging/usePackagingWorkflow.test.ts`
  - Unit tests for workflow stage transitions and mutation URL behavior
- Modify: `apps/skills-web/src/pages/ImportPage.tsx`
  - Become a composition layer over the new packaging modules
- Modify: `apps/skills-web/src/pages/ImportPage.test.tsx`
  - Shift coverage from internal page ceremony to hook consumption behavior

- Create: `apps/skills-web/src/features/catalog-mutations/useCatalogMutation.ts`
  - Shared mutation hook for release record and governance writes
- Create: `apps/skills-web/src/features/catalog-mutations/useCatalogMutation.test.ts`
  - Unit tests for mutation result normalization and refresh contract
- Modify: `apps/skills-web/src/pages/SkillDetailPage.tsx`
  - Use shared mutation hook for retire behavior and session seam for capability posture
- Modify: `apps/skills-web/src/pages/VersionDetailPage.tsx`
  - Use shared mutation hook for release record and version governance behavior

- Create: `apps/skills-web/src/api/catalogReadClient.ts`
  - Read-only Catalog adapter
- Create: `apps/skills-web/src/api/packagingClient.ts`
  - Packaging adapter
- Create: `apps/skills-web/src/api/catalogMutationClient.ts`
  - Release/governance mutation adapter
- Create: `apps/skills-web/src/api/sessionClient.ts`
  - Session adapter
- Create: `apps/skills-web/src/api/mockState.ts`
  - In-memory mock `State Store` for skills, versions, governance, and audit
- Modify: `apps/skills-web/src/api/client.ts`
  - Re-export the split adapters behind the existing public seam
- Modify: `apps/skills-web/src/api/mockClient.ts`
  - Delegate to mock state store instead of mixing fixtures, state mutation, and adapter orchestration
- Modify: `apps/skills-web/src/api/client.test.ts`
  - Keep facade normalization behavior covered
- Modify: `apps/skills-web/src/api/session.test.ts`
  - Verify the strengthened session seam still uses the correct adapter

- Modify: `apps/skills-web/src/pages/GovernancePage.tsx`
  - Consume the session seam instead of direct `getSession()` calls
- Modify: `apps/skills-web/src/pages/AuthoringFlowPage.tsx`
  - Consume the session seam instead of direct `getSession()` calls

## Task 1: Make the Session Seam Real

**Files:**
- Create: `apps/skills-web/src/session/posture.ts`
- Create: `apps/skills-web/src/session/posture.test.ts`
- Modify: `apps/skills-web/src/session/SessionProvider.tsx`
- Modify: `apps/skills-web/src/components/RouteGuards.tsx`
- Modify: `apps/skills-web/src/components/PermissionGate.tsx`
- Modify: `apps/skills-web/src/components/PermissionGate.test.tsx`
- Test: `apps/skills-web/src/session/posture.test.ts`
- Test: `apps/skills-web/src/components/PermissionGate.test.tsx`

- [ ] **Step 1: Write failing posture tests**

Create `apps/skills-web/src/session/posture.test.ts` with:

```ts
import { describe, expect, it } from 'vitest';
import type { RegistrySession } from '../api/types';
import {
  getSessionPosture,
  hasCapability,
  isSessionUnavailable,
} from './posture';

function createSession(actions: string[], allowedPackages: string[] = []): RegistrySession {
  return {
    role: 'operator',
    actor: 'tester',
    authMechanism: 'header',
    allowedActions: actions,
    allowedPackages,
  };
}

describe('session posture', () => {
  it('returns package_filter_required when a scoped governance session has no package focus', () => {
    const session = createSession(['catalog.governance.skill.apply'], ['@platform.skill/*']);

    expect(getSessionPosture(session, { packageName: '' })).toBe('package_filter_required');
  });

  it('allows a scoped package write when the package matches', () => {
    const session = createSession(['catalog.governance.skill.apply'], ['@platform.skill/*']);

    expect(
      hasCapability(session, 'catalog.governance.skill.apply', '@platform.skill/pdf-skill')
    ).toBe(true);
  });

  it('marks an errored session load as unavailable', () => {
    expect(isSessionUnavailable(true, 'network down')).toBe(true);
  });
});
```

- [ ] **Step 2: Run the posture tests to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/session/posture.test.ts
```

Expected: FAIL because `src/session/posture.ts` does not exist yet.

- [ ] **Step 3: Implement the posture module and wire the provider seam**

Create `apps/skills-web/src/session/posture.ts` with:

```ts
import type { RegistrySession } from '../api/types';
import { canPerformAction, canMutateCatalog, describeSessionPosture } from '../api/session';

export type SessionPosture =
  | 'loading'
  | 'unavailable'
  | 'unauthenticated'
  | 'read_only'
  | 'write_enabled'
  | 'package_filter_required';

export function isSessionUnavailable(loading: boolean, error: string | null) {
  return !loading && Boolean(error);
}

export function hasCapability(
  session: RegistrySession | null,
  action: string,
  packageName?: string | null
) {
  return canPerformAction(session, action, packageName);
}

export function getSessionPosture(
  session: RegistrySession | null,
  options: { loading?: boolean; error?: string | null; packageName?: string | null; requiredAction?: string } = {}
): SessionPosture {
  if (options.loading) {
    return 'loading';
  }
  if (isSessionUnavailable(Boolean(options.loading), options.error ?? null)) {
    return 'unavailable';
  }
  if (!session) {
    return 'unauthenticated';
  }
  if (options.requiredAction && options.packageName?.trim() === '' && session.allowedPackages.length > 0) {
    return 'package_filter_required';
  }
  if (options.requiredAction && hasCapability(session, options.requiredAction, options.packageName)) {
    return 'write_enabled';
  }
  return canMutateCatalog(session) ? 'write_enabled' : 'read_only';
}

export function getSessionPostureLabel(
  session: RegistrySession | null,
  options: { loading?: boolean; error?: string | null; packageName?: string | null; requiredAction?: string } = {}
) {
  const posture = getSessionPosture(session, options);
  if (posture === 'loading') return '会话加载中';
  if (posture === 'unavailable') return '会话不可用';
  if (posture === 'unauthenticated') return '未登录';
  if (posture === 'package_filter_required') return '需要先选定包名';
  return describeSessionPosture(session);
}
```

Update `apps/skills-web/src/session/SessionProvider.tsx` so the provider value includes:

```ts
import { getSessionPosture, getSessionPostureLabel, hasCapability } from './posture';

type SessionContextValue = {
  session: AsyncState<RegistrySession | null>;
  refreshSession: () => void;
  getPosture: (options?: { packageName?: string | null; requiredAction?: string }) => string;
  getPostureLabel: (options?: { packageName?: string | null; requiredAction?: string }) => string;
  hasCapability: (action: string, packageName?: string | null) => boolean;
};
```

and:

```tsx
value={{
  session,
  refreshSession: () => setRefreshVersion((current) => current + 1),
  getPosture: (options) =>
    getSessionPosture(session.data, {
      loading: session.loading,
      error: session.error,
      ...options,
    }),
  getPostureLabel: (options) =>
    getSessionPostureLabel(session.data, {
      loading: session.loading,
      error: session.error,
      ...options,
    }),
  hasCapability: (action, packageName) => hasCapability(session.data, action, packageName),
}}
```

Update `RouteGuards.tsx` and `PermissionGate.tsx` to consume `useRegistrySession()` instead of expecting every page to pass `sessionLoading` and `sessionError`.

- [ ] **Step 4: Update the permission gate test and verify GREEN**

Update the main gate test in `apps/skills-web/src/components/PermissionGate.test.tsx` to render through the provider seam:

```tsx
it('renders the fallback when the current session lacks package capability', () => {
  render(
    <PermissionGate
      requiredAction="catalog.governance.skill.apply"
      packageName="@platform.skill/agent-notes"
      session={createSession(['session.read', 'catalog.governance.skill.apply'], ['@platform.skill/code-review'])}
    >
      <div>allowed</div>
    </PermissionGate>
  );

  expect(
    screen.getByText('当前会话没有执行 catalog.governance.skill.apply（@platform.skill/agent-notes）的权限。')
  ).toBeInTheDocument();
});
```

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/session/posture.test.ts src/components/PermissionGate.test.tsx
```

Expected: PASS.

- [ ] **Step 5: Commit the session seam**

```bash
cd /Users/ultraman/workspace/skill_registry
git add apps/skills-web/src/session/posture.ts apps/skills-web/src/session/posture.test.ts apps/skills-web/src/session/SessionProvider.tsx apps/skills-web/src/components/RouteGuards.tsx apps/skills-web/src/components/PermissionGate.tsx apps/skills-web/src/components/PermissionGate.test.tsx
git commit -m "refactor: deepen skills-web session seam"
```

## Task 2: Extract the Authoring Draft Module

**Files:**
- Create: `apps/skills-web/src/features/packaging/useAuthoringDraft.ts`
- Create: `apps/skills-web/src/features/packaging/useAuthoringDraft.test.ts`
- Modify: `apps/skills-web/src/pages/ImportPage.tsx`
- Test: `apps/skills-web/src/features/packaging/useAuthoringDraft.test.ts`

- [ ] **Step 1: Write failing tests for the authoring draft rules**

Create `apps/skills-web/src/features/packaging/useAuthoringDraft.test.ts` with:

```ts
import { describe, expect, it } from 'vitest';
import {
  createDefaultAuthoringDraft,
  draftFilesToRecord,
  parseAuthoringJson,
  validateAuthoringPath,
} from './useAuthoringDraft';

describe('authoring draft', () => {
  it('starts the command template with a locked commands/run.txt file', () => {
    const draft = createDefaultAuthoringDraft('command');

    expect(draft.files).toEqual([
      {
        path: 'commands/run.txt',
        content: 'echo authored\n',
        locked: true,
      },
    ]);
  });

  it('rejects SKILL.md and parent-path entries', () => {
    const files = createDefaultAuthoringDraft('command').files;

    expect(validateAuthoringPath('SKILL.md', files)).toBe('根级 SKILL.md 由服务端生成，不能手动添加。');
    expect(validateAuthoringPath('../bad.txt', files)).toBe('文件路径不能包含 ..。');
  });

  it('parses raw json into a file record', () => {
    expect(parseAuthoringJson('{\n  "docs/README.md": "hello\\n"\n}')).toEqual({
      'docs/README.md': 'hello\n',
    });
  });

  it('serializes files back into the request record shape', () => {
    const draft = createDefaultAuthoringDraft('workflow');

    expect(draftFilesToRecord(draft.files)).toEqual({
      'workflows/main.md': '1. First step\n2. Second step\n',
    });
  });
});
```

- [ ] **Step 2: Run the authoring draft tests to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/features/packaging/useAuthoringDraft.test.ts
```

Expected: FAIL because the module does not exist yet.

- [ ] **Step 3: Implement the authoring draft module**

Create `apps/skills-web/src/features/packaging/useAuthoringDraft.ts` with:

```ts
export type AuthoringTemplateId = 'prompt' | 'command' | 'workflow';

export type AuthoringFile = {
  path: string;
  content: string;
  locked: boolean;
};

export type AuthoringDraft = {
  template: AuthoringTemplateId;
  files: AuthoringFile[];
};

const AUTHORING_TEMPLATES: Record<AuthoringTemplateId, AuthoringFile[]> = {
  prompt: [{ path: 'prompts/main.md', content: 'Write the core prompt here.\n', locked: true }],
  command: [{ path: 'commands/run.txt', content: 'echo authored\n', locked: true }],
  workflow: [{ path: 'workflows/main.md', content: '1. First step\n2. Second step\n', locked: true }],
};

export function createDefaultAuthoringDraft(template: AuthoringTemplateId): AuthoringDraft {
  return {
    template,
    files: AUTHORING_TEMPLATES[template].map((file) => ({ ...file })),
  };
}

export function draftFilesToRecord(files: AuthoringFile[]) {
  return Object.fromEntries(files.map((file) => [file.path, file.content]));
}

export function parseAuthoringJson(raw: string) {
  const parsed = JSON.parse(raw) as Record<string, unknown>;
  return Object.fromEntries(
    Object.entries(parsed).map(([path, value]) => {
      if (typeof value !== 'string') {
        throw new Error(`Inline file ${path} 的内容必须是字符串。`);
      }
      return [path, value];
    })
  );
}

export function validateAuthoringPath(path: string, existing: AuthoringFile[]) {
  const trimmed = path.trim();
  if (trimmed === '') return '新文件路径不能为空。';
  if (trimmed.startsWith('/')) return '文件路径必须是相对路径。';
  if (trimmed === 'SKILL.md') return '根级 SKILL.md 由服务端生成，不能手动添加。';
  if (trimmed.split('/').some((segment) => segment === '..')) return '文件路径不能包含 ..。';
  if (existing.some((file) => file.path === trimmed)) return '文件路径不能重复。';
  return null;
}
```

Leave the existing page behavior intact for now; only make the helper module pass its own tests.

- [ ] **Step 4: Migrate `ImportPage` to import the authoring types/helpers**

At the top of `apps/skills-web/src/pages/ImportPage.tsx`, replace the local authoring type/helper block with:

```ts
import {
  createDefaultAuthoringDraft,
  draftFilesToRecord,
  parseAuthoringJson,
  type AuthoringFile,
  type AuthoringTemplateId,
  validateAuthoringPath,
} from '../features/packaging/useAuthoringDraft';
```

and update the initialization logic to:

```ts
const [authoringTemplate, setAuthoringTemplate] = useState<AuthoringTemplateId>('command');
const [authoringFiles, setAuthoringFiles] = useState<AuthoringFile[]>(
  () => createDefaultAuthoringDraft('command').files
);
```

Use `draftFilesToRecord(authoringFiles)` and `parseAuthoringJson(rawInlineFiles)` where the page currently uses its local helpers.

- [ ] **Step 5: Run the helper tests and commit**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/features/packaging/useAuthoringDraft.test.ts src/pages/ImportPage.test.tsx
```

Expected: PASS for the helper tests; `ImportPage` tests should remain green if the page still behaves the same.

Commit:

```bash
cd /Users/ultraman/workspace/skill_registry
git add apps/skills-web/src/features/packaging/useAuthoringDraft.ts apps/skills-web/src/features/packaging/useAuthoringDraft.test.ts apps/skills-web/src/pages/ImportPage.tsx
git commit -m "refactor: extract skills-web authoring draft module"
```

## Task 3: Extract the Packaging Workflow Module and Thin `ImportPage`

**Files:**
- Create: `apps/skills-web/src/features/packaging/usePackagingWorkflow.ts`
- Create: `apps/skills-web/src/features/packaging/usePackagingWorkflow.test.ts`
- Modify: `apps/skills-web/src/pages/ImportPage.tsx`
- Modify: `apps/skills-web/src/pages/ImportPage.test.tsx`
- Test: `apps/skills-web/src/features/packaging/usePackagingWorkflow.test.ts`
- Test: `apps/skills-web/src/pages/ImportPage.test.tsx`

- [ ] **Step 1: Write failing workflow tests**

Create `apps/skills-web/src/features/packaging/usePackagingWorkflow.test.ts` with:

```ts
import { describe, expect, it, vi } from 'vitest';
import { buildReleaseHandoffPath } from '../../lib/releaseHandoff';
import {
  derivePackagingViewState,
  toMutationResult,
} from './usePackagingWorkflow';

describe('packaging workflow view state', () => {
  it('moves to validated after a successful validate result', () => {
    expect(
      derivePackagingViewState({
        prepareResult: {
          workspacePath: '/tmp/workspace',
          packageName: '@company.skill/zoom-out',
          version: '1.2.0',
          warnings: [],
          warningDetails: [],
          errors: [],
        },
        validateResult: {
          ok: true,
          workspacePath: '/tmp/workspace',
          packageName: '@company.skill/zoom-out',
          version: '1.2.0',
          manifestDigest: 'sha256:abc',
          components: ['docs'],
          warnings: [],
          warningDetails: [],
          errors: [],
        },
        packResult: null,
        publishRegistry: 'https://npm.internal.example.com',
      }).currentStage
    ).toBe('validated');
  });

  it('builds a handoff path once pack completes', () => {
    const state = derivePackagingViewState({
      prepareResult: null,
      validateResult: null,
      packResult: {
        workspacePath: '/tmp/workspace',
        bundlePath: '/tmp/workspace/bundle.tgz',
        packageName: '@company.skill/zoom-out',
        version: '1.2.0',
        manifestDigest: 'sha256:abc',
        components: ['docs'],
        warnings: [],
        warningDetails: [],
        errors: [],
      },
      publishRegistry: 'https://npm.internal.example.com',
    });

    expect(state.releaseHandoffPath).toBe(
      buildReleaseHandoffPath({
        packageName: '@company.skill/zoom-out',
        version: '1.2.0',
        publishRegistry: 'https://npm.internal.example.com',
        manifestDigest: 'sha256:abc',
        components: ['docs'],
        workspacePath: '/tmp/workspace',
        bundlePath: '/tmp/workspace/bundle.tgz',
      })
    );
  });

  it('maps a thrown 403 into a forbidden mutation result', () => {
    const result = toMutationResult(
      'pack',
      {
        kind: 'server',
        message: 'forbidden',
        requestId: 'req_123',
      },
      'forbidden',
      { packageName: '@company.skill/zoom-out', version: '1.2.0', actor: 'tester' }
    );

    expect(result).toEqual({
      action: 'pack',
      outcome: 'forbidden',
      message: 'forbidden',
      requestId: 'req_123',
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      actor: 'tester',
    });
  });
});
```

- [ ] **Step 2: Run the workflow tests to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/features/packaging/usePackagingWorkflow.test.ts
```

Expected: FAIL because `usePackagingWorkflow.ts` does not exist yet.

- [ ] **Step 3: Implement the workflow module**

Create `apps/skills-web/src/features/packaging/usePackagingWorkflow.ts` with:

```ts
import { inferMutationFailureOutcome, writeMutationResultSearchParams } from '../../components/MutationResultPanel';
import type {
  PackagingPackResponse,
  PackagingPrepareResponse,
  PackagingValidateResponse,
  RegistryRequestError,
} from '../../api/types';
import { buildReleaseHandoffPath } from '../../lib/releaseHandoff';

type WorkflowInputs = {
  prepareResult: PackagingPrepareResponse | null;
  validateResult: PackagingValidateResponse | null;
  packResult: PackagingPackResponse | null;
  publishRegistry: string;
};

export function derivePackagingViewState({
  prepareResult,
  validateResult,
  packResult,
  publishRegistry,
}: WorkflowInputs) {
  const currentStage = packResult ? 'pack_complete' : validateResult ? 'validated' : prepareResult ? 'prepared' : 'draft';
  const releaseHandoffPath = packResult
    ? buildReleaseHandoffPath({
        packageName: packResult.packageName,
        version: packResult.version,
        publishRegistry: publishRegistry.trim() || undefined,
        manifestDigest: packResult.manifestDigest,
        components: packResult.components,
        workspacePath: packResult.workspacePath,
        bundlePath: packResult.bundlePath,
      })
    : null;

  return {
    currentStage,
    activeStepIndex: packResult ? 3 : validateResult ? 2 : prepareResult ? 1 : 0,
    canValidate: Boolean(prepareResult?.workspacePath),
    canPack: Boolean(validateResult?.ok),
    canCleanup: Boolean(prepareResult?.workspacePath),
    releaseHandoffPath,
  };
}

export function toMutationResult(
  action: string,
  error: RegistryRequestError,
  outcome: ReturnType<typeof inferMutationFailureOutcome>,
  defaults: { packageName?: string | null; version?: string | null; actor?: string | null }
) {
  return {
    action,
    outcome,
    message: error.message,
    requestId: error.requestId,
    packageName: defaults.packageName ?? null,
    version: defaults.version ?? null,
    actor: defaults.actor ?? null,
  };
}

export function persistPackagingMutationResult(searchParams: URLSearchParams, result: Parameters<typeof writeMutationResultSearchParams>[1]) {
  return writeMutationResultSearchParams(searchParams, result);
}
```

- [ ] **Step 4: Refactor `ImportPage` to consume the workflow seam**

In `apps/skills-web/src/pages/ImportPage.tsx`:

- import `useRegistrySession()` and stop calling `useAsyncData(() => getSession(), [])`
- replace inline stage/handoff booleans with:

```ts
const { session, getPostureLabel, hasCapability } = useRegistrySession();
const workflow = derivePackagingViewState({
  prepareResult,
  validateResult,
  packResult,
  publishRegistry,
});
const packagingWriteEnabled = hasCapability('packaging.pack', normalizedPackageName);
```

- replace the local `persistMutationResult()` body with:

```ts
setSearchParams(
  persistPackagingMutationResult(searchParams, {
    ...result,
    actor: session.data?.actor ?? null,
  }),
  { replace: true }
);
```

- use `workflow.currentStage`, `workflow.canValidate`, `workflow.canPack`, `workflow.canCleanup`, and `workflow.releaseHandoffPath` in the render tree.

- [ ] **Step 5: Run the workflow and page tests, then commit**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/features/packaging/usePackagingWorkflow.test.ts src/pages/ImportPage.test.tsx
```

Expected: PASS.

Commit:

```bash
cd /Users/ultraman/workspace/skill_registry
git add apps/skills-web/src/features/packaging/usePackagingWorkflow.ts apps/skills-web/src/features/packaging/usePackagingWorkflow.test.ts apps/skills-web/src/pages/ImportPage.tsx apps/skills-web/src/pages/ImportPage.test.tsx
git commit -m "refactor: move packaging workflow behind a seam"
```

## Task 4: Share Catalog Mutation Behavior Across Detail Pages

**Files:**
- Create: `apps/skills-web/src/features/catalog-mutations/useCatalogMutation.ts`
- Create: `apps/skills-web/src/features/catalog-mutations/useCatalogMutation.test.ts`
- Modify: `apps/skills-web/src/pages/SkillDetailPage.tsx`
- Modify: `apps/skills-web/src/pages/VersionDetailPage.tsx`
- Test: `apps/skills-web/src/features/catalog-mutations/useCatalogMutation.test.ts`

- [ ] **Step 1: Write failing tests for the shared mutation seam**

Create `apps/skills-web/src/features/catalog-mutations/useCatalogMutation.test.ts` with:

```ts
import { describe, expect, it } from 'vitest';
import {
  buildCatalogMutationResult,
  createMutationState,
} from './useCatalogMutation';

describe('catalog mutation seam', () => {
  it('creates a clean idle state', () => {
    expect(createMutationState()).toEqual({
      loading: false,
      error: null,
      success: null,
    });
  });

  it('builds a success mutation result for release record writes', () => {
    expect(
      buildCatalogMutationResult(
        'release-record',
        'succeeded',
        '已创建新的 release record。',
        { packageName: '@company.skill/zoom-out', version: '1.2.0', actor: 'tester' }
      )
    ).toEqual({
      action: 'release-record',
      outcome: 'succeeded',
      message: '已创建新的 release record。',
      requestId: null,
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      actor: 'tester',
    });
  });
});
```

- [ ] **Step 2: Run the mutation tests to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/features/catalog-mutations/useCatalogMutation.test.ts
```

Expected: FAIL because the module does not exist yet.

- [ ] **Step 3: Implement the shared mutation module**

Create `apps/skills-web/src/features/catalog-mutations/useCatalogMutation.ts` with:

```ts
import { writeMutationResultSearchParams } from '../../components/MutationResultPanel';

export type CatalogMutationState = {
  loading: false | 'release' | 'governance' | 'retire';
  error: string | null;
  success: string | null;
};

export function createMutationState(): CatalogMutationState {
  return {
    loading: false,
    error: null,
    success: null,
  };
}

export function buildCatalogMutationResult(
  action: string,
  outcome: 'succeeded' | 'failed' | 'forbidden' | 'unauthorized',
  message: string,
  defaults: { packageName?: string | null; version?: string | null; actor?: string | null; requestId?: string | null }
) {
  return {
    action,
    outcome,
    message,
    requestId: defaults.requestId ?? null,
    packageName: defaults.packageName ?? null,
    version: defaults.version ?? null,
    actor: defaults.actor ?? null,
  };
}

export function persistCatalogMutation(searchParams: URLSearchParams, result: Parameters<typeof writeMutationResultSearchParams>[1]) {
  return writeMutationResultSearchParams(searchParams, result);
}
```

- [ ] **Step 4: Refactor both detail pages to use the mutation seam and session seam**

In `apps/skills-web/src/pages/SkillDetailPage.tsx`:

- replace `const session = useAsyncData(() => getSession(), []);` with:

```ts
const { session, hasCapability } = useRegistrySession();
```

- replace the local `persistMutationResult()` with:

```ts
setSearchParams(
  persistCatalogMutation(
    searchParams,
    buildCatalogMutationResult(result.action, result.outcome, result.message, {
      requestId: result.requestId ?? null,
      packageName: decodedPackage,
      version: null,
      actor: session.data?.actor ?? null,
    })
  ),
  { replace: true }
);
```

In `apps/skills-web/src/pages/VersionDetailPage.tsx`, do the same for both release-record and governance submissions, and replace:

```ts
const releaseWriteEnabled = canPerformAction(session.data, 'catalog.release_record.create', decodedPackage);
```

with:

```ts
const releaseWriteEnabled = hasCapability('catalog.release_record.create', decodedPackage);
```

- [ ] **Step 5: Run the mutation tests and commit**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/features/catalog-mutations/useCatalogMutation.test.ts src/components/MutationResultPanel.test.tsx
```

Expected: PASS.

Commit:

```bash
cd /Users/ultraman/workspace/skill_registry
git add apps/skills-web/src/features/catalog-mutations/useCatalogMutation.ts apps/skills-web/src/features/catalog-mutations/useCatalogMutation.test.ts apps/skills-web/src/pages/SkillDetailPage.tsx apps/skills-web/src/pages/VersionDetailPage.tsx
git commit -m "refactor: share catalog mutation workflow"
```

## Task 5: Split the Internal Adapters and Mock State Store, Then Run Regression Checks

**Files:**
- Create: `apps/skills-web/src/api/sessionClient.ts`
- Create: `apps/skills-web/src/api/catalogReadClient.ts`
- Create: `apps/skills-web/src/api/packagingClient.ts`
- Create: `apps/skills-web/src/api/catalogMutationClient.ts`
- Create: `apps/skills-web/src/api/mockState.ts`
- Modify: `apps/skills-web/src/api/client.ts`
- Modify: `apps/skills-web/src/api/mockClient.ts`
- Modify: `apps/skills-web/src/api/client.test.ts`
- Modify: `apps/skills-web/src/api/session.test.ts`
- Modify: `apps/skills-web/src/pages/GovernancePage.tsx`
- Modify: `apps/skills-web/src/pages/AuthoringFlowPage.tsx`
- Test: `apps/skills-web/src/api/client.test.ts`
- Test: `apps/skills-web/src/api/session.test.ts`
- Test: `apps/skills-web/src/pages/ImportPage.test.tsx`
- Test: `apps/skills-web/src/features/catalog-mutations/useCatalogMutation.test.ts`
- Test: `apps/skills-web/src/features/packaging/usePackagingWorkflow.test.ts`

- [ ] **Step 1: Write a failing adapter test that expects the facade to keep working after the split**

Add to `apps/skills-web/src/api/client.test.ts`:

```ts
it('normalizes packaging review responses through the facade exports', () => {
  expect(
    normalizePackagingReview({
      warnings: null,
      warningDetails: null,
      errors: null,
    })
  ).toEqual({
    warnings: [],
    warningDetails: [],
    errors: [],
  });
});
```

Add to `apps/skills-web/src/api/session.test.ts`:

```ts
it('continues to resolve getSession through the facade seam', async () => {
  const clientModule = await import('./client');

  await expect(clientModule.getSession()).resolves.toBeNull();
});
```

- [ ] **Step 2: Run the adapter tests to verify RED after the file move starts**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/api/client.test.ts src/api/session.test.ts
```

Expected: FAIL once the files are moved but the facade is not fully rewired.

- [ ] **Step 3: Split the adapters and the mock state store**

Create `apps/skills-web/src/api/mockState.ts` with:

```ts
import type { AuditEventRecord, GovernanceRecord, SkillSummary, SkillVersionSummary } from './types';

export type MockState = {
  skills: SkillSummary[];
  versions: Record<string, SkillVersionSummary[]>;
  governance: GovernanceRecord[];
  auditEvents: AuditEventRecord[];
};

export function createMockState(): MockState {
  return {
    skills: [...initialSkills],
    versions: structuredClone(initialVersions),
    governance: [...initialGovernance],
    auditEvents: [...initialAuditEvents],
  };
}

export function recordAudit(state: MockState, event: Omit<AuditEventRecord, 'id' | 'requestId' | 'traceId' | 'createdAt'>) {
  state.auditEvents.unshift({
    id: `aud_${String(state.auditEvents.length + 1).padStart(2, '0')}`,
    requestId: `req_${Date.now()}`,
    traceId: `trace_${Date.now()}`,
    createdAt: new Date().toISOString(),
    ...event,
  });
}
```

Create the split adapter files so `client.ts` can become a facade:

```ts
// sessionClient.ts
export const sessionClient = {
  getConfig() {
    return registryClientConfig;
  },
  getSession() {
    return requestJSON<RegistrySession | null>('/api/v1/session').catch((error: unknown) => {
      if (error instanceof RegistryHttpError && error.status === 401) return null;
      throw error;
    });
  },
  loginSession(username: string, password: string) {
    return requestJSON<RegistrySession>('/api/v1/session/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });
  },
  logoutSession() {
    return requestJSON<{ ok: boolean }>('/api/v1/session/logout', {
      method: 'POST',
      body: JSON.stringify({}),
    }).then(() => undefined);
  },
};
```

and in `client.ts` re-export:

```ts
export const apiClient: RegistryApiClient =
  registryClientConfig.mode === 'real'
    ? {
        ...sessionClient,
        ...catalogReadClient,
        ...packagingClient,
        ...catalogMutationClient,
      }
    : mockClient;
```

Update `mockClient.ts` to hold:

```ts
const state = createMockState();
```

and replace direct array/object mutation with helper calls against `state`.

- [ ] **Step 4: Migrate the remaining session-bypassing pages and run the regression suite**

Update `apps/skills-web/src/pages/GovernancePage.tsx` and `apps/skills-web/src/pages/AuthoringFlowPage.tsx`:

```ts
const { session, getPostureLabel, hasCapability } = useRegistrySession();
```

Remove their direct `getSession()` imports and replace their posture badges with `getPostureLabel(...)`.

Run:

```bash
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm test -- src/api/client.test.ts src/api/session.test.ts src/pages/ImportPage.test.tsx src/features/packaging/usePackagingWorkflow.test.ts src/features/catalog-mutations/useCatalogMutation.test.ts
cd /Users/ultraman/workspace/skill_registry/apps/skills-web && npm run type-check
```

Expected: PASS.

- [ ] **Step 5: Commit the adapter split and final verification**

```bash
cd /Users/ultraman/workspace/skill_registry
git add apps/skills-web/src/api/sessionClient.ts apps/skills-web/src/api/catalogReadClient.ts apps/skills-web/src/api/packagingClient.ts apps/skills-web/src/api/catalogMutationClient.ts apps/skills-web/src/api/mockState.ts apps/skills-web/src/api/client.ts apps/skills-web/src/api/mockClient.ts apps/skills-web/src/api/client.test.ts apps/skills-web/src/api/session.test.ts apps/skills-web/src/pages/GovernancePage.tsx apps/skills-web/src/pages/AuthoringFlowPage.tsx
git commit -m "refactor: split skills-web adapters by domain"
```

## Spec Coverage Check

- Session seam deepening: covered by Task 1 and the remaining page migration in Task 5
- Authoring draft deepening: covered by Task 2
- Packaging workflow deepening: covered by Task 3
- Catalog mutation deepening: covered by Task 4
- Adapter/mock state store deepening: covered by Task 5

No spec section is left without a corresponding implementation task.

## Placeholder Scan

- No unresolved placeholder markers remain
- Every code-changing step includes concrete file paths and code snippets
- Every verification step includes exact commands and expected outcomes

## Type Consistency Check

- Session seam uses `hasCapability`, `getSessionPosture`, and `getSessionPostureLabel` consistently
- Packaging seam uses `derivePackagingViewState`, `toMutationResult`, and `persistPackagingMutationResult` consistently
- Catalog mutation seam uses `createMutationState`, `buildCatalogMutationResult`, and `persistCatalogMutation` consistently
