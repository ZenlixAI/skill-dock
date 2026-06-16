package http

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestRoutesUseInternalPlatformHTTPHelpers(t *testing.T) {
	imports := parseImports(t, "routes.go")

	if !imports["skill_registry/internal/platform/http"] {
		t.Fatalf("expected routes.go to import skill_registry/internal/platform/http directly")
	}
	for _, importPath := range []string{
		"skill_registry/app/catalogapi/common",
		"skill_registry/app/catalogapi/response",
		"skill_registry/app/catalogapi/system",
	} {
		if imports[importPath] {
			t.Fatalf("expected routes.go to stop importing %s", importPath)
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
