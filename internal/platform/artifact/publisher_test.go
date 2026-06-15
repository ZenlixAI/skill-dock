package artifact

import (
	"encoding/json"
	"testing"
)

func TestBuildNPMPublishPayloadUsesBasenameAttachmentForScopedPackage(t *testing.T) {
	payloadBytes, err := buildNPMPublishPayload("http://registry.example.com/repository/skills-npm", NPMPublishInput{
		PackageName: "@company.skill/zoom-out",
		Version:     "1.2.0",
		Description: "Probe payload",
		Tarball:     []byte("fake tgz bytes"),
	})
	if err != nil {
		t.Fatalf("build publish payload: %v", err)
	}

	var payload struct {
		Attachments map[string]json.RawMessage `json:"_attachments"`
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	if _, ok := payload.Attachments["zoom-out-1.2.0.tgz"]; !ok {
		t.Fatalf("expected basename attachment key, got keys=%v", mapKeys(payload.Attachments))
	}
	if _, ok := payload.Attachments["company.skill-zoom-out-1.2.0.tgz"]; ok {
		t.Fatalf("did not expect normalized scoped attachment key, got keys=%v", mapKeys(payload.Attachments))
	}
}

func mapKeys(values map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
