package sessionhandlers

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestHandlersUseInternalPlatformHTTPHelpers(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "handlers.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse handlers.go imports: %v", err)
	}

	imports := map[string]bool{}
	for _, spec := range file.Imports {
		imports[strings.Trim(spec.Path.Value, "\"")] = true
	}

	if !imports["skill_registry/internal/platform/http"] {
		t.Fatalf("expected handlers.go to import skill_registry/internal/platform/http directly")
	}
	if imports["skill_registry/app/catalogapi/response"] {
		t.Fatalf("expected handlers.go to stop importing skill_registry/app/catalogapi/response")
	}
}
