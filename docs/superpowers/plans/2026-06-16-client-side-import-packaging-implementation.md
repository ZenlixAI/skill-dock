# Client-Side Import Packaging Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove server-side git import support, move web external import onto client-side archive upload, and delete the obsolete tests/docs that still describe the old contract.

**Architecture:** First tighten the backend packaging contract so `prepare` accepts only `upload_archive` and `platform_authoring`, with git requests rejected by a stable 422 error. Next change the web import page to treat external sources as browser-side acquisition plus archive upload instead of a server-side git source mode. Finish by removing stale tests and documentation that still describe successful server-side git import.

**Tech Stack:** Go 1.26, Gin, React 19, TypeScript 5, Vitest, existing packaging HTTP API

---

## File Structure

- Create: `apps/skills-web/src/features/packaging/externalImport.ts`
- Modify: `internal/packaging/infra/workspace_source.go`
- Modify: `internal/backend/httpserver/server_test.go`
- Modify: `internal/backend/httpserver/module_test_helpers_test.go`
- Modify: `apps/skills-web/package.json`
- Modify: `apps/skills-web/package-lock.json`
- Modify: `apps/skills-web/src/api/types.ts`
- Modify: `apps/skills-web/src/api/client.test.ts`
- Modify: `apps/skills-web/src/pages/ImportPage.tsx`
- Modify: `apps/skills-web/src/pages/ImportPage.test.tsx`
- Modify: `docs/batch-import-skills.zh-CN.md`

### Task 1: Remove backend git-source preparation and reject the old contract

**Files:**
- Modify: `internal/packaging/infra/workspace_source.go`
- Modify: `internal/backend/httpserver/server_test.go`
- Modify: `internal/backend/httpserver/module_test_helpers_test.go`

- [ ] **Step 1: Write the failing backend tests for the rejected git contract**

```go
// internal/backend/httpserver/server_test.go
func TestSkillPackagingRejectsGitSource(t *testing.T) {
	server := newTestServer()

	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"source":{"type":"git","location":"https://github.com/example/skills?path=skills/pdf"},
		"target":{"packageName":"@platform.skill/rejected-git","version":"0.5.0"},
		"metadata":{"displayName":"Rejected Git","description":"Old contract","sourceType":"imported"}
	}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
	}

	assertErrorDetailContains(t, recorder, "sourceType", "git")
	assertErrorDetailContains(t, recorder, "suggestedAction", "upload_archive")
}
```

```go
// internal/backend/httpserver/server_test.go
func TestSkillPackagingStillSupportsUploadArchiveSource(t *testing.T) {
	server := newTestServer()
	fixtureDir := createPackagingFixture(t)

	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"source":`+uploadArchiveSourceJSON(t, fixtureDir)+`,
		"target":{"packageName":"@platform.skill/upload-fixture","version":"0.4.0"},
		"metadata":{"displayName":"Upload Fixture","description":"Uploaded archive","sourceType":"imported"}
	}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
}
```

- [ ] **Step 2: Run the focused backend tests to verify the new rejection test fails first**

Run: `go test ./internal/backend/httpserver -run 'TestSkillPackagingRejectsGitSource|TestSkillPackagingStillSupportsUploadArchiveSource'`

Expected: FAIL because `source.type=git` is still accepted by `workspace_source.go`.

- [ ] **Step 3: Remove the git preparation path and return a stable 422**

```go
// internal/packaging/infra/workspace_source.go
switch input.Source.Type {
case "local":
	return "", catalog.NewError(422, "unsupported_source_type", "source.type=local has been removed. Upload the source archive content instead of sending a local filesystem path.", map[string]any{"sourceType": input.Source.Type})
case "git":
	return "", catalog.NewError(422, "unsupported_source_type", "source.type=git has been removed. Package the source on the client side and upload it as source.type=upload_archive.", map[string]any{
		"sourceType":      input.Source.Type,
		"suggestedAction": "Use source.type=upload_archive instead.",
	})
case "upload_archive":
	if err := extractUploadedArchive(input.Source.ArchiveBase64, workspacePath); err != nil {
		return "", err
	}
case "platform_authoring":
	if err := materializeInlineFiles(workspacePath, input.Source.Files); err != nil {
		return "", err
	}
default:
	return "", catalog.NewError(422, "invalid_request", "source.type is invalid.", map[string]any{"sourceType": input.Source.Type})
}
```

```go
// internal/packaging/infra/workspace_source.go
// Delete these obsolete exports and helpers entirely:
// - RunGitCommand
// - prepareGitSource
// - normalizeGitLocation
// - isExistingDirectory
// - runGitCommand
// - gitCommandTimeout
// - DefaultGitCloneTimeout
```

- [ ] **Step 4: Delete the old successful git-path tests and helpers**

```go
// internal/backend/httpserver/server_test.go
// Delete:
// - TestSkillPackagingSupportsLocalGitSourcePath
// - TestSkillPackagingSupportsRemoteGitSourceRefAndPath

// internal/backend/httpserver/server_test.go
// Delete helpers only used by those tests:
// - createPackagingGitRemote
// - commitPackagingGitFixture
```

- [ ] **Step 5: Run the backend verification**

Run: `go test ./internal/backend/httpserver ./internal/packaging/...`

Expected: PASS with `source.type=git` rejected and `upload_archive` / `platform_authoring` still green.

- [ ] **Step 6: Commit**

```bash
git add internal/packaging/infra/workspace_source.go internal/backend/httpserver/server_test.go internal/backend/httpserver/module_test_helpers_test.go
git commit -m "refactor: remove server-side git packaging source"
```

### Task 2: Move web external import onto client-side archive upload

**Files:**
- Create: `apps/skills-web/src/features/packaging/externalImport.ts`
- Modify: `apps/skills-web/package.json`
- Modify: `apps/skills-web/package-lock.json`
- Modify: `apps/skills-web/src/api/types.ts`
- Modify: `apps/skills-web/src/pages/ImportPage.tsx`
- Modify: `apps/skills-web/src/pages/ImportPage.test.tsx`

- [ ] **Step 1: Write the failing web tests for the new prepare payload**

```tsx
// apps/skills-web/src/pages/ImportPage.test.tsx
it('builds an upload_archive payload for external import', async () => {
  vi.mocked(client.preparePackaging).mockResolvedValue({
    workspacePath: '/tmp/skill-packaging/workspace',
    packageName: '@company.skill/zoom-out',
    version: '1.2.0',
    warnings: [],
    warningDetails: [],
    errors: [],
  });
  vi.mocked(client.validatePackaging).mockResolvedValue({
    ok: true,
    workspacePath: '/tmp/skill-packaging/workspace',
    packageName: '@company.skill/zoom-out',
    version: '1.2.0',
    manifestDigest: 'sha256:manifest',
    components: ['commands'],
    warnings: [],
    warningDetails: [],
    errors: [],
  });
  vi.spyOn(globalThis, 'fetch').mockResolvedValue(
    new Response('echo external\n', { status: 200 })
  );

  renderImportPage();

  fireEvent.click(await screen.findByRole('button', { name: '检查并生成发布预览' }));

  await waitFor(() =>
    expect(client.preparePackaging).toHaveBeenCalledWith(
      expect.objectContaining({
        source: expect.objectContaining({
          type: 'upload_archive',
          archiveBase64: expect.any(String),
        }),
      })
    )
  );
});
```

```tsx
// apps/skills-web/src/pages/ImportPage.test.tsx
it('does not call preparePackaging when external import acquisition fails', async () => {
  vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('network down'));

  renderImportPage();

  fireEvent.click(await screen.findByRole('button', { name: '检查并生成发布预览' }));

  expect(await screen.findByText(/无法获取外部 skill 源/i)).toBeInTheDocument();
  expect(client.preparePackaging).not.toHaveBeenCalled();
});
```

- [ ] **Step 2: Run the focused web tests to verify they fail**

Run: `npm test -- --run src/pages/ImportPage.test.tsx`

Workdir: `apps/skills-web`

Expected: FAIL because the page still sends `source.type=git` and has no browser-side packaging flow.

- [ ] **Step 3: Add a browser-side external import packager**

```ts
// apps/skills-web/src/features/packaging/externalImport.ts
const IGNORED_SEGMENTS = new Set(['.git', '.dist']);
const IGNORED_FILE = '.skill-packaging-import-report.json';

export interface ExternalImportArchive {
  archiveBase64: string;
}

export async function buildExternalImportArchive(
  location: string,
  ref?: string
): Promise<ExternalImportArchive> {
  const fileMap = await fetchExternalSkillFiles(location, ref);
  const filteredFiles = filterPackagingFiles(fileMap);
  const tarBytes = createTarArchive(filteredFiles);
  const gzipBytes = gzipTarArchive(tarBytes);
  return { archiveBase64: bytesToBase64(gzipBytes) };
}
```

```ts
// apps/skills-web/src/features/packaging/externalImport.ts
// Implement the minimum browser-side flow:
// - parse GitHub-style URL + optional ?path=...
// - fetch the selected skill subtree
// - filter .git / .dist / import report entries
// - create a tar archive with relative paths
// - gzip the tar bytes
// - return base64 for preparePackaging
```

- [ ] **Step 4: Switch the import page off server-facing git mode**

```tsx
// apps/skills-web/src/api/types.ts
export type PackagingSourceType = 'upload_archive' | 'platform_authoring';
```

```tsx
// apps/skills-web/src/pages/ImportPage.tsx
const [sourceType, setSourceType] = useState<PackagingSourceType>('upload_archive');
const [location, setLocation] = useState('https://github.com/mattpocock/skills?path=skills/engineering/zoom-out');
```

```tsx
// apps/skills-web/src/pages/ImportPage.tsx
async function buildPrepareRequest(): Promise<PackagingPrepareRequest> {
  if (sourceType === 'platform_authoring') {
    return {
      source: { type: 'platform_authoring', files: platformFiles },
      target: { packageName: packageName.trim(), version: version.trim() },
      metadata: { displayName: displayName.trim(), description: description.trim(), sourceType: metadataSourceType },
      options: { publishRegistry: publishRegistry.trim() || undefined, strict: strictMode },
    };
  }

  const archive = await buildExternalImportArchive(location.trim(), sourceRef.trim() || undefined);
  return {
    source: {
      type: 'upload_archive',
      archiveBase64: archive.archiveBase64,
    },
    target: { packageName: packageName.trim(), version: version.trim() },
    metadata: { displayName: displayName.trim(), description: description.trim(), sourceType: metadataSourceType },
    options: { publishRegistry: publishRegistry.trim() || undefined, strict: strictMode },
  };
}
```

```tsx
// apps/skills-web/src/pages/ImportPage.tsx
// Update copy so the non-authoring path explains:
// - browser-side download
// - browser-side packaging
// - upload to catalog
```

- [ ] **Step 5: Run the focused web verification**

Run: `npm test -- --run src/pages/ImportPage.test.tsx`

Workdir: `apps/skills-web`

Expected: PASS with external import now producing `upload_archive` and acquisition failures stopping before `preparePackaging`.

- [ ] **Step 6: Commit**

```bash
git add apps/skills-web/package.json apps/skills-web/package-lock.json apps/skills-web/src/api/types.ts apps/skills-web/src/features/packaging/externalImport.ts apps/skills-web/src/pages/ImportPage.tsx apps/skills-web/src/pages/ImportPage.test.tsx
git commit -m "feat: package external imports on the client side"
```

### Task 3: Remove stale docs/tests and verify the end-to-end contract

**Files:**
- Modify: `apps/skills-web/src/api/client.test.ts`
- Modify: `docs/batch-import-skills.zh-CN.md`

- [ ] **Step 1: Write the failing contract-cleanup tests**

```ts
// apps/skills-web/src/api/client.test.ts
await expect(
  clientModule.preparePackaging({
    source: {
      type: 'upload_archive',
      archiveBase64: 'YXJjaGl2ZQ==',
    },
    target: {
      packageName: '@platform.skill/code-review',
      version: '1.0.0',
    },
    metadata: {
      displayName: 'Code Review',
      description: 'Checklist pack',
      sourceType: 'platform_created',
    },
    options: {
      strict: false,
    },
  })
).resolves.toEqual(expect.objectContaining({
  workspacePath: '/tmp/mock-workspace',
}));
```

- [ ] **Step 2: Run the focused contract test to verify it fails if stale git payloads remain**

Run: `npm test -- --run src/api/client.test.ts`

Workdir: `apps/skills-web`

Expected: FAIL while the facade/mock tests still hard-code `source.type=git`.

- [ ] **Step 3: Update stale docs and facade tests**

```md
<!-- docs/batch-import-skills.zh-CN.md -->
- 删除“服务端自行拉源码”与“source.type=git”作为正式路径的表述
- 明确保留路径为：客户端目录打包成 `upload_archive` 后上传
```

```ts
// apps/skills-web/src/api/client.test.ts
source: {
  type: 'upload_archive',
  archiveBase64: 'YXJjaGl2ZQ==',
},
```

- [ ] **Step 4: Run the final verification**

Run: `npm test -- --run src/api/client.test.ts src/pages/ImportPage.test.tsx`

Workdir: `apps/skills-web`

Expected: PASS

Run: `go test ./...`

Workdir: repository root

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add apps/skills-web/src/api/client.test.ts docs/batch-import-skills.zh-CN.md
git commit -m "docs: remove stale git import contract references"
```

## Self-Review

- Spec coverage: backend rejection, web client-side archive upload, stale test cleanup, and doc cleanup each have an explicit task above.
- Placeholder scan: no `TODO`, `TBD`, or implicit “write tests later” steps remain.
- Type consistency: the retained prepare contract is `upload_archive | platform_authoring` across backend, web types, and tests.
