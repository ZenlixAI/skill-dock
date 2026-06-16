package access

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestWorkflowUsesInternalPlatformHTTPHelpers(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "workflow.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse workflow.go imports: %v", err)
	}

	imports := map[string]bool{}
	for _, spec := range file.Imports {
		imports[strings.Trim(spec.Path.Value, "\"")] = true
	}

	if !imports["skill_registry/internal/platform/http"] {
		t.Fatalf("expected workflow.go to import skill_registry/internal/platform/http directly")
	}
	if imports["skill_registry/app/catalogapi/common"] {
		t.Fatalf("expected workflow.go to stop importing skill_registry/app/catalogapi/common")
	}
}
