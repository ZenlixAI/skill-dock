package http

import (
	"go/parser"
	"go/token"
	"net/http"
	"strings"
	"testing"

	catalog "skill_registry/internal/catalog/domain"
)

func TestHandlersUseInternalPlatformHTTPHelpers(t *testing.T) {
	imports := parseImports(t, "handlers.go")

	if !imports["skill_registry/internal/platform/http"] {
		t.Fatalf("expected handlers.go to import skill_registry/internal/platform/http directly")
	}
	for _, importPath := range []string{
		"skill_registry/app/catalogapi/common",
		"skill_registry/app/catalogapi/response",
	} {
		if imports[importPath] {
			t.Fatalf("expected handlers.go to stop importing %s", importPath)
		}
	}
}

func TestReadWorkflowUsesInternalPlatformHTTPHelpers(t *testing.T) {
	imports := parseImports(t, "read_workflow.go")

	if !imports["skill_registry/internal/platform/http"] {
		t.Fatalf("expected read_workflow.go to import skill_registry/internal/platform/http directly")
	}
	for _, importPath := range []string{
		"skill_registry/app/catalogapi/common",
		"skill_registry/app/catalogapi/response",
	} {
		if imports[importPath] {
			t.Fatalf("expected read_workflow.go to stop importing %s", importPath)
		}
	}
}

func TestInstallTargetHelpersDoNotImportAppWrappers(t *testing.T) {
	imports := parseImports(t, "install_target.go")

	for _, importPath := range []string{
		"skill_registry/app/catalogapi/common",
		"skill_registry/app/catalogapi/response",
	} {
		if imports[importPath] {
			t.Fatalf("expected install_target.go to stop importing %s", importPath)
		}
	}
}

func TestRewriteInstallTargetToProxyHandlesHostOnlyRegistry(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:18080/api/v1/skills/%40platform.skill%2Fpdf-skill/install-target", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	response := RewriteInstallTargetToProxy(request, catalog.InstallResolution{
		NPMRegistry: "https://npm.internal.example.com",
		TarballURL:  "https://npm.internal.example.com/platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz",
	})

	if response.TarballURL != "http://127.0.0.1:18080/repository/skills-npm/platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz" {
		t.Fatalf("unexpected tarball url: %q", response.TarballURL)
	}
	if response.NPMRegistry != "http://127.0.0.1:18080/repository/skills-npm" {
		t.Fatalf("unexpected registry url: %q", response.NPMRegistry)
	}
}

func TestRewriteInstallTargetToProxyStripsInternalRegistryPath(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:18080/api/v1/skills/%40platform.skill%2Fpdf-skill/install-target", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	response := RewriteInstallTargetToProxy(request, catalog.InstallResolution{
		NPMRegistry: "https://npm.internal.example.com/repository/skills-npm",
		TarballURL:  "https://npm.internal.example.com/repository/skills-npm/platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz",
	})

	if response.TarballURL != "http://127.0.0.1:18080/repository/skills-npm/platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz" {
		t.Fatalf("unexpected tarball url: %q", response.TarballURL)
	}
	if response.NPMRegistry != "http://127.0.0.1:18080/repository/skills-npm" {
		t.Fatalf("unexpected registry url: %q", response.NPMRegistry)
	}
}

func parseImports(t *testing.T, path string) map[string]bool {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s imports: %v", path, err)
	}

	imports := map[string]bool{}
	for _, spec := range file.Imports {
		imports[strings.Trim(spec.Path.Value, "\"")] = true
	}
	return imports
}
