package http

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
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

func TestWorkflowUsesInternalPlatformHTTPHelpers(t *testing.T) {
	imports := parseImports(t, "workflow.go")

	if !imports["skill_registry/internal/platform/http"] {
		t.Fatalf("expected workflow.go to import skill_registry/internal/platform/http directly")
	}
	if !imports["skill_registry/internal/platform/audit"] {
		t.Fatalf("expected workflow.go to import skill_registry/internal/platform/audit directly")
	}
	for _, importPath := range []string{
		"skill_registry/app/catalogapi/audit",
		"skill_registry/app/catalogapi/common",
	} {
		if imports[importPath] {
			t.Fatalf("expected workflow.go to stop importing %s", importPath)
		}
	}
}

func TestAccessWorkflowUsesInternalPlatformHTTPHelpers(t *testing.T) {
	imports := parseImports(t, "access_workflow.go")

	if !imports["skill_registry/internal/platform/http"] {
		t.Fatalf("expected access_workflow.go to import skill_registry/internal/platform/http directly")
	}
	if !imports["skill_registry/internal/platform/audit"] {
		t.Fatalf("expected access_workflow.go to import skill_registry/internal/platform/audit directly")
	}
	for _, importPath := range []string{
		"skill_registry/app/catalogapi/audit",
		"skill_registry/app/catalogapi/common",
	} {
		if imports[importPath] {
			t.Fatalf("expected access_workflow.go to stop importing %s", importPath)
		}
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
