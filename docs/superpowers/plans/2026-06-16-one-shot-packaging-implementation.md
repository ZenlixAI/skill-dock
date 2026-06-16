# One-Shot Packaging Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the public workspace-session packaging API with stateless `import-review` and stateless `publish`, then migrate backend, web, CLI, mocks, and tests off `workspacePath` and `bundlePath`.

**Architecture:** First replace the packaging domain contract and HTTP routes so the backend exposes only one-shot review and one-shot publish. Next collapse backend packaging ownership from session-style workspace lifecycle into request-local orchestration. Finish by migrating the web UI, mock surface, CLI, and docs to the new stateless contract and deleting the old workspace-session artifacts.

**Tech Stack:** Go 1.26, Gin, React 19, TypeScript 5, Vitest, existing packaging bundle rules, existing registry publish + release-record recovery

---

## File Structure

- Create: `docs/superpowers/plans/2026-06-16-one-shot-packaging-implementation.md`
- Create: `internal/packaging/application/import_review_service.go`
- Create: `internal/packaging/application/import_review_service_test.go`
- Create: `internal/packaging/http/one_shot_workflow.go`
- Create: `internal/packaging/http/one_shot_workflow_test.go`
- Modify: `internal/packaging/domain/types.go`
- Modify: `internal/packaging/application/service.go`
- Modify: `internal/packaging/application/publish_service.go`
- Modify: `internal/packaging/application/publish_service_test.go`
- Modify: `internal/packaging/http/handlers.go`
- Modify: `internal/packaging/http/routes.go`
- Modify: `internal/backend/httpserver/server.go`
- Modify: `internal/backend/runtime.go`
- Modify: `internal/backend/httpserver/server_test.go`
- Modify: `internal/backend/httpserver/packaging_workflow_test.go`
- Modify: `internal/backend/httpserver/server_workflow_seam_test.go`
- Delete: `internal/packaging/http/access_workflow.go`
- Delete: `internal/packaging/http/access_workflow_test.go`
- Delete: `internal/packaging/infra/workspace_locking.go`
- Modify: `internal/packaging/application/workspace_service.go`
- Modify: `internal/packaging/infra/workspace_adapter.go`
- Modify: `internal/packaging/infra/workspace_backing.go`
- Modify: `internal/packaging/infra/workspace_pack_helpers.go`
- Modify: `internal/platform/persistence/workspace_store.go`
- Modify: `apps/skills-web/src/api/types.ts`
- Modify: `apps/skills-web/src/api/packagingClient.ts`
- Modify: `apps/skills-web/src/api/client.ts`
- Modify: `apps/skills-web/src/api/mockClient.ts`
- Modify: `apps/skills-web/src/lib/releaseHandoff.ts`
- Modify: `apps/skills-web/src/features/packaging/usePackagingWorkflow.ts`
- Modify: `apps/skills-web/src/pages/ImportPage.tsx`
- Modify: `apps/skills-web/src/pages/VersionDetailPage.tsx`
- Modify: `apps/skills-web/src/pages/AuthoringFlowPage.tsx`
- Modify: `apps/skills-web/src/api/client.test.ts`
- Modify: `apps/skills-web/src/pages/ImportPage.test.tsx`
- Modify: `apps/skills-web/src/pages/VersionDetailPage.test.tsx`
- Modify: `apps/skills-web/src/features/packaging/usePackagingWorkflow.test.ts`
- Modify: `apps/skills-cli/src/catalog.ts`
- Modify: `apps/skills-cli/src/packaging-session.ts`
- Modify: `apps/skills-cli/src/import-local-workflow.ts`
- Modify: `apps/skills-cli/src/catalog.test.ts`
- Modify: `apps/skills-cli/src/packaging-session.test.ts`
- Modify: `apps/skills-cli/src/import-local-workflow.test.ts`
- Modify: `apps/skills-cli/README.md`

### Task 1: Replace the packaging contract with stateless request and response types

**Files:**
- Modify: `internal/packaging/domain/types.go`
- Modify: `apps/skills-web/src/api/types.ts`
- Modify: `apps/skills-cli/src/catalog.ts`

- [ ] **Step 1: Write the failing type-level and normalization tests for the new one-shot contract**

```ts
// apps/skills-web/src/api/client.test.ts
it('normalizes import-review responses without workspace fields', async () => {
  const requestJSON = vi.fn().mockResolvedValue({
    ok: true,
    packageName: '@platform.skill/zoom-out',
    version: '1.2.0',
    manifestDigest: 'sha256:test',
    components: ['commands'],
    warnings: [],
    warningDetails: [],
    errors: [],
  });

  const clientModule = await importWithRequestJSON(requestJSON);
  const response = await clientModule.importReviewPackaging({
    source: { type: 'upload_archive', archiveBase64: 'ZmFrZQ==' },
    target: { packageName: '@platform.skill/zoom-out', version: '1.2.0' },
    metadata: { displayName: 'Zoom Out', description: 'Broader context' },
    options: {},
  });

  expect('workspacePath' in response).toBe(false);
  expect(response.ok).toBe(true);
});
```

```ts
// apps/skills-cli/src/catalog.test.ts
it('parses one-shot import-review responses without workspacePath', async () => {
  queueFetchJson({
    ok: true,
    packageName: '@platform.skill/zoom-out',
    version: '1.2.0',
    manifestDigest: 'sha256:test',
    components: ['commands'],
    warnings: [],
    errors: [],
  });

  const response = await importReviewPackaging({
    registry: 'https://catalog.example.com',
    source: { type: 'upload_archive', archiveBase64: 'ZmFrZQ==' },
    target: { packageName: '@platform.skill/zoom-out', version: '1.2.0' },
    metadata: { displayName: 'Zoom Out', description: 'Broader context' },
    options: {},
  });

  expect('workspacePath' in response).toBe(false);
  expect(response.manifestDigest).toBe('sha256:test');
});
```

- [ ] **Step 2: Run the focused frontend and CLI tests to verify they fail first**

Run: `npm test -- --run src/api/client.test.ts`

Run: `npm test -- --run src/catalog.test.ts`

Expected: FAIL because the current API surface only knows `prepare/validate/pack` and still requires `workspacePath`.

- [ ] **Step 3: Replace the domain request and response structs in Go**

```go
// internal/packaging/domain/types.go
type ReviewSummary struct {
	ManifestDigest string   `json:"manifestDigest,omitempty"`
	Components     []string `json:"components,omitempty"`
}

type ImportReviewRequest struct {
	IdempotencyKey     string   `json:"idempotencyKey,omitempty"`
	Source             Source   `json:"source"`
	Target             Target   `json:"target"`
	Metadata           Metadata `json:"metadata"`
	Options            Options  `json:"options"`
	IncludeBundleBase64 bool    `json:"includeBundleBase64,omitempty"`
}

type ImportReviewResponse struct {
	OK             bool            `json:"ok"`
	PackageName    string          `json:"packageName"`
	Version        string          `json:"version"`
	ManifestDigest string          `json:"manifestDigest,omitempty"`
	Components     []string        `json:"components,omitempty"`
	BundleBase64   string          `json:"bundleBase64,omitempty"`
	Warnings       []string        `json:"warnings"`
	WarningDetails []WarningDetail `json:"warningDetails,omitempty"`
	ImportReport   *ImportReport   `json:"importReport,omitempty"`
	Errors         []string        `json:"errors"`
}

type PublishRequest struct {
	IdempotencyKey string        `json:"idempotencyKey,omitempty"`
	Source         Source        `json:"source"`
	Target         Target        `json:"target"`
	Metadata       Metadata      `json:"metadata"`
	Options        Options       `json:"options"`
	Review         ReviewSummary `json:"review,omitempty"`
}
```

- [ ] **Step 4: Replace the web and CLI client-side types and helpers**

```ts
// apps/skills-web/src/api/types.ts
export interface PackagingImportReviewRequest {
  source: PackagingSource;
  target: PackagingTarget;
  metadata: PackagingMetadata;
  options: PackagingOptions;
  idempotencyKey?: string;
  includeBundleBase64?: boolean;
}

export interface PackagingImportReviewResponse {
  ok: boolean;
  packageName: string;
  version: string;
  manifestDigest?: string;
  components: string[];
  bundleBase64?: string;
  warnings: string[];
  warningDetails: WarningDetail[];
  importReport?: ImportReport;
  errors: string[];
}

export interface PackagingPublishRequest {
  source: PackagingSource;
  target: PackagingTarget;
  metadata: PackagingMetadata;
  options: PackagingOptions;
  review?: {
    manifestDigest?: string;
    components?: string[];
  };
  idempotencyKey?: string;
}
```

```ts
// apps/skills-cli/src/catalog.ts
export interface ImportReviewPackagingResponse {
  ok: boolean;
  packageName: string;
  version: string;
  manifestDigest?: string;
  components?: string[];
  bundleBase64?: string;
  warnings: string[];
  warningDetails?: PackagingWarningDetail[];
  importReport?: PackagingImportReport;
  errors: string[];
}
```

- [ ] **Step 5: Run the focused normalization tests again**

Run: `npm test -- --run src/api/client.test.ts`

Run: `npm test -- --run src/catalog.test.ts`

Expected: PASS with one-shot request and response types normalized without `workspacePath`.

- [ ] **Step 6: Commit**

```bash
git add internal/packaging/domain/types.go apps/skills-web/src/api/types.ts apps/skills-cli/src/catalog.ts apps/skills-web/src/api/client.test.ts apps/skills-cli/src/catalog.test.ts
git commit -m "refactor: define one-shot packaging contract"
```

### Task 2: Build one-shot backend orchestration and replace the HTTP routes

**Files:**
- Create: `internal/packaging/application/import_review_service.go`
- Create: `internal/packaging/application/import_review_service_test.go`
- Create: `internal/packaging/http/one_shot_workflow.go`
- Create: `internal/packaging/http/one_shot_workflow_test.go`
- Modify: `internal/packaging/application/service.go`
- Modify: `internal/packaging/application/publish_service.go`
- Modify: `internal/packaging/http/handlers.go`
- Modify: `internal/packaging/http/routes.go`
- Modify: `internal/backend/httpserver/server.go`
- Modify: `internal/backend/httpserver/server_test.go`
- Modify: `internal/backend/httpserver/packaging_workflow_test.go`

- [ ] **Step 1: Write the failing backend route tests for `import-review` and stateless `publish`**

```go
// internal/backend/httpserver/server_test.go
func TestSkillPackagingImportReviewReturnsReviewDataWithoutWorkspacePath(t *testing.T) {
	server := newTestServer()
	fixtureDir := createPackagingFixture(t)

	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/import-review", `{
		"source":`+uploadArchiveSourceJSON(t, fixtureDir)+`,
		"target":{"packageName":"@platform.skill/review-fixture","version":"0.4.0"},
		"metadata":{"displayName":"Review Fixture","description":"Uploaded archive","sourceType":"imported"},
		"options":{}
	}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	assertJSONBodyNotContains(t, recorder, "workspacePath")
	assertJSONBodyContains(t, recorder, "manifestDigest")
}
```

```go
// internal/backend/httpserver/server_test.go
func TestSkillPackagingPublishUsesStatelessInput(t *testing.T) {
	server := newTestServer()
	fixtureDir := createPackagingFixture(t)

	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/publish", `{
		"source":`+uploadArchiveSourceJSON(t, fixtureDir)+`,
		"target":{"packageName":"@platform.skill/review-fixture","version":"0.4.0"},
		"metadata":{"displayName":"Review Fixture","description":"Uploaded archive","sourceType":"imported"},
		"options":{"publishRegistry":"https://npm.internal.example.com"},
		"review":{"manifestDigest":"sha256:test","components":["commands"]}
	}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated && recorder.Code != http.StatusOK {
		t.Fatalf("expected publish success, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	assertJSONBodyNotContains(t, recorder, "workspacePath")
}
```

- [ ] **Step 2: Run the focused backend route tests to verify they fail first**

Run: `go test ./internal/backend/httpserver -run 'TestSkillPackagingImportReviewReturnsReviewDataWithoutWorkspacePath|TestSkillPackagingPublishUsesStatelessInput'`

Expected: FAIL because the route table does not expose `/import-review` and publish still expects `workspacePath`.

- [ ] **Step 3: Implement the new import-review application owner**

```go
// internal/packaging/application/import_review_service.go
type ImportReviewService struct {
	workspace *WorkspaceService
}

func (s *ImportReviewService) Review(ctx context.Context, input packagingdomain.ImportReviewRequest) (packagingdomain.ImportReviewResponse, error) {
	prepared, err := s.workspace.Prepare(ctx, packagingdomain.PrepareRequest{
		IdempotencyKey: input.IdempotencyKey,
		Source:         input.Source,
		Target:         input.Target,
		Metadata:       input.Metadata,
		Options:        input.Options,
	})
	if err != nil {
		return packagingdomain.ImportReviewResponse{}, err
	}
	defer s.workspace.Cleanup(ctx, prepared.WorkspacePath)

	packed, err := s.workspace.Pack(ctx, packagingdomain.PackRequest{
		WorkspacePath: prepared.WorkspacePath,
		Target:        input.Target,
		IncludeBundle: input.IncludeBundleBase64,
	})
	if err != nil {
		return packagingdomain.ImportReviewResponse{}, err
	}

	return packagingdomain.ImportReviewResponse{
		OK:             len(packed.Errors) == 0,
		PackageName:    packed.PackageName,
		Version:        packed.Version,
		ManifestDigest: packed.ManifestDigest,
		Components:     packed.Components,
		BundleBase64:   packed.BundleBase64,
		Warnings:       packed.Warnings,
		WarningDetails: packed.WarningDetails,
		ImportReport:   packed.ImportReport,
		Errors:         packed.Errors,
	}, nil
}
```

- [ ] **Step 4: Rework publish to consume stateless input and verify the optional review summary**

```go
// internal/packaging/application/publish_service.go
func (w *PublishService) Publish(ctx context.Context, input packagingdomain.PublishRequest) (packagingdomain.PublishResponse, error) {
	reviewed, material, err := w.prepareReviewMaterial(ctx, input)
	if err != nil {
		return packagingdomain.PublishResponse{}, err
	}
	defer material.cleanup()

	if err := verifyReviewSummary(input.Review, reviewed); err != nil {
		return packagingdomain.PublishResponse{}, err
	}

	// keep existing registry publish + release-record recovery logic
	// but use reviewed.ManifestDigest / reviewed.Components instead of workspacePath lookups
}
```

```go
// internal/packaging/application/publish_service.go
func verifyReviewSummary(review packagingdomain.ReviewSummary, reviewed packagingdomain.ImportReviewResponse) error {
	if review.ManifestDigest != "" && review.ManifestDigest != reviewed.ManifestDigest {
		return catalog.NewError(409, "review_summary_mismatch", "review manifest digest does not match publish-time package output.", nil)
	}
	if len(review.Components) > 0 && !slices.Equal(review.Components, reviewed.Components) {
		return catalog.NewError(409, "review_summary_mismatch", "review component list does not match publish-time package output.", nil)
	}
	return nil
}
```

- [ ] **Step 5: Replace the HTTP seams and route table**

```go
// internal/packaging/http/routes.go
type Config struct {
	ServesWriteRoutes bool
	ImportReview      gin.HandlerFunc
	Publish           gin.HandlerFunc
}

func RegisterRoutes(engine *gin.Engine, config Config) {
	group := engine.Group("/api/v1/skill-packaging")
	group.POST("/import-review", config.ImportReview)
	group.POST("/publish", config.Publish)
}
```

```go
// internal/backend/httpserver/server.go
server.packagingWorkflow = internalpackaginghttp.NewOneShotWorkflow(internalpackaginghttp.OneShotWorkflowConfig{
	ReviewService:  config.Packaging,
	PublishService: config.Packaging,
	Idempotency:    config.Idempotency,
	Audit:          auditRecorder,
})
```

- [ ] **Step 6: Run the focused backend tests again**

Run: `go test ./internal/packaging/... ./internal/backend/httpserver -run 'TestSkillPackagingImportReviewReturnsReviewDataWithoutWorkspacePath|TestSkillPackagingPublishUsesStatelessInput'`

Expected: PASS with `/import-review` live and stateless publish returning no `workspacePath`.

- [ ] **Step 7: Commit**

```bash
git add internal/packaging/application/import_review_service.go internal/packaging/application/import_review_service_test.go internal/packaging/application/service.go internal/packaging/application/publish_service.go internal/packaging/http/one_shot_workflow.go internal/packaging/http/one_shot_workflow_test.go internal/packaging/http/handlers.go internal/packaging/http/routes.go internal/backend/httpserver/server.go internal/backend/httpserver/server_test.go internal/backend/httpserver/packaging_workflow_test.go
git commit -m "feat: add one-shot packaging workflows"
```

### Task 3: Delete the public workspace-session infrastructure

**Files:**
- Delete: `internal/packaging/http/access_workflow.go`
- Delete: `internal/packaging/http/access_workflow_test.go`
- Delete: `internal/packaging/infra/workspace_locking.go`
- Modify: `internal/packaging/application/workspace_service.go`
- Modify: `internal/packaging/infra/workspace_adapter.go`
- Modify: `internal/packaging/infra/workspace_backing.go`
- Modify: `internal/packaging/infra/workspace_pack_helpers.go`
- Modify: `internal/platform/persistence/workspace_store.go`
- Modify: `internal/backend/runtime.go`
- Modify: `internal/backend/httpserver/server_workflow_seam_test.go`

- [ ] **Step 1: Write the failing backend tests that prove the old routes are gone**

```go
// internal/backend/httpserver/server_test.go
func TestSkillPackagingLegacyWorkspaceRoutesReturnNotFound(t *testing.T) {
	server := newTestServer()

	for _, target := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/v1/skill-packaging/prepare", `{}`},
		{http.MethodPost, "/api/v1/skill-packaging/validate", `{}`},
		{http.MethodPost, "/api/v1/skill-packaging/pack", `{}`},
		{http.MethodPost, "/api/v1/skill-packaging/cleanup", `{}`},
		{http.MethodGet, "/api/v1/skill-packaging/bundle?workspacePath=x&packageName=@platform.skill/x&version=1.0.0", ``},
	} {
		request := httptest.NewRequest(target.method, target.path, strings.NewReader(target.body))
		request.Header.Set("X-Role", "publisher")
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s %s expected 404, got %d", target.method, target.path, recorder.Code)
		}
	}
}
```

- [ ] **Step 2: Run the focused deletion test to verify it fails first**

Run: `go test ./internal/backend/httpserver -run TestSkillPackagingLegacyWorkspaceRoutesReturnNotFound`

Expected: FAIL because the old route table still registers all workspace-session endpoints.

- [ ] **Step 3: Remove the old access workflow and runtime wiring**

```go
// internal/backend/runtime.go
type Config struct {
	Addr            string
	DatabaseURL     string
	ArtifactSource  catalog.ArtifactSource
	RepositoryOptions cataloginfra.SQLRepositoryOptions
	Server          httpserver.ServerConfig
	HTTPServer      httpserver.HTTPServerConfig
	DBPool          DBPoolConfig
	IdempotencyTTL  time.Duration
}

func newPackagingService(root string) *packagingapplication.Service {
	return packagingapplication.NewService(root)
}
```

```go
// internal/backend/httpserver/server.go
type packagingApplication interface {
	packagingapplication.ReviewService
	packagingapplication.PublishService
	ConfigurePublishWorkflow(repo packagingapplication.ReleaseRecordCreator, publisher packagingapplication.PackagePublisher)
}
```

- [ ] **Step 4: Delete the obsolete files and remove workspace-store references from tests**

```text
Delete:
- internal/packaging/http/access_workflow.go
- internal/packaging/http/access_workflow_test.go
- internal/packaging/infra/workspace_locking.go

Remove or rewrite:
- tests that instantiate memory workspace stores only to exercise public workspace-session behavior
- server seam stubs that still implement CleanupContext or BundleFilePathContext
```

- [ ] **Step 5: Run the backend packaging and server suites**

Run: `go test ./internal/packaging/... ./internal/backend/httpserver ./internal/backend`

Expected: PASS with no public workspace-session route registration and no runtime dependency on shared workspace persistence.

- [ ] **Step 6: Commit**

```bash
git add internal/backend/runtime.go internal/backend/httpserver/server.go internal/backend/httpserver/server_workflow_seam_test.go internal/backend/httpserver/server_test.go internal/packaging/application/workspace_service.go internal/packaging/infra/workspace_adapter.go internal/packaging/infra/workspace_backing.go internal/packaging/infra/workspace_pack_helpers.go internal/platform/persistence/workspace_store.go
git rm internal/packaging/http/access_workflow.go internal/packaging/http/access_workflow_test.go internal/packaging/infra/workspace_locking.go
git commit -m "refactor: remove workspace-session packaging infrastructure"
```

### Task 4: Migrate the web API client, release handoff, and import page to the one-shot flow

**Files:**
- Modify: `apps/skills-web/src/api/packagingClient.ts`
- Modify: `apps/skills-web/src/api/client.ts`
- Modify: `apps/skills-web/src/api/mockClient.ts`
- Modify: `apps/skills-web/src/lib/releaseHandoff.ts`
- Modify: `apps/skills-web/src/features/packaging/usePackagingWorkflow.ts`
- Modify: `apps/skills-web/src/pages/ImportPage.tsx`
- Modify: `apps/skills-web/src/pages/VersionDetailPage.tsx`
- Modify: `apps/skills-web/src/pages/AuthoringFlowPage.tsx`
- Modify: `apps/skills-web/src/api/client.test.ts`
- Modify: `apps/skills-web/src/pages/ImportPage.test.tsx`
- Modify: `apps/skills-web/src/pages/VersionDetailPage.test.tsx`
- Modify: `apps/skills-web/src/features/packaging/usePackagingWorkflow.test.ts`

- [ ] **Step 1: Write the failing web tests for the simplified two-step flow**

```tsx
// apps/skills-web/src/pages/ImportPage.test.tsx
it('runs import-review and shows review metadata without workspace status', async () => {
  vi.mocked(client.importReviewPackaging).mockResolvedValue({
    ok: true,
    packageName: '@platform.skill/zoom-out',
    version: '1.2.0',
    manifestDigest: 'sha256:test',
    components: ['commands'],
    warnings: [],
    warningDetails: [],
    errors: [],
  });

  renderImportPage();
  fireEvent.click(await screen.findByRole('button', { name: '检查并生成发布预览' }));

  expect(await screen.findByText('sha256:test')).toBeInTheDocument();
  expect(screen.queryByText(/workspace handle/i)).toBeNull();
});
```

```tsx
// apps/skills-web/src/pages/VersionDetailPage.test.tsx
it('reads release handoff without workspace or bundle paths', async () => {
  renderVersionDetailAt(
    '/skills/%40platform.skill%2Fagent-notes/versions/1.0.0?manifestDigest=sha256%3Atest&components=commands&publishRegistry=https%3A%2F%2Fnpm.internal.example.com'
  );

  expect(await screen.findByText('sha256:test')).toBeInTheDocument();
  expect(screen.queryByText('/tmp/skill-packaging/workspace')).toBeNull();
});
```

- [ ] **Step 2: Run the focused web tests to verify they fail first**

Run: `npm test -- --run src/pages/ImportPage.test.tsx src/pages/VersionDetailPage.test.tsx src/features/packaging/usePackagingWorkflow.test.ts`

Expected: FAIL because the UI still renders workspace-step state and bundle download behavior.

- [ ] **Step 3: Replace the web packaging client surface**

```ts
// apps/skills-web/src/api/packagingClient.ts
export const packagingClient: Pick<
  RegistryApiClient,
  'importReviewPackaging' | 'publishSkill'
> = {
  async importReviewPackaging(request: PackagingImportReviewRequest) {
    const response = await requestJSON<PackagingImportReviewResponse>(
      '/api/v1/skill-packaging/import-review',
      { method: 'POST', body: JSON.stringify(request) }
    );
    return {
      ...normalizePackagingReview(response),
      components: Array.isArray(response.components) ? response.components : [],
    };
  },

  async publishSkill(request: PackagingPublishRequest) {
    const response = await requestJSON<PackagingPublishResponse>(
      '/api/v1/skill-packaging/publish',
      { method: 'POST', body: JSON.stringify(request) }
    );
    return {
      ...normalizePackagingReview(response),
      components: Array.isArray(response.components) ? response.components : [],
    };
  },
};
```

```ts
// apps/skills-web/src/api/client.ts
export const importReviewPackaging = apiClient.importReviewPackaging.bind(apiClient);
export const publishSkill = apiClient.publishSkill.bind(apiClient);
```

- [ ] **Step 4: Remove workspace and bundle-path handoff fields**

```ts
// apps/skills-web/src/lib/releaseHandoff.ts
export interface ReleaseHandoff {
  packageName: string;
  version: string;
  publishRegistry?: string;
  manifestDigest?: string;
  components?: string[];
}
```

```ts
// apps/skills-web/src/features/packaging/usePackagingWorkflow.ts
const releaseHandoffPath = reviewResult
  ? buildReleaseHandoffPath({
      packageName: reviewResult.packageName,
      version: reviewResult.version,
      publishRegistry: publishRegistry.trim() || undefined,
      manifestDigest: reviewResult.manifestDigest,
      components: reviewResult.components,
    })
  : null;
```

- [ ] **Step 5: Simplify `ImportPage` to review then publish**

```tsx
// apps/skills-web/src/pages/ImportPage.tsx
const [reviewResult, setReviewResult] = useState<PackagingImportReviewResponse | null>(null);

async function runImportReview() {
  const response = await importReviewPackaging(await buildReviewRequest());
  setReviewResult(response);
}

async function runPublish() {
  if (!reviewResult?.ok) {
    return;
  }
  const response = await publishSkill({
    ...(await buildPublishRequest()),
    review: {
      manifestDigest: reviewResult.manifestDigest,
      components: reviewResult.components,
    },
  });
  setPublishResult(response);
}
```

- [ ] **Step 6: Run the focused web tests again**

Run: `npm test -- --run src/api/client.test.ts src/pages/ImportPage.test.tsx src/pages/VersionDetailPage.test.tsx src/features/packaging/usePackagingWorkflow.test.ts`

Expected: PASS with no workspace UI, no bundle download path builder, and one-shot review plus publish behavior.

- [ ] **Step 7: Commit**

```bash
git add apps/skills-web/src/api/packagingClient.ts apps/skills-web/src/api/client.ts apps/skills-web/src/api/mockClient.ts apps/skills-web/src/lib/releaseHandoff.ts apps/skills-web/src/features/packaging/usePackagingWorkflow.ts apps/skills-web/src/pages/ImportPage.tsx apps/skills-web/src/pages/VersionDetailPage.tsx apps/skills-web/src/pages/AuthoringFlowPage.tsx apps/skills-web/src/api/client.test.ts apps/skills-web/src/pages/ImportPage.test.tsx apps/skills-web/src/pages/VersionDetailPage.test.tsx apps/skills-web/src/features/packaging/usePackagingWorkflow.test.ts
git commit -m "refactor: migrate web packaging flow to one-shot APIs"
```

### Task 5: Migrate the CLI review and publish flows to the stateless contract

**Files:**
- Modify: `apps/skills-cli/src/catalog.ts`
- Modify: `apps/skills-cli/src/packaging-session.ts`
- Modify: `apps/skills-cli/src/import-local-workflow.ts`
- Modify: `apps/skills-cli/src/catalog.test.ts`
- Modify: `apps/skills-cli/src/packaging-session.test.ts`
- Modify: `apps/skills-cli/src/import-local-workflow.test.ts`
- Modify: `apps/skills-cli/README.md`

- [ ] **Step 1: Write the failing CLI tests for one-shot review and publish**

```ts
// apps/skills-cli/src/catalog.test.ts
it('calls /import-review for registry review flows', async () => {
  queueFetchJson({
    ok: true,
    packageName: '@platform.skill/zoom-out',
    version: '1.2.0',
    manifestDigest: 'sha256:test',
    components: ['commands'],
    warnings: [],
    errors: [],
  });

  await importReviewPackaging({
    registry: 'https://catalog.example.com',
    source: { type: 'upload_archive', archiveBase64: 'ZmFrZQ==' },
    target: { packageName: '@platform.skill/zoom-out', version: '1.2.0' },
    metadata: { displayName: 'Zoom Out', description: 'Broader context' },
    options: {},
  });

  expect(fetch).toHaveBeenCalledWith(
    'https://catalog.example.com/api/v1/skill-packaging/import-review',
    expect.any(Object)
  );
});
```

```ts
// apps/skills-cli/src/import-local-workflow.test.ts
it('publishes through the stateless catalog publish route', async () => {
  queueFetchJson({
    packageName: '@platform.skill/zoom-out',
    version: '1.2.0',
    manifestDigest: 'sha256:test',
    components: ['commands'],
    publishedToRegistry: true,
    created: true,
    idempotentReplay: false,
    versionStatus: 'published',
    installPolicy: 'resolvable',
    warnings: [],
    errors: [],
  });

  await runImportLocalWorkflow({ path: fixtureDir, registry: 'https://catalog.example.com' });

  expect(fetch).toHaveBeenCalledWith(
    'https://catalog.example.com/api/v1/skill-packaging/publish',
    expect.any(Object)
  );
});
```

- [ ] **Step 2: Run the focused CLI tests to verify they fail first**

Run: `npm test -- --run src/catalog.test.ts src/packaging-session.test.ts src/import-local-workflow.test.ts`

Expected: FAIL because the CLI still calls `/prepare`, `/validate`, and `/pack`, and still stores `workspacePath`.

- [ ] **Step 3: Replace the CLI catalog helpers**

```ts
// apps/skills-cli/src/catalog.ts
export async function importReviewPackaging(input: {
  registry?: string;
  idempotencyKey?: string;
  source: PackagingSource;
  target: PackagingTarget;
  metadata: PackagingMetadata;
  options: PackagingOptions;
  includeBundleBase64?: boolean;
}): Promise<ImportReviewPackagingResponse> {
  const url = buildCatalogUrl('/api/v1/skill-packaging/import-review', undefined, input.registry);
  // fetch + normalize ImportReviewPackagingResponse
}

export async function publishSkillPackaging(input: {
  registry?: string;
  idempotencyKey?: string;
  source: PackagingSource;
  target: PackagingTarget;
  metadata: PackagingMetadata;
  options: PackagingOptions;
  review?: { manifestDigest?: string; components?: string[] };
}): Promise<PackagingPublishResponse> {
  const url = buildCatalogUrl('/api/v1/skill-packaging/publish', undefined, input.registry);
  // fetch + normalize PackagingPublishResponse
}
```

- [ ] **Step 4: Collapse CLI session state away from `workspacePath`**

```ts
// apps/skills-cli/src/packaging-session.ts
export interface RegistryPackagingSessionSuccessResult extends SkillAuthoringInspection {
  ok: true;
  sessionKind: 'registry';
  idempotencyKey: string;
  warnings?: string[];
  warningDetails?: PackagingWarningDetail[];
  importReport?: PackagingImportReport;
  manifestDigest?: string;
  components?: string[];
}
```

```ts
// apps/skills-cli/src/import-local-workflow.ts
const publishPayload = await publishSkillPackaging({
  registry,
  source: {
    type: 'upload_archive',
    archiveBase64: createSourceArchiveBase64(directory),
  },
  target: {
    packageName: finalPackageName,
    version: finalVersion,
  },
  metadata: {
    displayName: finalDisplayName,
    description: finalDescription,
    sourceType: inspection.sourceType || 'imported',
  },
  options: {
    ...(publishRegistry ? { publishRegistry } : {}),
    ...(options.strict ? { strict: true } : {}),
  },
});
```

- [ ] **Step 5: Update the CLI README contract description**

```md
### `skills publish`

The publish command follows the one-shot registry flow:

1. `POST /api/v1/skill-packaging/publish`

Optional review-oriented flows use:

1. `POST /api/v1/skill-packaging/import-review`
```

- [ ] **Step 6: Run the focused CLI tests again**

Run: `npm test -- --run src/catalog.test.ts src/packaging-session.test.ts src/import-local-workflow.test.ts`

Expected: PASS with no registry-mode `workspacePath` dependency and no prepare/validate/pack session contract.

- [ ] **Step 7: Commit**

```bash
git add apps/skills-cli/src/catalog.ts apps/skills-cli/src/packaging-session.ts apps/skills-cli/src/import-local-workflow.ts apps/skills-cli/src/catalog.test.ts apps/skills-cli/src/packaging-session.test.ts apps/skills-cli/src/import-local-workflow.test.ts apps/skills-cli/README.md
git commit -m "refactor: migrate CLI packaging flows to one-shot APIs"
```

### Task 6: Run full verification, clean remaining references, and prepare the branch for execution

**Files:**
- Modify: any remaining compile-break references discovered by `rg` and tests

- [ ] **Step 1: Search for leftover workspace-session references**

Run: `rg -n "workspacePath|bundlePath|preparePackaging|validatePackaging|packSkill|cleanupPackagingWorkspace|/api/v1/skill-packaging/(prepare|validate|pack|cleanup|bundle)" internal apps`

Expected: only intentionally retained internal helper references remain; no public API or UI references to the removed contract.

- [ ] **Step 2: Run backend verification**

Run: `go test ./...`

Expected: PASS.

- [ ] **Step 3: Run web verification**

Run: `cd apps/skills-web && npm run type-check`

Expected: PASS.

Run: `cd apps/skills-web && npm test -- --run src/api/client.test.ts src/pages/ImportPage.test.tsx src/pages/VersionDetailPage.test.tsx src/features/packaging/usePackagingWorkflow.test.ts`

Expected: PASS.

- [ ] **Step 4: Run CLI verification**

Run: `cd apps/skills-cli && npm test -- --run src/catalog.test.ts src/packaging-session.test.ts src/import-local-workflow.test.ts`

Expected: PASS.

- [ ] **Step 5: Commit the final cleanup if any verification-driven edits were needed**

```bash
git add -A
git commit -m "test: finish one-shot packaging migration"
```
