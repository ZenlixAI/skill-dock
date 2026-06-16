package catalogapi

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestPlatformHTTPAliasCallersUseInternalPlatformHTTPDirectly(t *testing.T) {
	for _, path := range []string{"access_workflow.go", "server.go"} {
		imports := parseImports(t, path)
		if !imports["skill_registry/internal/platform/http"] {
			t.Fatalf("expected %s to import skill_registry/internal/platform/http directly", path)
		}
		if imports["skill_registry/app/catalogapi/common"] {
			t.Fatalf("expected %s to stop importing skill_registry/app/catalogapi/common", path)
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
