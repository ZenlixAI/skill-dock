package artifact

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"skill_registry/business/catalog"
)

func TestJSONArtifactSourceReadsArtifactsFile(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "artifacts.json")

	content, err := json.Marshal([]catalog.ReleaseArtifact{
		{
			SkillDisplayName: "Test Skill",
			SkillDescription: "Artifact source test",
			SkillSourceType:  catalog.SourceTypeImported,
			Version:          newVersion("https://registry.example.com", "@platform.skill/test-skill", "0.2.0", catalog.VersionStatusPublished, catalog.InstallPolicyResolvable, mustTime("2026-05-30T13:00:00Z")),
		},
	})
	if err != nil {
		t.Fatalf("marshal artifacts: %v", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write artifacts file: %v", err)
	}

	source := NewJSONArtifactSource(path)
	artifact, err := source.FindReleaseArtifact(context.Background(), "@platform.skill/test-skill", "0.2.0")
	if err != nil {
		t.Fatalf("find release artifact: %v", err)
	}

	if artifact.Version.PackageName != "@platform.skill/test-skill" {
		t.Fatalf("unexpected package name: %s", artifact.Version.PackageName)
	}
}

func TestNPMRegistryArtifactSourceReadsPackageMetadata(t *testing.T) {
	skillContent := "---\nname: Test Skill\ndescription: Artifact source test\n---\n"
	tarball := buildNPMPackageTarball(t, map[string]string{
		"package/SKILL.md":         skillContent,
		"package/package.json":     `{"name":"@platform.skill/test-skill","version":"0.2.0"}`,
		"package/commands/run.txt": "echo test",
		"package/scripts/do.sh":    "echo script",
		"package/plugin.json":      `{"name":"compat"}`,
	})
	client := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.URL.EscapedPath() != "/@platform.skill%2Ftest-skill" {
				if request.URL.Path == ExpectedPackageTarballPathForTest("@platform.skill/test-skill", "0.2.0") {
					return binaryResponse(http.StatusOK, tarball), nil
				}
				return jsonResponse(http.StatusNotFound, `{}`), nil
			}
			return jsonResponse(http.StatusOK, `{
			"name":"@platform.skill/test-skill",
			"description":"top-level description",
			"dist-tags":{"latest":"0.2.0","beta":"0.2.0"},
			"time":{"0.2.0":"2026-05-30T13:00:00Z"},
			"versions":{
				"0.2.0":{
					"name":"@platform.skill/test-skill",
					"version":"0.2.0",
					"description":"artifact source test",
					"dist":{
						"tarball":"https://registry.example.com/platform.skill/test-skill/-/test-skill-0.2.0.tgz",
						"integrity":"sha512-020",
						"shasum":"sha1-020"
					}
				}
			}
		}`), nil
		}),
	}

	source := NewNPMRegistryArtifactSource("https://registry.example.com", client)
	artifact, err := source.FindReleaseArtifact(context.Background(), "@platform.skill/test-skill", "0.2.0")
	if err != nil {
		t.Fatalf("find release artifact: %v", err)
	}

	if artifact.SkillDisplayName != "Test Skill" {
		t.Fatalf("unexpected display name: %s", artifact.SkillDisplayName)
	}
	if artifact.Version.Registry.TarballURL != "https://registry.example.com/platform.skill/test-skill/-/test-skill-0.2.0.tgz" {
		t.Fatalf("unexpected tarball URL: %s", artifact.Version.Registry.TarballURL)
	}
	if len(artifact.Version.Registry.DistTags) != 2 {
		t.Fatalf("expected 2 dist tags, got %d", len(artifact.Version.Registry.DistTags))
	}
	if artifact.Version.Bundle.SkillManifestDigest != BundleDigestFromStringForTest(skillContent) {
		t.Fatalf("unexpected manifest digest: %s", artifact.Version.Bundle.SkillManifestDigest)
	}
	if len(artifact.Version.Bundle.Components) != 3 || artifact.Version.Bundle.Components[0] != "commands" || artifact.Version.Bundle.Components[1] != "plugin" || artifact.Version.Bundle.Components[2] != "scripts" {
		t.Fatalf("unexpected components: %#v", artifact.Version.Bundle.Components)
	}
}

func TestNPMRegistryArtifactSourceAcceptsPluginOnlyBundleMetadata(t *testing.T) {
	tarball := buildNPMPackageTarball(t, map[string]string{
		"package/SKILL.md":     "---\nname: Test Skill\ndescription: Artifact source test\n---\n",
		"package/package.json": `{"name":"@platform.skill/test-skill","version":"0.2.0"}`,
		"package/plugin.json":  `{"name":"compat"}`,
	})
	client := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.URL.Path == ExpectedPackageTarballPathForTest("@platform.skill/test-skill", "0.2.0") {
				return binaryResponse(http.StatusOK, tarball), nil
			}
			return jsonResponse(http.StatusOK, `{
			"name":"@platform.skill/test-skill",
			"time":{"0.2.0":"2026-05-30T13:00:00Z"},
			"versions":{
				"0.2.0":{
					"name":"@platform.skill/test-skill",
					"version":"0.2.0",
					"dist":{
						"tarball":"https://registry.example.com/platform.skill/test-skill/-/test-skill-0.2.0.tgz",
						"integrity":"sha512-020",
						"shasum":"sha1-020"
					}
				}
			}
		}`), nil
		}),
	}

	source := NewNPMRegistryArtifactSource("https://registry.example.com", client)
	artifact, err := source.FindReleaseArtifact(context.Background(), "@platform.skill/test-skill", "0.2.0")
	if err != nil {
		t.Fatalf("find release artifact: %v", err)
	}

	if artifact.ValidationError != "" {
		t.Fatalf("expected plugin-only bundle to validate, got %s", artifact.ValidationError)
	}
	if len(artifact.Version.Bundle.Components) != 1 || artifact.Version.Bundle.Components[0] != "plugin" {
		t.Fatalf("unexpected plugin components: %#v", artifact.Version.Bundle.Components)
	}
}

func TestNPMRegistryArtifactSourceAcceptsSkillMdOnlyBundleMetadata(t *testing.T) {
	tarball := buildNPMPackageTarball(t, map[string]string{
		"package/SKILL.md":     "---\nname: Test Skill\ndescription: Artifact source test\n---\nBody content\n",
		"package/package.json": `{"name":"@platform.skill/test-skill","version":"0.2.0"}`,
	})
	client := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.URL.Path == ExpectedPackageTarballPathForTest("@platform.skill/test-skill", "0.2.0") {
				return binaryResponse(http.StatusOK, tarball), nil
			}
			return jsonResponse(http.StatusOK, `{
			"name":"@platform.skill/test-skill",
			"time":{"0.2.0":"2026-05-30T13:00:00Z"},
			"versions":{
				"0.2.0":{
					"name":"@platform.skill/test-skill",
					"version":"0.2.0",
					"dist":{
						"tarball":"https://registry.example.com/platform.skill/test-skill/-/test-skill-0.2.0.tgz",
						"integrity":"sha512-020",
						"shasum":"sha1-020"
					}
				}
			}
		}`), nil
		}),
	}

	source := NewNPMRegistryArtifactSource("https://registry.example.com", client)
	artifact, err := source.FindReleaseArtifact(context.Background(), "@platform.skill/test-skill", "0.2.0")
	if err != nil {
		t.Fatalf("find release artifact: %v", err)
	}

	if artifact.ValidationError != "" {
		t.Fatalf("expected SKILL.md-only bundle to validate, got %s", artifact.ValidationError)
	}
	if len(artifact.Version.Bundle.Components) != 0 {
		t.Fatalf("expected no derived components for SKILL.md-only bundle, got %#v", artifact.Version.Bundle.Components)
	}
}

func mustTime(value string) time.Time {
	timestamp, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return timestamp
}

func newVersion(registryBase string, packageName string, version string, versionStatus catalog.VersionStatus, policy catalog.InstallPolicy, publishedAt time.Time) catalog.SkillVersionDetail {
	return catalog.SkillVersionDetail{
		SkillVersionSummary: catalog.SkillVersionSummary{
			PackageName:   packageName,
			Version:       version,
			VersionStatus: versionStatus,
			InstallPolicy: policy,
			PublishedAt:   publishedAt,
		},
		Registry: catalog.RegistryMetadata{
			NPMRegistry: registryBase,
			TarballURL:  "https://registry.example.com/" + strings.TrimPrefix(packageName, "@") + "/-/" + strings.Split(packageName, "/")[1] + "-" + version + ".tgz",
			PublishedAt: publishedAt,
		},
	}
}

type roundTripFunc func(request *http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func binaryResponse(status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/octet-stream"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func buildNPMPackageTarball(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			t.Fatalf("write tar content: %v", err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return archive.Bytes()
}
