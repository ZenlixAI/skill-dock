package artifact

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"path"
	"sort"
	"strings"
)

type BundleScanResult struct {
	DisplayName    string
	Description    string
	ManifestDigest string
	Components     []string
	Shasum         string
}

func scanNPMPackageTarball(reader io.Reader, expectedPackageName string, expectedVersion string) (BundleScanResult, string) {
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return BundleScanResult{}, "bundle is not a valid gzip archive"
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	var (
		skillCount       int
		packageJSONFound bool
		skillContent     []byte
		componentSet     = map[string]struct{}{}
		displayName      string
		description      string
	)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return BundleScanResult{}, "bundle tar stream is unreadable"
		}

		if validationError := validateTarHeaderName(header.Name); validationError != "" {
			return BundleScanResult{}, validationError
		}
		if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeLink {
			return BundleScanResult{}, "bundle must not contain symbolic or hard links"
		}

		normalized, ok := normalizePackageEntry(header.Name)
		if !ok {
			continue
		}

		switch normalized {
		case "SKILL.md":
			skillCount++
			if skillCount > 1 {
				return BundleScanResult{}, "single-skill bundle must contain exactly one root SKILL.md"
			}
			content, err := io.ReadAll(tarReader)
			if err != nil {
				return BundleScanResult{}, "root SKILL.md could not be read"
			}
			skillContent = content
			displayName, description = parseSkillMetadata(content)
		case "package.json":
			packageJSONFound = true
			content, err := io.ReadAll(tarReader)
			if err != nil {
				return BundleScanResult{}, "package.json could not be read"
			}
			if validationError := validatePackageJSON(content, expectedPackageName, expectedVersion); validationError != "" {
				return BundleScanResult{}, validationError
			}
		case ".mcp.json":
			componentSet["mcp"] = struct{}{}
		case "plugin.json":
			componentSet["plugin"] = struct{}{}
		default:
			if strings.HasSuffix(normalized, "/SKILL.md") || strings.Contains(normalized, "/SKILL.md/") {
				return BundleScanResult{}, "single-skill bundle must not contain nested SKILL.md files"
			}
			firstSegment := strings.Split(normalized, "/")[0]
			switch firstSegment {
			case "commands", "agents", "hooks", "scripts", "docs":
				componentSet[firstSegment] = struct{}{}
			}
		}
	}

	switch {
	case !packageJSONFound:
		return BundleScanResult{}, "single-skill bundle must contain package.json"
	case skillCount != 1:
		return BundleScanResult{}, "single-skill bundle must contain exactly one root SKILL.md"
	}

	components := make([]string, 0, len(componentSet))
	for component := range componentSet {
		components = append(components, component)
	}
	sort.Strings(components)

	digest := sha256.Sum256(skillContent)
	return BundleScanResult{
		DisplayName:    displayName,
		Description:    description,
		ManifestDigest: "sha256:" + hex.EncodeToString(digest[:]),
		Components:     components,
	}, ""
}

func validateTarHeaderName(name string) string {
	if strings.HasPrefix(name, "/") {
		return "bundle must not contain absolute paths"
	}
	cleaned := path.Clean(name)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.Contains(cleaned, "/../") {
		return "bundle must not contain path traversal entries"
	}
	return ""
}

func normalizePackageEntry(name string) (string, bool) {
	cleaned := path.Clean(name)
	if cleaned == "package" {
		return "", false
	}
	if !strings.HasPrefix(cleaned, "package/") {
		return "", false
	}
	return strings.TrimPrefix(cleaned, "package/"), true
}

func validatePackageJSON(content []byte, expectedPackageName string, expectedVersion string) string {
	var payload struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(content, &payload); err != nil {
		return "package.json is invalid JSON"
	}
	switch {
	case payload.Name == "":
		return "package.json.name is required"
	case payload.Name != expectedPackageName:
		return "package metadata name does not match requested package"
	case payload.Version == "":
		return "package.json.version is required"
	case payload.Version != expectedVersion:
		return "package metadata version does not match requested version"
	default:
		return ""
	}
}

func parseSkillMetadata(content []byte) (string, string) {
	text := string(content)
	lines := strings.Split(text, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", ""
	}

	var frontmatter []string
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		frontmatter = append(frontmatter, line)
	}

	var displayName string
	var description string
	for _, line := range frontmatter {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "name":
			displayName = strings.Trim(strings.TrimSpace(value), `"'`)
		case "description":
			description = strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	return displayName, description
}

func ScanNPMPackageTarballForTest(reader io.Reader, expectedPackageName string, expectedVersion string) (BundleScanResult, string) {
	return scanNPMPackageTarball(reader, expectedPackageName, expectedVersion)
}
