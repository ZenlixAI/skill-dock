package audit

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestRecorderUsesInternalPlatformHTTPDirectly(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "recorder.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse recorder.go imports: %v", err)
	}

	imports := map[string]bool{}
	for _, spec := range file.Imports {
		imports[strings.Trim(spec.Path.Value, "\"")] = true
	}

	if !imports["skill_registry/internal/platform/http"] {
		t.Fatalf("expected recorder.go to import skill_registry/internal/platform/http directly")
	}
}

func TestQueryUsesInternalCatalogHTTPDirectly(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "query.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse query.go imports: %v", err)
	}

	imports := map[string]bool{}
	for _, spec := range file.Imports {
		imports[strings.Trim(spec.Path.Value, "\"")] = true
	}

	if !imports["skill_registry/internal/catalog/http"] {
		t.Fatalf("expected query.go to import skill_registry/internal/catalog/http directly")
	}
}
