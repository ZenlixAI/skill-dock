# Unified Public Registry Entry Point Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expose Catalog API and the npm repository behind one public host so users can install the `skills` CLI package from `/repository/skills-npm/` and receive the same public npm URL back from Catalog-driven flows.

**Architecture:** Keep Catalog and Nexus as separate backends, but unify them at the public edge. Static k8s manifests and the Helm chart will route `/api` to web/Catalog and `/repository/skills-npm` to Nexus on the same public host. Catalog keeps using `SKILL_REGISTRY_NPM_REGISTRY_URL` internally and `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` externally, but Phase B locks the external value to the unified path form. The Go service does not become an npm proxy.

**Tech Stack:** Go (`app/catalogapi`, `cmd/skill-registry`), static Kubernetes YAML, Helm + Traefik `IngressRoute`, Markdown docs, git across two repos (`/Users/ultraman/workspace/skill_registry` and `/Users/ultraman/workspace/skill-registry-helm-chart`)

---

## File Map

### Repo A: `/Users/ultraman/workspace/skill_registry`

- `app/catalogapi/server_test.go`
  - Regression coverage that public npm URLs and tarball URLs preserve the configured unified `/repository/skills-npm/` path.
- `zarf/k8s/k3s-all-in-one.yaml`
  - Static all-in-one k3s deployment; currently exposes separate `skills.example.com` and `nexus.example.com` Ingress objects.
- `zarf/k8s/customer-k3s-example.yaml`
  - Customer-facing static deployment example; currently only exposes the web host.
- `zarf/k8s/dev/kind-stack.yaml`
  - Local/dev k8s example; currently publishes the npm repository as a separate direct URL.
- `docs/k3s-all-in-one.zh-CN.md`
  - Runbook that currently documents separate public web and Nexus addresses.
- `docs/customer-k3s-deployment.zh-CN.md`
  - Customer deployment guide that documents `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` and external npm install examples.

### Repo B: `/Users/ultraman/workspace/skill-registry-helm-chart`

- `values.yaml`
  - Default route topology and public repository URL behavior.
- `values.production.example.yaml`
  - Production sample values; currently models separate `skills.example.com` and `nexus.example.com`.
- `values.192.168.10.124.yaml`
  - Real-world sample values file used locally; must stay aligned with the unified-entry recommendation.
- `values.schema.json`
  - Schema updates for any new/renamed values or validation rules.
- `templates/_helpers.tpl`
  - Central helpers for web/nexus external URLs, public repository URL derivation, and Traefik match rules.
- `templates/traefik/web-ingressroute.yaml`
  - Current web route; may need route expansion or helper changes to coexist with a `/repository/skills-npm` rule on the same host.
- `templates/traefik/nexus-ingressroute.yaml`
  - Current Nexus route; needs to support same-host path-based routing rather than a separate Nexus host by default.
- `templates/catalog/secret.yaml`
  - Feeds `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` into Catalog from helper output.
- `templates/NOTES.txt`
  - Post-install notes that currently print separate web and Nexus routes.
- `README.md`
  - Public chart usage and migration docs that currently teach a separate public Nexus URL.

## Preflight Constraints

- Work in two separate git repos.
- The Helm repo is already dirty on this machine (`templates/web/configmap.yaml`, `values.192.168.10.124.yaml`, `values.schema.json`, `values.yaml`). Stage and commit only the files touched for this feature.
- Do not add a Go-side reverse proxy fallback. If the public edge is misconfigured, failure should stay an edge-routing failure.

### Task 1: Lock the public npm URL contract in Go tests

**Files:**
- Modify: `/Users/ultraman/workspace/skill_registry/app/catalogapi/server_test.go`

- [ ] **Step 1: Write the failing test update**

Replace the current separate-host fixture in `TestInstallTargetUsesPublicRegistryURLWhenConfigured` with a unified-host fixture so the test encodes the Phase B contract.

```go
func TestInstallTargetUsesPublicRegistryURLWhenConfigured(t *testing.T) {
	internalRegistry := "http://nexus.skill-registry.svc.cluster.local:8081/repository/skills-npm"
	publicRegistry := "https://skills.example.com/repository/skills-npm"
	server := NewServer(ServerConfig{
		Repository:           catalog.NewMemoryRepository(catalog.SampleDataWithRegistryBase(internalRegistry)),
		Packaging:            packaging.NewService(""),
		PublicNPMRegistryURL: publicRegistry,
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fpdf-skill/install-target", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response catalog.InstallResolution
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.NPMRegistry != publicRegistry {
		t.Fatalf("expected npmRegistry %q, got %q", publicRegistry, response.NPMRegistry)
	}

	wantTarballURL := publicRegistry + "/platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz"
	if response.TarballURL != wantTarballURL {
		t.Fatalf("expected tarballUrl %q, got %q", wantTarballURL, response.TarballURL)
	}
}
```

- [ ] **Step 2: Run the targeted Go test to verify the baseline**

Run:

```bash
go test ./app/catalogapi -run TestInstallTargetUsesPublicRegistryURLWhenConfigured -v
```

Expected:

- `PASS` if the existing logic is already path-safe
- If it fails, the failure should mention `npmRegistry` or `tarballUrl` mismatch, which means later tasks must preserve the unified path in output

- [ ] **Step 3: Make the minimal test-only adjustment**

Keep the implementation untouched unless Step 2 proves it is broken. The intended code delta is just the fixture switch:

```diff
-	publicRegistry := "http://127.0.0.1:8081/repository/skills-npm"
+	publicRegistry := "https://skills.example.com/repository/skills-npm"
```

- [ ] **Step 4: Re-run the targeted Go test**

Run:

```bash
go test ./app/catalogapi -run TestInstallTargetUsesPublicRegistryURLWhenConfigured -v
```

Expected: `PASS`

- [ ] **Step 5: Commit the current-repo test guard**

```bash
git -C /Users/ultraman/workspace/skill_registry add app/catalogapi/server_test.go
git -C /Users/ultraman/workspace/skill_registry commit -m "test: lock unified public npm registry URL contract"
```

### Task 2: Update static k8s manifests to use one public host with a repository path

**Files:**
- Modify: `/Users/ultraman/workspace/skill_registry/zarf/k8s/k3s-all-in-one.yaml`
- Modify: `/Users/ultraman/workspace/skill_registry/zarf/k8s/customer-k3s-example.yaml`
- Modify: `/Users/ultraman/workspace/skill_registry/zarf/k8s/dev/kind-stack.yaml`

- [ ] **Step 1: Write the manifest expectation down as a failing grep check**

Before editing, verify the current manifests still point to separate public npm URLs:

```bash
rg -n "skill_registry_public_npm_registry_url:|host: nexus\\.example\\.com|127\\.0\\.0\\.1:8081/repository/skills-npm" \
  /Users/ultraman/workspace/skill_registry/zarf/k8s/k3s-all-in-one.yaml \
  /Users/ultraman/workspace/skill_registry/zarf/k8s/customer-k3s-example.yaml \
  /Users/ultraman/workspace/skill_registry/zarf/k8s/dev/kind-stack.yaml
```

Expected:

- matches showing the old separate Nexus host / direct Nexus port forms

- [ ] **Step 2: Update the all-in-one manifest**

Make these exact shape changes in `zarf/k8s/k3s-all-in-one.yaml`:

```yaml
stringData:
  skill_registry_public_npm_registry_url: https://skills.example.com/repository/skills-npm/
```

Replace the separate Nexus Ingress with a same-host path rule:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: skills-web
  namespace: skill-registry
  annotations:
    kubernetes.io/ingress.class: traefik
spec:
  rules:
    - host: skills.example.com
      http:
        paths:
          - path: /repository/skills-npm
            pathType: Prefix
            backend:
              service:
                name: nexus
                port:
                  number: 8081
          - path: /
            pathType: Prefix
            backend:
              service:
                name: skills-web
                port:
                  number: 8080
```

Delete the old `kind: Ingress` block named `nexus`.

- [ ] **Step 3: Update the customer and kind examples**

Use the same public path form in both files:

```yaml
skill_registry_public_npm_registry_url: https://skills.example.com/repository/skills-npm
```

For the local kind example, use the public web host/port rather than the direct Nexus port:

```yaml
skill_registry_public_npm_registry_url: http://127.0.0.1:18080/repository/skills-npm
```

For the customer example, extend the existing `skills.example.com` Ingress to include the Nexus route first:

```yaml
spec:
  rules:
    - host: skills.example.com
      http:
        paths:
          - path: /repository/skills-npm
            pathType: Prefix
            backend:
              service:
                name: nexus
                port:
                  number: 8081
          - path: /
            pathType: Prefix
            backend:
              service:
                name: skills-web
                port:
                  number: 8080
```

- [ ] **Step 4: Validate the manifests structurally**

Run:

```bash
kubectl apply --dry-run=client -f /Users/ultraman/workspace/skill_registry/zarf/k8s/k3s-all-in-one.yaml >/tmp/k3s-all-in-one.dryrun
kubectl apply --dry-run=client -f /Users/ultraman/workspace/skill_registry/zarf/k8s/customer-k3s-example.yaml >/tmp/customer-k3s-example.dryrun
kubectl apply --dry-run=client -f /Users/ultraman/workspace/skill_registry/zarf/k8s/dev/kind-stack.yaml >/tmp/kind-stack.dryrun
```

Expected:

- no YAML/validation errors

Then verify the route/value strings:

```bash
rg -n "skills\\.example\\.com|repository/skills-npm|127\\.0\\.0\\.1:18080/repository/skills-npm" \
  /Users/ultraman/workspace/skill_registry/zarf/k8s/k3s-all-in-one.yaml \
  /Users/ultraman/workspace/skill_registry/zarf/k8s/customer-k3s-example.yaml \
  /Users/ultraman/workspace/skill_registry/zarf/k8s/dev/kind-stack.yaml
```

Expected: only unified-host/path forms

- [ ] **Step 5: Commit the static-manifest slice**

```bash
git -C /Users/ultraman/workspace/skill_registry add \
  zarf/k8s/k3s-all-in-one.yaml \
  zarf/k8s/customer-k3s-example.yaml \
  zarf/k8s/dev/kind-stack.yaml
git -C /Users/ultraman/workspace/skill_registry commit -m "feat: unify public npm registry entry in static k8s manifests"
```

### Task 3: Update the Helm chart repo for same-host path-based routing

**Files:**
- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/_helpers.tpl`
- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/traefik/web-ingressroute.yaml`
- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/traefik/nexus-ingressroute.yaml`
- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/catalog/secret.yaml`
- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/NOTES.txt`
- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/values.yaml`
- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/values.production.example.yaml`
- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/values.192.168.10.124.yaml`
- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/values.schema.json`
- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/README.md`

- [ ] **Step 1: Encode the new default topology in values**

Change the chart defaults so the recommended mode is a shared host with a repository path:

```yaml
traefik:
  routes:
    web:
      enabled: true
      host: skills.example.com
      pathPrefix: /
    nexus:
      enabled: true
      host: skills.example.com
      pathPrefix: /repository/skills-npm
```

Set sample `nexus.repository.publicUrl` examples to empty when the route is enabled so helper-derived unified URLs win.

- [ ] **Step 2: Update helper behavior for public URLs and route matching**

Keep `nexusInternalRepositoryUrl` unchanged, but make the external/public URL helpers derive same-host path forms when the Nexus route is enabled:

```gotemplate
{{- define "skill-registry.nexusExternalBaseUrl" -}}
{{- if and .Values.traefik.routes.nexus.enabled .Values.traefik.routes.nexus.host -}}
{{- printf "https://%s%s" .Values.traefik.routes.nexus.host (include "skill-registry.urlPathPrefix" .Values.traefik.routes.nexus.pathPrefix) -}}
{{- else -}}
{{- "" -}}
{{- end -}}
{{- end -}}

{{- define "skill-registry.nexusPublicRepositoryUrl" -}}
{{- if .Values.nexus.repository.publicUrl -}}
{{- printf "%s/" (.Values.nexus.repository.publicUrl | trimSuffix "/") -}}
{{- else if and .Values.traefik.routes.nexus.enabled .Values.traefik.routes.nexus.host -}}
{{- printf "%s/" (include "skill-registry.nexusExternalBaseUrl" . | trimSuffix "/") -}}
{{- else -}}
{{- "" -}}
{{- end -}}
{{- end -}}
```

Keep the web route helper on `/`, and the nexus route helper on `/repository/skills-npm`.

- [ ] **Step 3: Render separate web and nexus IngressRoutes on the same host**

Do not merge the services into one Traefik object. Keep two `IngressRoute` objects but point them at the same host with non-overlapping `PathPrefix` rules:

```gotemplate
{{- define "skill-registry.webTraefikMatch" -}}
{{- printf "Host(`%s`) && PathPrefix(`%s`)" .Values.traefik.routes.web.host (include "skill-registry.routePathPrefix" .Values.traefik.routes.web.pathPrefix) -}}
{{- end -}}

{{- define "skill-registry.nexusTraefikMatch" -}}
{{- printf "Host(`%s`) && PathPrefix(`%s`)" .Values.traefik.routes.nexus.host (include "skill-registry.routePathPrefix" .Values.traefik.routes.nexus.pathPrefix) -}}
{{- end -}}
```

Update `templates/NOTES.txt` and `README.md` examples to show:

```bash
export SKILLS_REGISTRY_URL=https://skills.example.com
npx --registry https://skills.example.com/repository/skills-npm/ skills --help
```

- [ ] **Step 4: Validate the chart output**

Run:

```bash
helm template skill-registry /Users/ultraman/workspace/skill-registry-helm-chart \
  -f /Users/ultraman/workspace/skill-registry-helm-chart/values.production.example.yaml \
  >/tmp/skill-registry-chart.yaml
```

Expected:

- render succeeds

Then verify the unified-route outputs:

```bash
rg -n "skills.example.com|repository/skills-npm|skill_registry_public_npm_registry_url" /tmp/skill-registry-chart.yaml
```

Expected:

- one web route on `Host(skills.example.com) && PathPrefix(/)`
- one Nexus route on `Host(skills.example.com) && PathPrefix(/repository/skills-npm)`
- catalog secret value `https://skills.example.com/repository/skills-npm/`

Also run the Helm repo validation already used in earlier work:

```bash
helm lint /Users/ultraman/workspace/skill-registry-helm-chart \
  -f /Users/ultraman/workspace/skill-registry-helm-chart/values.production.example.yaml
```

Expected: `0 chart(s) failed`

- [ ] **Step 5: Commit only the chart files for this feature**

```bash
git -C /Users/ultraman/workspace/skill-registry-helm-chart add \
  templates/_helpers.tpl \
  templates/traefik/web-ingressroute.yaml \
  templates/traefik/nexus-ingressroute.yaml \
  templates/catalog/secret.yaml \
  templates/NOTES.txt \
  values.yaml \
  values.production.example.yaml \
  values.192.168.10.124.yaml \
  values.schema.json \
  README.md
git -C /Users/ultraman/workspace/skill-registry-helm-chart commit -m "feat: unify public npm registry host in chart routes"
```

### Task 4: Refresh current-repo docs and run end-to-end documentation checks

**Files:**
- Modify: `/Users/ultraman/workspace/skill_registry/docs/k3s-all-in-one.zh-CN.md`
- Modify: `/Users/ultraman/workspace/skill_registry/docs/customer-k3s-deployment.zh-CN.md`
- Optionally modify if referenced by search hits during edit: `/Users/ultraman/workspace/skill_registry/README.md`

- [ ] **Step 1: Update the runbooks to the new recommended public URL**

Make these documentation examples concrete:

```md
- `skill_registry_public_npm_registry_url`
  - 推荐值：`https://skills.example.com/repository/skills-npm/`
  - 它和 `SKILLS_REGISTRY_URL=https://skills.example.com/api` 共用同一个公共 host
```

And replace CLI package install examples with:

```bash
npx --registry https://skills.example.com/repository/skills-npm/ skills --help
npm install -g skills --registry https://skills.example.com/repository/skills-npm/
```

Keep one explicit compatibility note:

```md
兼容模式：如果你已经单独暴露 `nexus.example.com`，仍可继续把 `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` 指向该地址；统一入口只是新的默认推荐。
```

- [ ] **Step 2: Grep for stale separate-host examples**

Run:

```bash
rg -n "nexus\\.example\\.com/repository/skills-npm|127\\.0\\.0\\.1:8081/repository/skills-npm" \
  /Users/ultraman/workspace/skill_registry/docs \
  /Users/ultraman/workspace/skill_registry/README.md
```

Expected:

- only intentional compatibility notes remain

- [ ] **Step 3: Re-run the focused verification set**

Run:

```bash
go test ./app/catalogapi -run TestInstallTargetUsesPublicRegistryURLWhenConfigured -v
kubectl apply --dry-run=client -f /Users/ultraman/workspace/skill_registry/zarf/k8s/k3s-all-in-one.yaml >/tmp/k3s-all-in-one.final
kubectl apply --dry-run=client -f /Users/ultraman/workspace/skill_registry/zarf/k8s/customer-k3s-example.yaml >/tmp/customer-k3s-example.final
kubectl apply --dry-run=client -f /Users/ultraman/workspace/skill_registry/zarf/k8s/dev/kind-stack.yaml >/tmp/kind-stack.final
```

Expected:

- targeted Go test passes
- all three manifest dry-runs succeed

- [ ] **Step 4: Commit the current-repo docs slice**

```bash
git -C /Users/ultraman/workspace/skill_registry add \
  docs/k3s-all-in-one.zh-CN.md \
  docs/customer-k3s-deployment.zh-CN.md \
  README.md
git -C /Users/ultraman/workspace/skill_registry commit -m "docs: recommend unified public registry entry point"
```

- [ ] **Step 5: Final review summary**

Before marking the task complete, capture the two commit IDs and the final verification commands in the task thread. The summary must include:

```text
- current repo commit(s)
- helm repo commit
- go test command/result
- kubectl dry-run command/result
- helm template + helm lint command/result
```

## Self-Review

- Spec coverage:
  - Phase A public `npx --registry` unification is covered by Task 2 + Task 3 + Task 4.
  - Phase B public URL unification from Catalog responses is covered by Task 1 + Task 3.
  - Compatibility mode is covered by Task 3 + Task 4.
- Placeholder scan:
  - No `TODO` / `TBD` markers.
  - Every code-changing step includes concrete file targets or code snippets.
- Type consistency:
  - `SKILL_REGISTRY_NPM_REGISTRY_URL` remains the internal URL.
  - `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` remains the external URL returned to clients.
  - Unified public route is consistently `/repository/skills-npm/`.
