# CatalogAPI Mutation / Audit Workflow Architecture Deepening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deepen `app/catalogapi` so `Server` stops owning the remaining mutation, audit query, and authorization denial workflow details.

**Architecture:** Keep `catalogapi.NewServer(ServerConfig)`, all HTTP paths, all JSON contracts, and existing authorization semantics stable while extracting four deeper modules behind the HTTP Adapter: `catalogMutationModule`, `auditQueryModule`, `packagingAccessModule`, and `accessWorkflow`. Move status mapping, decode/validation, denied-path ceremony, and mutation audit orchestration behind those seams, then shrink `server_test.go` back to Adapter-level coverage.

**Tech Stack:** Go, `net/http`, `testing`, existing `catalog.Repository`, `packaging.Service`, `persist.AuditStore`, in-memory adapters

---

## References

- Spec: `docs/superpowers/specs/2026-06-05-catalogapi-mutation-audit-architecture-deepening-design.zh-CN.md`
- Current Adapter: `app/catalogapi/server.go`
- Existing workflow seams:
  - `app/catalogapi/catalog_read_module.go`
  - `app/catalogapi/governance_module.go`
  - `app/catalogapi/packaging_module.go`
  - `app/catalogapi/session_module.go`

## File Map

- Modify: `app/catalogapi/server.go`
  - Remove in-place mutation, audit query, packaging access, and denial workflow logic from `Server`
- Modify: `app/catalogapi/session.go`
  - Route session authorization denial through `accessWorkflow`
- Create: `app/catalogapi/access_workflow.go`
  - Hold shared action/package authorization, denied response mapping, denied audit, and auth decision observability
- Create: `app/catalogapi/audit_query_module.go`
  - Hold audit query parsing, package filter enforcement, and audit store access
- Create: `app/catalogapi/catalog_mutation_module.go`
  - Hold release record and governance change workflow
- Create: `app/catalogapi/packaging_access_module.go`
  - Hold packaging cleanup and bundle download workflow
- Modify: `app/catalogapi/module_test_helpers_test.go`
  - Add stubs and request helpers for the new seams
- Modify: `app/catalogapi/server_test.go`
  - Keep only route wiring, middleware, and thin delegation coverage
- Create: `app/catalogapi/access_workflow_test.go`
- Create: `app/catalogapi/audit_query_module_test.go`
- Create: `app/catalogapi/catalog_mutation_module_test.go`
- Create: `app/catalogapi/packaging_access_module_test.go`

### Task 1: Lock the Remaining Workflow Behavior With Focused RED Tests

**Files:**
- Modify: `app/catalogapi/module_test_helpers_test.go`
- Create: `app/catalogapi/access_workflow_test.go`
- Create: `app/catalogapi/audit_query_module_test.go`
- Create: `app/catalogapi/catalog_mutation_module_test.go`
- Create: `app/catalogapi/packaging_access_module_test.go`

- [ ] **Step 1: Add a failing denied-path test for the shared authorization workflow**

```go
func TestAccessWorkflowDeniesPackageAccessAndRecordsAudit(t *testing.T) {
	store := persist.NewMemoryAuditStore()
	workflow := &accessWorkflow{
		authorizer: auth.NewAuthorizer(auth.Config{}),
		audit:      NewAuditRecorder(store),
		logger:     log.New(io.Discard, "", 0),
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills?package_name=@platform.skill/private-skill", nil)
	request = request.WithContext(contextWithRequestMetadata(
		auth.ContextWithPrincipal(request.Context(), auth.Principal{Role: "reader", Actor: "reader"}),
		"req_1",
		"trace_1",
	))
	recorder := httptest.NewRecorder()

	ok := workflow.authorizePackage(recorder, request, "@platform.skill/private-skill")

	if ok {
		t.Fatalf("expected package access to be denied")
	}
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", recorder.Code)
	}

	page, err := store.List(context.Background(), persist.AuditQuery{Action: "auth.denied"})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected one auth.denied event, got %d", len(page.Items))
	}
	if page.Items[0].AuthDecision != observability.AuthDecisionDenied {
		t.Fatalf("expected denied auth decision, got %q", page.Items[0].AuthDecision)
	}
}
```

- [ ] **Step 2: Run the denied-path test to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./app/catalogapi -run TestAccessWorkflowDeniesPackageAccessAndRecordsAudit
```

Expected: FAIL because `accessWorkflow` and the request metadata helper do not exist yet.

- [ ] **Step 3: Add a failing audit query module test for package-scoped filter enforcement**

```go
func TestAuditQueryModuleRequiresPackageFilterForScopedPrincipal(t *testing.T) {
	module := &auditQueryModule{
		audit:      persist.NewMemoryAuditStore(),
		authorizer: auth.NewAuthorizer(auth.Config{}),
	}

	_, err := module.listEvents(context.Background(), auth.Principal{
		Role:            "reader",
		Actor:           "reader",
		AllowedPackages: []string{"@platform.skill/private-skill"},
	}, url.Values{})

	if err == nil {
		t.Fatalf("expected package filter requirement error")
	}
	if err.Status != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", err.Status)
	}
}
```

- [ ] **Step 4: Run the audit query module test to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./app/catalogapi -run TestAuditQueryModuleRequiresPackageFilterForScopedPrincipal
```

Expected: FAIL because `auditQueryModule` does not exist yet.

- [ ] **Step 5: Add a failing catalog mutation test for release record status and audit details**

```go
func TestCatalogMutationModuleCreatesReleaseRecordAndRecordsCreatedAudit(t *testing.T) {
	repo := catalog.NewMemoryRepository(catalog.SampleData())
	store := persist.NewMemoryAuditStore()
	module := &catalogMutationModule{
		repo:  repo,
		audit: NewAuditRecorder(store),
	}

	request := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/skills/@platform.skill%2Fnew-skill/versions/1.0.0/release-record",
		strings.NewReader(`{"sourceType":"native"}`),
	)
	request = request.WithContext(contextWithRequestMetadata(
		auth.ContextWithPrincipal(request.Context(), auth.Principal{Role: "publisher", Actor: "publisher"}),
		"req_2",
		"trace_2",
	))

	response, status, err := module.createReleaseRecord(
		request.Context(),
		auth.PrincipalFromContext(request.Context()),
		"@platform.skill/new-skill",
		"1.0.0",
		[]byte(`{"sourceType":"native"}`),
	)

	if err != nil {
		t.Fatalf("create release record: %v", err)
	}
	if status != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", status)
	}
	if !response.Created {
		t.Fatalf("expected created response")
	}
}
```

- [ ] **Step 6: Run the mutation test to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./app/catalogapi -run TestCatalogMutationModuleCreatesReleaseRecordAndRecordsCreatedAudit
```

Expected: FAIL because `catalogMutationModule` does not exist yet.

- [ ] **Step 7: Add a failing packaging access test for bundle download validation and success audit**

```go
func TestPackagingAccessModuleDownloadsBundleAndRecordsAudit(t *testing.T) {
	service := packaging.NewService(t.TempDir())
	workspacePath := prepareManagedWorkspaceForTest(t, service, "@platform.skill/pdf-skill", "1.2.0")
	store := persist.NewMemoryAuditStore()
	module := &packagingAccessModule{
		service: service,
		audit:   NewAuditRecorder(store),
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/skill-packaging/bundle?workspacePath="+url.QueryEscape(workspacePath)+"&packageName=%40platform.skill%2Fpdf-skill&version=1.2.0",
		nil,
	)
	request = request.WithContext(contextWithRequestMetadata(
		auth.ContextWithPrincipal(request.Context(), auth.Principal{Role: "publisher", Actor: "publisher"}),
		"req_3",
		"trace_3",
	))

	content, err := module.downloadBundle(
		request.Context(),
		auth.PrincipalFromContext(request.Context()),
		workspacePath,
		"@platform.skill/pdf-skill",
		"1.2.0",
	)

	if err != nil {
		t.Fatalf("download bundle: %v", err)
	}
	if len(content) == 0 {
		t.Fatalf("expected bundle bytes")
	}
}
```

- [ ] **Step 8: Run the packaging access test to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./app/catalogapi -run TestPackagingAccessModuleDownloadsBundleAndRecordsAudit
```

Expected: FAIL because `packagingAccessModule` and its test helper do not exist yet.

- [ ] **Step 9: Commit the RED test scaffold**

```bash
git add app/catalogapi/module_test_helpers_test.go app/catalogapi/access_workflow_test.go app/catalogapi/audit_query_module_test.go app/catalogapi/catalog_mutation_module_test.go app/catalogapi/packaging_access_module_test.go
git commit -m "test: lock catalogapi mutation and audit workflow seams"
```

### Task 2: Extract `accessWorkflow` and Route Denied Paths Through It

**Files:**
- Create: `app/catalogapi/access_workflow.go`
- Modify: `app/catalogapi/server.go`
- Modify: `app/catalogapi/session.go`
- Modify: `app/catalogapi/module_test_helpers_test.go`
- Create: `app/catalogapi/access_workflow_test.go`

- [ ] **Step 1: Implement the shared authorization seam**

```go
type accessWorkflow struct {
	authorizer *auth.Authorizer
	audit      *AuditRecorder
	logger     *log.Logger
}

func (w *accessWorkflow) authorizeAction(writer http.ResponseWriter, request *http.Request, action auth.Action, packageName string, version string) bool {
	current := auth.PrincipalFromContext(request.Context())
	observability.SetObservedAction(writer, string(action))
	observability.SetObservedAuthDecision(writer, observability.AuthDecisionAllowed)
	if authErr := w.authorizer.AuthorizeAction(current, action); authErr != nil {
		w.deny(writer, request, current, authErr, packageName, version)
		return false
	}
	return true
}

func (w *accessWorkflow) authorizePackage(writer http.ResponseWriter, request *http.Request, packageName string) bool {
	current := auth.PrincipalFromContext(request.Context())
	if authErr := w.authorizer.AuthorizePackage(current, packageName); authErr != nil {
		w.deny(writer, request, current, authErr, packageName, "")
		return false
	}
	return true
}
```

- [ ] **Step 2: Move denied audit writing and failure logging into the seam**

```go
func (w *accessWorkflow) deny(writer http.ResponseWriter, request *http.Request, current auth.Principal, authErr *auth.AuthorizationError, packageName string, version string) {
	observability.SetObservedAuthDecision(writer, observability.AuthDecisionDenied)
	observability.SetObservedOutcome(writer, authErr.Code)
	if err := w.audit.RecordAuthorizationDenied(request, current, authErr.Code, authErr.Status, packageName, version, authErr.Details); err != nil {
		w.logger.Printf(
			`{"event":"audit_write_failed","requestId":%q,"traceId":%q,"action":"auth.denied","authDecision":%q,"error":%q}`,
			requestIDFromContext(request.Context()),
			traceIDFromContext(request.Context()),
			observability.AuthDecisionDenied,
			err.Error(),
		)
	}
	writeAuthorizationError(writer, request, authErr)
}

func contextWithRequestMetadata(ctx context.Context, requestID string, traceID string) context.Context {
	ctx = context.WithValue(ctx, requestIDContextKey, requestID)
	ctx = context.WithValue(ctx, traceIDContextKey, traceID)
	return ctx
}
```

- [ ] **Step 3: Wire `Server` and session handlers to delegate authorization denial**

```go
type Server struct {
	// ...
	accessWorkflow *accessWorkflow
}

server.accessWorkflow = &accessWorkflow{
	authorizer: authorizer,
	audit:      auditRecorder,
	logger:     logger,
}

func (s *Server) authorizeActionAccess(writer http.ResponseWriter, request *http.Request, action auth.Action, packageName string, version string) bool {
	return s.accessWorkflow.authorizeAction(writer, request, action, packageName, version)
}

func (s *Server) authorizePackageAccess(writer http.ResponseWriter, request *http.Request, packageName string) bool {
	return s.accessWorkflow.authorizePackage(writer, request, packageName)
}
```

- [ ] **Step 4: Update the session GET handler to reuse the same seam**

```go
if authErr := s.authorizer.AuthorizeAction(current, auth.ActionReadSession); authErr != nil {
	s.accessWorkflow.deny(writer, request, current, authErr, "", "")
	return
}
```

- [ ] **Step 5: Run the focused denied-path tests to verify GREEN**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./app/catalogapi -run 'TestAccessWorkflowDeniesPackageAccessAndRecordsAudit|TestSessionGetDeniedUsesSharedAccessWorkflow'
```

Expected: PASS.

- [ ] **Step 6: Commit the access workflow extraction**

```bash
git add app/catalogapi/access_workflow.go app/catalogapi/server.go app/catalogapi/session.go app/catalogapi/module_test_helpers_test.go app/catalogapi/access_workflow_test.go
git commit -m "refactor: extract catalogapi access workflow"
```

### Task 3: Extract `auditQueryModule` and Move `audit.read` Off the Adapter

**Files:**
- Create: `app/catalogapi/audit_query_module.go`
- Modify: `app/catalogapi/server.go`
- Create: `app/catalogapi/audit_query_module_test.go`

- [ ] **Step 1: Implement the audit query module with parsing and principal constraints**

```go
type auditQueryModule struct {
	audit      persist.AuditStore
	authorizer *auth.Authorizer
}

func (m *auditQueryModule) listEvents(ctx context.Context, principal auth.Principal, values url.Values) (persist.AuditPage, *catalog.Error) {
	if m.audit == nil {
		return persist.AuditPage{}, catalog.NewError(503, "internal_error", "Audit store is not configured.", nil)
	}

	pageSize, err := parsePageSize(values)
	if err != nil {
		return persist.AuditPage{}, err
	}
	cursor, err := parseCursor(values)
	if err != nil {
		return persist.AuditPage{}, err
	}
	packageName := values.Get("package_name")
	if authErr := m.authorizer.RequirePackageFilter(principal, packageName); authErr != nil {
		return persist.AuditPage{}, authErr.CatalogError()
	}
	authDecision, authDecisionErr := parseAuthDecisionFilter(values.Get("auth_decision"))
	if authDecisionErr != nil {
		return persist.AuditPage{}, authDecisionErr
	}
	version, versionErr := parseExactVersionQueryParam(values.Get("version"), "version")
	if versionErr != nil {
		return persist.AuditPage{}, versionErr
	}

	response, listErr := m.audit.List(ctx, persist.AuditQuery{
		Action:       values.Get("action"),
		Actor:        values.Get("actor"),
		AuthDecision: authDecision,
		PackageName:  packageName,
		Version:      version,
		Outcome:      values.Get("outcome"),
		PageSize:     pageSize,
		Cursor:       cursor,
	})
	if listErr != nil {
		return persist.AuditPage{}, catalogErrorFromError(listErr)
	}
	return response, nil
}
```

- [ ] **Step 2: Make `Server.handleAuditEvents` a thin delegator**

```go
func (s *Server) handleAuditEvents(writer http.ResponseWriter, request *http.Request) {
	packageName := request.URL.Query().Get("package_name")
	if packageName != "" && !s.authorizePackageAccess(writer, request, packageName) {
		return
	}

	response, err := s.auditQueryModule.listEvents(
		request.Context(),
		auth.PrincipalFromContext(request.Context()),
		request.URL.Query(),
	)
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, response)
}
```

- [ ] **Step 3: Cover invalid filters, missing package filter, and successful query combinations**

```go
func TestAuditQueryModuleRejectsInvalidAuthDecision(t *testing.T) {
	module := &auditQueryModule{
		audit:      persist.NewMemoryAuditStore(),
		authorizer: auth.NewAuthorizer(auth.Config{}),
	}

	_, err := module.listEvents(
		context.Background(),
		auth.Principal{Role: "reader", Actor: "reader"},
		url.Values{"auth_decision": []string{"maybe"}},
	)

	if err == nil {
		t.Fatalf("expected invalid auth_decision error")
	}
	if err.Status != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", err.Status)
	}
}

func TestAuditQueryModuleListsEventsWithCombinedFilters(t *testing.T) {
	store := persist.NewMemoryAuditStore()
	_ = store.Append(context.Background(), persist.AuditEvent{
		Action:       "release.record",
		Actor:        "publisher",
		AuthDecision: observability.AuthDecisionAllowed,
		PackageName:  "@platform.skill/pdf-skill",
		Outcome:      "succeeded",
		StatusCode:   http.StatusCreated,
		CreatedAt:    time.Now().UTC(),
	})

	module := &auditQueryModule{
		audit:      store,
		authorizer: auth.NewAuthorizer(auth.Config{}),
	}

	page, err := module.listEvents(
		context.Background(),
		auth.Principal{Role: "publisher", Actor: "publisher"},
		url.Values{
			"action":       []string{"release.record"},
			"auth_decision": []string{"allowed"},
			"package_name": []string{"@platform.skill/pdf-skill"},
			"outcome":      []string{"succeeded"},
		},
	)

	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected one event, got %d", len(page.Items))
	}
	if page.Items[0].Action != "release.record" {
		t.Fatalf("expected release.record event, got %q", page.Items[0].Action)
	}
}
```

- [ ] **Step 4: Run the audit query tests and the existing adapter smoke test**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./app/catalogapi -run 'TestAuditQueryModule|TestAuditEventsCanBeQueriedAndFiltered'
```

Expected: PASS.

- [ ] **Step 5: Commit the audit query extraction**

```bash
git add app/catalogapi/audit_query_module.go app/catalogapi/server.go app/catalogapi/audit_query_module_test.go
git commit -m "refactor: extract catalogapi audit query module"
```

### Task 4: Extract `catalogMutationModule` for Release Record and Governance Change Workflow

**Files:**
- Create: `app/catalogapi/catalog_mutation_module.go`
- Modify: `app/catalogapi/server.go`
- Create: `app/catalogapi/catalog_mutation_module_test.go`

- [ ] **Step 1: Implement release record workflow behind the new module**

```go
type catalogMutationModule struct {
	repo  catalog.WriteRepository
	audit *AuditRecorder
}

func (m *catalogMutationModule) createReleaseRecord(ctx context.Context, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.ReleaseRecordResult, int, *catalog.Error) {
	version, err := decodeVersion(versionPath)
	if err != nil {
		return catalog.ReleaseRecordResult{}, 0, err
	}
	input, decodeErr := decodeOptionalReleaseRecordInputBytes(rawBody)
	if decodeErr != nil {
		m.recordAudit(ctx, principal, "release.record", packageName, version, decodeErr.Status, "failed", map[string]any{"errorCode": decodeErr.Code})
		return catalog.ReleaseRecordResult{}, 0, decodeErr
	}
	response, repoErr := m.repo.CreateReleaseRecord(ctx, packageName, version, input)
	if repoErr != nil {
		catalogErr := catalogErrorFromError(repoErr)
		m.recordAudit(ctx, principal, "release.record", packageName, version, catalogErr.Status, "failed", map[string]any{"errorCode": catalogErr.Code})
		return catalog.ReleaseRecordResult{}, 0, catalogErr
	}
	status := http.StatusOK
	if response.Created {
		status = http.StatusCreated
	}
	m.recordAudit(ctx, principal, "release.record", packageName, version, status, "succeeded", map[string]any{
		"created":          response.Created,
		"idempotentReplay": response.IdempotentReplay,
	})
	return response, status, nil
}
```

- [ ] **Step 2: Implement skill-level and version-level governance workflow in the same seam**

```go
func (m *catalogMutationModule) applySkillGovernanceChange(ctx context.Context, principal auth.Principal, packageName string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error) {
	var payload catalog.SkillGovernanceChangeInput
	if err := decodeJSONBytes(rawBody, &payload); err != nil {
		return catalog.GovernanceChangeResponse{}, 0, err
	}
	response, repoErr := m.repo.ApplySkillGovernanceChange(ctx, packageName, payload, principal.Actor)
	if repoErr != nil {
		catalogErr := catalogErrorFromError(repoErr)
		m.recordAudit(ctx, principal, "governance.skill", packageName, "", catalogErr.Status, "failed", map[string]any{"action": payload.Action, "errorCode": catalogErr.Code})
		return catalog.GovernanceChangeResponse{}, 0, catalogErr
	}
	m.recordAudit(ctx, principal, "governance.skill", packageName, "", http.StatusCreated, "succeeded", map[string]any{"action": payload.Action})
	return response, http.StatusCreated, nil
}
```

- [ ] **Step 3: Add a byte-oriented optional release decoder so the module does not need `*http.Request`**

```go
func decodeOptionalReleaseRecordInputBytes(rawBody []byte) (catalog.ReleaseRecordInput, *catalog.Error) {
	if len(bytes.TrimSpace(rawBody)) == 0 {
		return catalog.ReleaseRecordInput{}, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(rawBody))
	decoder.DisallowUnknownFields()

	var payload catalog.ReleaseRecordInput
	if err := decoder.Decode(&payload); err != nil {
		return catalog.ReleaseRecordInput{}, catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": err.Error()})
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return catalog.ReleaseRecordInput{}, catalog.NewError(422, "invalid_request", "request body must contain a single JSON object.", nil)
	}
	return payload, nil
}
```

- [ ] **Step 4: Replace in-place handlers with thin delegates**

```go
func (s *Server) handleReleaseRecord(writer http.ResponseWriter, request *http.Request, packageName string, versionPath string) {
	rawBody, readErr := io.ReadAll(request.Body)
	if readErr != nil {
		writeCatalogError(writer, request, catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": readErr.Error()}))
		return
	}
	response, status, err := s.catalogMutationModule.createReleaseRecord(
		request.Context(),
		auth.PrincipalFromContext(request.Context()),
		packageName,
		versionPath,
		rawBody,
	)
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	writeJSON(writer, status, response)
}
```

- [ ] **Step 5: Run the mutation module tests and the existing governance/release adapter tests**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./app/catalogapi -run 'TestCatalogMutationModule|TestReleaseRecordCreatesNewSkill|TestSkillGovernanceRetiresSkillAndBlocksInstall'
```

Expected: PASS.

- [ ] **Step 6: Commit the mutation workflow extraction**

```bash
git add app/catalogapi/catalog_mutation_module.go app/catalogapi/server.go app/catalogapi/catalog_mutation_module_test.go
git commit -m "refactor: extract catalogapi mutation module"
```

### Task 5: Extract `packagingAccessModule` for Cleanup and Bundle Download

**Files:**
- Create: `app/catalogapi/packaging_access_module.go`
- Modify: `app/catalogapi/server.go`
- Create: `app/catalogapi/packaging_access_module_test.go`

- [ ] **Step 1: Implement cleanup workflow with validation and mutation audit**

```go
type packagingAccessModule struct {
	service *packaging.Service
	audit   *AuditRecorder
}

func (m *packagingAccessModule) cleanup(ctx context.Context, principal auth.Principal, payload packaging.CleanupRequest) (packaging.CleanupResponse, int, *catalog.Error) {
	if strings.TrimSpace(payload.WorkspacePath) == "" {
		err := catalog.NewError(422, "invalid_request", "workspacePath is required.", nil)
		m.recordAudit(ctx, principal, "packaging.cleanup", "", "", err.Status, "failed", map[string]any{"errorCode": err.Code})
		return packaging.CleanupResponse{}, 0, err
	}
	if err := m.service.Cleanup(payload.WorkspacePath); err != nil {
		catalogErr := catalogErrorFromError(err)
		m.recordAudit(ctx, principal, "packaging.cleanup", "", "", catalogErr.Status, "failed", map[string]any{"errorCode": catalogErr.Code})
		return packaging.CleanupResponse{}, 0, catalogErr
	}
	m.recordAudit(ctx, principal, "packaging.cleanup", "", "", http.StatusOK, "succeeded", map[string]any{"workspacePath": payload.WorkspacePath})
	return packaging.CleanupResponse{WorkspacePath: payload.WorkspacePath, Removed: true}, http.StatusOK, nil
}
```

- [ ] **Step 2: Implement bundle download workflow with managed workspace and file read behavior**

```go
func (m *packagingAccessModule) downloadBundle(ctx context.Context, principal auth.Principal, workspacePath string, packageName string, version string) ([]byte, *catalog.Error) {
	if strings.TrimSpace(workspacePath) == "" || strings.TrimSpace(packageName) == "" || strings.TrimSpace(version) == "" {
		err := catalog.NewError(422, "invalid_request", "workspacePath, packageName, and version are required.", nil)
		m.recordAudit(ctx, principal, "packaging.bundle.download", packageName, version, err.Status, "failed", map[string]any{"errorCode": err.Code})
		return nil, err
	}

	bundlePath, err := m.service.BundlePath(workspacePath, packaging.Target{
		PackageName: packageName,
		Version:     version,
	})
	if err != nil {
		catalogErr := catalogErrorFromError(err)
		m.recordAudit(ctx, principal, "packaging.bundle.download", packageName, version, catalogErr.Status, "failed", map[string]any{"errorCode": catalogErr.Code})
		return nil, catalogErr
	}

	content, readErr := os.ReadFile(bundlePath)
	if readErr != nil {
		err := catalog.NewError(404, "bundle_not_found", "Bundle archive does not exist for the requested workspace.", map[string]any{
			"workspacePath": workspacePath,
			"packageName":   packageName,
			"version":       version,
		})
		m.recordAudit(ctx, principal, "packaging.bundle.download", packageName, version, err.Status, "failed", map[string]any{"errorCode": err.Code})
		return nil, err
	}

	m.recordAudit(ctx, principal, "packaging.bundle.download", packageName, version, http.StatusOK, "succeeded", nil)
	return content, nil
}
```

- [ ] **Step 3: Delegate the HTTP handlers to the new module**

```go
func (s *Server) handlePackagingCleanup(writer http.ResponseWriter, request *http.Request) {
	var payload packaging.CleanupRequest
	if err := decodeJSONBody(request, &payload); err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	response, status, err := s.packagingAccessModule.cleanup(
		request.Context(),
		auth.PrincipalFromContext(request.Context()),
		payload,
	)
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	writeJSON(writer, status, response)
}

func (s *Server) handlePackagingBundleDownload(writer http.ResponseWriter, request *http.Request) {
	packageName := strings.TrimSpace(request.URL.Query().Get("packageName"))
	if packageName != "" && !s.authorizePackageAccess(writer, request, packageName) {
		return
	}
	content, err := s.packagingAccessModule.downloadBundle(
		request.Context(),
		auth.PrincipalFromContext(request.Context()),
		strings.TrimSpace(request.URL.Query().Get("workspacePath")),
		packageName,
		strings.TrimSpace(request.URL.Query().Get("version")),
	)
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	writer.Header().Set("Content-Type", "application/octet-stream")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(content)
}
```

- [ ] **Step 4: Run packaging access tests and the existing cleanup/download adapter test**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./app/catalogapi -run 'TestPackagingAccessModule|TestSkillPackagingCleanupRemovesWorkspaceAndBundle'
```

Expected: PASS.

- [ ] **Step 5: Commit the packaging access extraction**

```bash
git add app/catalogapi/packaging_access_module.go app/catalogapi/server.go app/catalogapi/packaging_access_module_test.go
git commit -m "refactor: extract catalogapi packaging access module"
```

### Task 6: Shrink the Adapter Test Surface and Re-verify the Package

**Files:**
- Modify: `app/catalogapi/server_test.go`
- Modify: `app/catalogapi/module_test_helpers_test.go`

- [ ] **Step 1: Replace giant behavior-only assertions with thin delegation coverage where module tests now own the workflow**

```go
func TestServerRoutesAuditEventsThroughAuditQueryModule(t *testing.T) {
	module := newAuditQueryModuleStubForTest(t)
	server := newTestServer()
	server.auditQueryModule = module

	request := httptest.NewRequest(http.MethodGet, "/api/v1/audit-events", nil)
	request.Header.Set("Authorization", "Bearer reader-secret")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if module.listEventsCallCount() != 1 {
		t.Fatalf("expected audit query module to be called once, got %d", module.listEventsCallCount())
	}
}
```

- [ ] **Step 2: Keep only the adapter-level regression tests that prove route wiring and middleware still hold**

```go
func TestServerRoutesReleaseRecordThroughCatalogMutationModule(t *testing.T) {
	module := newCatalogMutationModuleStubForTest(t)
	server := newTestServer()
	server.catalogMutationModule = module

	request := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/skills/%40platform.skill%2Fpdf-skill/versions/1.2.0/release-record",
		strings.NewReader(`{}`),
	)
	request.Header.Set("Authorization", "Bearer publisher-secret")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK && recorder.Code != http.StatusCreated {
		t.Fatalf("expected success status, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if module.createReleaseRecordCallCount() != 1 {
		t.Fatalf("expected release record workflow to be called once, got %d", module.createReleaseRecordCallCount())
	}
}

func TestServerRoutesBundleDownloadThroughPackagingAccessModule(t *testing.T) {
	module := newPackagingAccessModuleStubForTest(t)
	server := newTestServer()
	server.packagingAccessModule = module

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/skill-packaging/bundle?workspacePath=/tmp/ws&packageName=%40platform.skill%2Fpdf-skill&version=1.2.0",
		nil,
	)
	request.Header.Set("Authorization", "Bearer publisher-secret")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if module.downloadBundleCallCount() != 1 {
		t.Fatalf("expected bundle workflow to be called once, got %d", module.downloadBundleCallCount())
	}
}

func TestServerSessionDeniedPathUsesSharedAccessWorkflow(t *testing.T) {
	server := newTestServer()
	server.accessWorkflow = newAccessWorkflowStubForTest(t, false)
	server.authorizer = auth.NewAuthorizer(auth.Config{})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
	request.Header.Set("Authorization", "Bearer reader-secret")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}
```

- [ ] **Step 3: Run the full `catalogapi` package tests**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./app/catalogapi
```

Expected: PASS.

- [ ] **Step 4: Run the full repository tests for regression coverage**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./...
```

Expected: PASS.

- [ ] **Step 5: Commit the final adapter thinning and verification**

```bash
git add app/catalogapi/server_test.go app/catalogapi/module_test_helpers_test.go
git commit -m "test: narrow catalogapi adapter coverage to route seams"
```

## Self-Review Checklist

- Spec coverage:
  - `accessWorkflow`: covered in Task 2
  - `auditQueryModule`: covered in Task 3
  - `catalogMutationModule`: covered in Task 4
  - `packagingAccessModule`: covered in Task 5
  - `server_test.go` test-surface shrink: covered in Task 6
  - package and full-suite verification: covered in Task 6
- Placeholder scan:
  - No placeholder shortcuts remain in task steps or code samples
- Type consistency:
  - New seam names match the design doc and current `catalogapi` naming style
