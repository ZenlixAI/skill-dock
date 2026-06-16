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
	if imports["skill_registry/app/catalogapi/common"] {
		t.Fatalf("expected recorder.go to stop importing skill_registry/app/catalogapi/common")
	}
}
