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
	if imports["skill_registry/app/catalogapi/response"] {
		t.Fatalf("expected handlers.go to stop importing skill_registry/app/catalogapi/response")
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
