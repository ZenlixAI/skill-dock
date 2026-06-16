package httpserver

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestInterfacesUseInternalSystemHTTPTypes(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "interfaces.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse interfaces.go imports: %v", err)
	}

	imports := map[string]bool{}
	for _, spec := range file.Imports {
		imports[strings.Trim(spec.Path.Value, "\"")] = true
	}

	if !imports["skill_registry/internal/system/http"] {
		t.Fatalf("expected interfaces.go to import skill_registry/internal/system/http directly")
	}
	if imports["skill_registry/app/catalogapi/system"] {
		t.Fatalf("expected interfaces.go to stop importing skill_registry/app/catalogapi/system")
	}
}
