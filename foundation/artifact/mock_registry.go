package artifact

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"skill_registry/business/catalog"
)

type MockRegistry struct {
	mu        sync.RWMutex
	artifacts map[string][]catalog.ReleaseArtifact
	published map[string]mockPublishedPackage
}

type mockPublishedPackage struct {
	Name        string
	Description string
	DistTags    map[string]string
	Versions    map[string]mockPublishedVersion
}

type mockPublishedVersion struct {
	Version     string
	Description string
	Skill       mockPublishedSkill
	Integrity   string
	Shasum      string
	PublishedAt time.Time
	Tarball     []byte
}

type mockPublishedSkill struct {
	DisplayName string
	SourceType  catalog.SourceType
}

func NewMockRegistry(artifacts []catalog.ReleaseArtifact) *MockRegistry {
	grouped := make(map[string][]catalog.ReleaseArtifact)
	for _, artifact := range artifacts {
		grouped[artifact.Version.PackageName] = append(grouped[artifact.Version.PackageName], artifact)
	}
	return &MockRegistry{
		artifacts: grouped,
		published: make(map[string]mockPublishedPackage),
	}
}

func (m *MockRegistry) PackageMetadata(baseURL string, packageName string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if published, ok := m.published[packageName]; ok {
		versions := make(map[string]map[string]any, len(published.Versions))
		timeMap := make(map[string]string, len(published.Versions))
		for version, item := range published.Versions {
			versionPayload := map[string]any{
				"name":        packageName,
				"version":     version,
				"description": item.Description,
				"dist": map[string]any{
					"tarball":   mockTarballURL(baseURL, packageName, version),
					"integrity": item.Integrity,
					"shasum":    item.Shasum,
				},
			}
			if item.Skill.DisplayName != "" || item.Skill.SourceType != "" {
				versionPayload["skill"] = map[string]any{
					"displayName": item.Skill.DisplayName,
					"sourceType":  item.Skill.SourceType,
				}
			}
			versions[version] = versionPayload
			timeMap[version] = item.PublishedAt.UTC().Format("2006-01-02T15:04:05Z")
		}
		payload, err := json.Marshal(map[string]any{
			"name":        published.Name,
			"description": published.Description,
			"dist-tags":   published.DistTags,
			"time":        timeMap,
			"versions":    versions,
		})
		if err != nil {
			return nil, false
		}
		return payload, true
	}

	artifacts, ok := m.artifacts[packageName]
	if !ok || len(artifacts) == 0 {
		return nil, false
	}

	sort.Slice(artifacts, func(i int, j int) bool {
		return compareExactVersions(artifacts[i].Version.Version, artifacts[j].Version.Version) > 0
	})

	versions := make(map[string]map[string]any, len(artifacts))
	timeMap := make(map[string]string, len(artifacts))
	distTags := map[string]string{}

	for index, artifact := range artifacts {
		version := artifact.Version.Version
		tarball := buildMockTarball(artifact)
		integrity, shasum := digestMockTarball(tarball)
		versions[version] = map[string]any{
			"name":        packageName,
			"version":     version,
			"description": artifact.SkillDescription,
			"dist": map[string]any{
				"tarball":   artifact.Version.Registry.TarballURL,
				"integrity": integrity,
				"shasum":    shasum,
			},
			"skill": map[string]any{
				"displayName":     artifact.SkillDisplayName,
				"sourceType":      artifact.SkillSourceType,
				"manifestDigest":  artifact.Version.Bundle.SkillManifestDigest,
				"components":      artifact.Version.Bundle.Components,
				"validationError": artifact.ValidationError,
			},
		}
		timeMap[version] = artifact.Version.Registry.PublishedAt.UTC().Format("2006-01-02T15:04:05Z")
		if index == 0 {
			distTags["latest"] = version
		}
	}

	payload, err := json.Marshal(map[string]any{
		"name":        packageName,
		"description": artifacts[0].SkillDescription,
		"dist-tags":   distTags,
		"time":        timeMap,
		"versions":    versions,
	})
	if err != nil {
		return nil, false
	}
	return payload, true
}

func (m *MockRegistry) Tarball(packageName string, version string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if published, ok := m.published[packageName]; ok {
		if item, found := published.Versions[version]; found {
			return item.Tarball, true
		}
	}

	artifacts, ok := m.artifacts[packageName]
	if !ok {
		return nil, false
	}
	for _, artifact := range artifacts {
		if artifact.Version.Version == version {
			return buildMockTarball(artifact), true
		}
	}
	return nil, false
}

func (m *MockRegistry) PublishPackage(packageName string, body []byte) bool {
	var payload mockPublishPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	if payload.Name == "" || payload.Name != packageName {
		return false
	}

	published := mockPublishedPackage{
		Name:        payload.Name,
		Description: firstVersionDescription(payload.Versions),
		DistTags:    payload.DistTags,
		Versions:    make(map[string]mockPublishedVersion, len(payload.Versions)),
	}

	for version, item := range payload.Versions {
		attachmentName := packageName + "-" + version + ".tgz"
		attachment, ok := payload.Attachments[attachmentName]
		if !ok {
			attachment, ok = payload.Attachments[packageBasename(packageName)+"-"+version+".tgz"]
		}
		if !ok {
			attachment, ok = payload.Attachments[packageAttachmentName(packageName, version)]
		}
		if !ok {
			continue
		}
		tarball, err := base64.StdEncoding.DecodeString(attachment.Data)
		if err != nil {
			return false
		}
		published.Versions[version] = mockPublishedVersion{
			Version:     version,
			Description: item.Description,
			Skill: mockPublishedSkill{
				DisplayName: item.Skill.DisplayName,
				SourceType:  item.Skill.SourceType,
			},
			Integrity:   item.Dist.Integrity,
			Shasum:      item.Dist.Shasum,
			PublishedAt: time.Now().UTC(),
			Tarball:     tarball,
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.published[packageName] = published
	return true
}

type mockPublishPayload struct {
	Name        string                            `json:"name"`
	DistTags    map[string]string                 `json:"dist-tags"`
	Versions    map[string]mockPublishVersionBody `json:"versions"`
	Attachments map[string]mockPublishAttachment  `json:"_attachments"`
}

type mockPublishVersionBody struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Skill       struct {
		DisplayName string             `json:"displayName"`
		SourceType  catalog.SourceType `json:"sourceType"`
	} `json:"skill"`
	Dist struct {
		Integrity string `json:"integrity"`
		Shasum    string `json:"shasum"`
		Tarball   string `json:"tarball"`
	} `json:"dist"`
}

type mockPublishAttachment struct {
	ContentType string `json:"content_type"`
	Data        string `json:"data"`
	Length      int    `json:"length"`
}

func firstVersionDescription(versions map[string]mockPublishVersionBody) string {
	bestVersion := ""
	description := ""
	for version, item := range versions {
		if bestVersion == "" || compareExactVersions(version, bestVersion) > 0 {
			bestVersion = version
			description = item.Description
		}
	}
	return description
}

func mockTarballURL(baseURL string, packageName string, version string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimPrefix(packageName, "@") + "/-/" + packageBasename(packageName) + "-" + version + ".tgz"
}

func buildMockTarball(artifact catalog.ReleaseArtifact) []byte {
	files := map[string]string{
		"package/SKILL.md":     buildMockSkillMD(artifact),
		"package/package.json": buildMockPackageJSON(artifact),
	}

	for _, component := range artifact.Version.Bundle.Components {
		switch component {
		case "mcp":
			files["package/.mcp.json"] = "{}"
		case "plugin":
			files["package/plugin.json"] = `{"name":"compat"}`
		default:
			files["package/"+component+"/placeholder.txt"] = "placeholder"
		}
	}

	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		content := files[name]
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		_ = tarWriter.WriteHeader(header)
		_, _ = tarWriter.Write([]byte(content))
	}
	_ = tarWriter.Close()
	_ = gzipWriter.Close()
	return archive.Bytes()
}

func digestMockTarball(tarball []byte) (string, string) {
	sha512Sum := sha512.Sum512(tarball)
	sha1Sum := sha1.Sum(tarball)
	return "sha512-" + base64.StdEncoding.EncodeToString(sha512Sum[:]), hex.EncodeToString(sha1Sum[:])
}

func buildMockSkillMD(artifact catalog.ReleaseArtifact) string {
	return "---\nname: " + artifact.SkillDisplayName + "\ndescription: " + artifact.SkillDescription + "\n---\n"
}

func buildMockPackageJSON(artifact catalog.ReleaseArtifact) string {
	return `{"name":"` + artifact.Version.PackageName + `","version":"` + artifact.Version.Version + `"}`
}

func compareExactVersions(left string, right string) int {
	leftParts := strings.Split(left, ".")
	rightParts := strings.Split(right, ".")
	for i := 0; i < 3; i++ {
		leftValue, _ := strconv.Atoi(leftParts[i])
		rightValue, _ := strconv.Atoi(rightParts[i])
		switch {
		case leftValue > rightValue:
			return 1
		case leftValue < rightValue:
			return -1
		}
	}
	return 0
}

func packageBasename(packageName string) string {
	parts := strings.Split(packageName, "/")
	return parts[len(parts)-1]
}
