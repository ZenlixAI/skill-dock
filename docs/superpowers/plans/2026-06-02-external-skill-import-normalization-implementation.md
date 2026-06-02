# External Skill Import Normalization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make external `local` / `git` skills with only a valid root `SKILL.md` importable by normalizing them into a minimal publishable workspace during packaging `prepare`, while preserving a strict mode that keeps the current failure behavior.

**Architecture:** Extend `business/packaging` so `Prepare` can detect imported-source gaps, generate a normalized workspace-only `package.json`, optionally add a `docs/README.md` placeholder, and surface normalization warnings. Keep `Validate` and `Pack` structurally strict, then thread the new `options.strict` and warning semantics through the HTTP API, CLI packaging client, and batch import entrypoint.

**Tech Stack:** Go, net/http tests, Vitest, Python (`uv run`), Make, Markdown docs

---

### Task 1: Add Packaging Normalization Core

**Files:**
- Modify: `business/packaging/types.go`
- Modify: `business/packaging/service.go`
- Create: `business/packaging/service_test.go`

- [ ] **Step 1: Write the failing Go tests for imported normalization**

```go
func TestPrepareNormalizesLocalSkillWithoutPackageJSONOrCapabilities(t *testing.T) {
	service := NewService(t.TempDir())
	sourceDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte(`---
name: zoom-out
description: Broader context
---

Prompt body.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	resp, err := service.Prepare(PrepareRequest{
		Source: Source{Type: "local", Location: sourceDir},
		Target: Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "Zoom Out", Description: "Broader context", SourceType: "imported"},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}
	if diff := cmp.Diff([]string{"normalized_missing_package_json", "normalized_missing_capability_content"}, resp.Warnings); diff != "" {
		t.Fatalf("unexpected warnings (-want +got):\n%s", diff)
	}
	if _, err := os.Stat(filepath.Join(resp.WorkspacePath, "docs", "README.md")); err != nil {
		t.Fatalf("expected placeholder docs README: %v", err)
	}
}

func TestPrepareStrictModeDoesNotNormalizeMissingCapabilities(t *testing.T) {
	service := NewService(t.TempDir())
	sourceDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte(`---
name: zoom-out
description: Broader context
---

Prompt body.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	resp, err := service.Prepare(PrepareRequest{
		Source: Source{Type: "local", Location: sourceDir},
		Target: Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "Zoom Out", Description: "Broader context", SourceType: "imported"},
		Options: Options{Strict: true},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}

	validateResp, err := service.Validate(ValidateRequest{
		WorkspacePath: resp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("validate returned error: %v", err)
	}
	if validateResp.OK {
		t.Fatal("expected validate to fail in strict mode")
	}
	if diff := cmp.Diff([]string{"missing_capability_content"}, validateResp.Errors); diff != "" {
		t.Fatalf("unexpected validate errors (-want +got):\n%s", diff)
	}
}
```

- [ ] **Step 2: Run the Go packaging tests and verify they fail for the expected reason**

Run:

```bash
env GOCACHE=/private/tmp/go-cache go test ./business/packaging -run 'TestPrepareNormalizesLocalSkillWithoutPackageJSONOrCapabilities|TestPrepareStrictModeDoesNotNormalizeMissingCapabilities'
```

Expected:

- FAIL because normalization warnings / placeholder files / `Strict` option do not exist yet

- [ ] **Step 3: Implement the minimal packaging changes**

```go
type Options struct {
	PublishRegistry string `json:"publishRegistry,omitempty"`
	Strict          bool   `json:"strict,omitempty"`
}

func (s *Service) Prepare(input PrepareRequest) (PrepareResponse, error) {
	// existing source copy / materialize logic

	warnings := []string{}
	if input.Source.Type != "platform_authoring" && !input.Options.Strict {
		var normalizationWarnings []string
		normalizationWarnings, err = normalizeImportedWorkspace(workspacePath, input.Target, input.Metadata, input.Options)
		if err != nil {
			return PrepareResponse{}, err
		}
		warnings = append(warnings, normalizationWarnings...)
	}

	if _, err := os.Stat(filepath.Join(workspacePath, "SKILL.md")); err != nil {
		return PrepareResponse{}, catalog.NewError(422, "missing_skill_md", "Root SKILL.md is required.", nil)
	}
	if err := writePackageJSON(workspacePath, input.Target, input.Metadata, input.Options, len(warnings) > 0); err != nil {
		return PrepareResponse{}, catalog.NewError(500, "internal_error", "Failed to write package.json.", nil)
	}

	return PrepareResponse{
		WorkspacePath: workspacePath,
		PackageName:   input.Target.PackageName,
		Version:       input.Target.Version,
		Warnings:      warnings,
		Errors:        []string{},
	}, nil
}
```

- [ ] **Step 4: Re-run the Go packaging tests and verify they pass**

Run:

```bash
env GOCACHE=/private/tmp/go-cache go test ./business/packaging -run 'TestPrepareNormalizesLocalSkillWithoutPackageJSONOrCapabilities|TestPrepareStrictModeDoesNotNormalizeMissingCapabilities'
```

Expected:

- PASS

- [ ] **Step 5: Commit the packaging core changes**

```bash
git add business/packaging/types.go business/packaging/service.go business/packaging/service_test.go
git commit -m "feat: normalize imported skill workspaces during packaging"
```

### Task 2: Prove API Contract and Strict Mode Through Catalog HTTP Tests

**Files:**
- Modify: `app/catalogapi/server_test.go`

- [ ] **Step 1: Write the failing API tests for warnings and strict mode**

```go
func TestSkillPackagingPrepareNormalizesImportedLocalSkill(t *testing.T) {
	server := newTestServer()
	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte(`---
name: zoom-out
description: Broader context
---

Prompt body.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"source":{"type":"local","location":"`+sourceDir+`"},
		"target":{"packageName":"@platform.skill/zoom-out","version":"1.0.0"},
		"metadata":{"displayName":"Zoom Out","description":"Broader context","sourceType":"imported"}
	}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected prepare success, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		WorkspacePath string   `json:"workspacePath"`
		Warnings      []string `json:"warnings"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if diff := cmp.Diff([]string{"normalized_missing_package_json", "normalized_missing_capability_content"}, payload.Warnings); diff != "" {
		t.Fatalf("unexpected warnings (-want +got):\n%s", diff)
	}
}

func TestSkillPackagingPrepareStrictModePreservesMissingCapabilityFailure(t *testing.T) {
	// prepare with options.strict=true, then validate and assert missing_capability_content
}
```

- [ ] **Step 2: Run the focused API tests to verify they fail**

Run:

```bash
env GOCACHE=/private/tmp/go-cache go test ./app/catalogapi -run 'TestSkillPackagingPrepareNormalizesImportedLocalSkill|TestSkillPackagingPrepareStrictModePreservesMissingCapabilityFailure'
```

Expected:

- FAIL because the HTTP surface does not yet return normalization warnings or honor `options.strict`

- [ ] **Step 3: Adjust HTTP request/response handling only as needed**

```go
// No new route needed. Ensure JSON decoding picks up options.strict through the updated packaging.Options type,
// and existing response serialization includes the PrepareResponse warnings.
```

- [ ] **Step 4: Re-run the focused API tests and verify they pass**

Run:

```bash
env GOCACHE=/private/tmp/go-cache go test ./app/catalogapi -run 'TestSkillPackagingPrepareNormalizesImportedLocalSkill|TestSkillPackagingPrepareStrictModePreservesMissingCapabilityFailure'
```

Expected:

- PASS

- [ ] **Step 5: Commit the API contract tests**

```bash
git add app/catalogapi/server_test.go
git commit -m "test: cover packaging normalization http contract"
```

### Task 3: Thread Strict and Warnings Through CLI and Batch Import

**Files:**
- Modify: `apps/skills-cli/src/catalog.ts`
- Modify: `apps/skills-cli/src/packaging-session.test.ts`
- Modify: `apps/skills-cli/src/publish-workflow.ts`
- Modify: `tools/batch-import/batch_import_skills.py`
- Modify: `Makefile`

- [ ] **Step 1: Write the failing CLI test for prepare warnings plumbing**

```ts
it('preserves prepare warnings returned by the catalog packaging flow', async () => {
  writeFileSync(join(testDir, 'SKILL.md'), `---
name: zoom-out
description: Broader context
---
`);
  writeFileSync(
    join(testDir, 'package.json'),
    JSON.stringify({ name: '@platform.skill/zoom-out', version: '1.0.0' }, null, 2)
  );
  mkdirSync(join(testDir, 'docs'), { recursive: true });

  fetchMock
    .mockResolvedValueOnce(
      new Response(JSON.stringify({
        workspacePath: '/tmp/workspace',
        packageName: '@platform.skill/zoom-out',
        version: '1.0.0',
        warnings: ['normalized_missing_capability_content'],
        errors: [],
      }), { status: 200, headers: { 'content-type': 'application/json' } })
    )
    .mockResolvedValueOnce(
      new Response(JSON.stringify({
        ok: true,
        workspacePath: '/tmp/workspace',
        packageName: '@platform.skill/zoom-out',
        version: '1.0.0',
        manifestDigest: 'sha256:test',
        components: ['docs'],
        warnings: [],
        errors: [],
      }), { status: 200, headers: { 'content-type': 'application/json' } })
    );

  const result = await runPackagingSession({ path: testDir, registry: 'https://catalog.example.com' });
  expect(result.ok).toBe(true);
  if (result.ok && result.sessionKind === 'registry') {
    expect(result.warnings).toEqual(['normalized_missing_capability_content']);
  }
});
```

- [ ] **Step 2: Run the focused Vitest suite and verify it fails**

Run:

```bash
env HOME=/private/tmp/skills-cli-home XDG_CACHE_HOME=/private/tmp/skills-cli-cache npm --prefix apps/skills-cli test -- packaging-session.test.ts
```

Expected:

- FAIL because registry packaging results do not yet preserve prepare warnings or expose strict-capable request types

- [ ] **Step 3: Implement the minimal CLI and batch-import plumbing**

```ts
export interface PackagingOptions {
  publishRegistry?: string;
  strict?: boolean;
}

export interface PreparePackagingResponse {
  workspacePath: string;
  packageName: string;
  version: string;
  warnings: string[];
  errors: string[];
}
```

```python
parser.add_argument(
    "--strict",
    action="store_true",
    help="Disable imported workspace normalization and require the source to already satisfy the publishable bundle contract.",
)

options: dict[str, Any] = {}
if entry["publishRegistry"]:
    options["publishRegistry"] = entry["publishRegistry"]
if args.strict:
    options["strict"] = True
```

- [ ] **Step 4: Re-run the focused Vitest suite and a dry-run batch-import smoke check**

Run:

```bash
env HOME=/private/tmp/skills-cli-home XDG_CACHE_HOME=/private/tmp/skills-cli-cache npm --prefix apps/skills-cli test -- packaging-session.test.ts
env UV_CACHE_DIR=/private/tmp/uv-cache uv run python tools/batch-import/batch_import_skills.py --help >/dev/null
```

Expected:

- Vitest PASS
- batch import help still exits successfully

- [ ] **Step 5: Commit the CLI and import plumbing**

```bash
git add apps/skills-cli/src/catalog.ts apps/skills-cli/src/packaging-session.test.ts apps/skills-cli/src/publish-workflow.ts tools/batch-import/batch_import_skills.py Makefile
git commit -m "feat: thread packaging normalization through cli and batch import"
```

### Task 4: Update User-Facing Docs and Run End-to-End Verification

**Files:**
- Modify: `README.md`
- Modify: `docs/batch-import-skills.zh-CN.md`
- Modify: `docs/production-packaging-and-customer-usage.zh-CN.md`

- [ ] **Step 1: Update docs to describe default normalization and strict mode**

```md
- External `local` / `git` skills with a valid root `SKILL.md` can now be imported without hand-writing `package.json`.
- Packaging `prepare` normalizes missing bundle metadata into the temporary workspace only.
- Use `--strict` in the batch import tool when you want to reject non-publishable source directories instead of normalizing them.
```

- [ ] **Step 2: Run the full targeted verification set**

Run:

```bash
env GOCACHE=/private/tmp/go-cache go test ./business/packaging ./app/catalogapi
env HOME=/private/tmp/skills-cli-home XDG_CACHE_HOME=/private/tmp/skills-cli-cache npm --prefix apps/skills-cli test -- packaging-session.test.ts publish-workflow.test.ts
env UV_CACHE_DIR=/private/tmp/uv-cache uv run python tools/batch-import/batch_import_skills.py --help >/dev/null
make help >/dev/null
```

Expected:

- All commands exit 0

- [ ] **Step 3: Commit the docs and verification-complete state**

```bash
git add README.md docs/batch-import-skills.zh-CN.md docs/production-packaging-and-customer-usage.zh-CN.md
git commit -m "docs: describe imported skill normalization behavior"
```
