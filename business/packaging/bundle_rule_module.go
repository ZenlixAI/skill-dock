package packaging

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"skill_registry/business/catalog"
)

var (
	packageNameRE  = regexp.MustCompile(`^(?:@[a-z0-9][a-z0-9._-]*/)?[a-z0-9][a-z0-9._-]*$`)
	exactVersionRE = regexp.MustCompile(`^\d+\.\d+\.\d+(?:-[0-9A-Za-z]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
)

type bundleRuleModule struct{}

type preparedWorkspaceOutcome struct {
	Warnings     []string
	ImportReport *ImportReport
}

type validationResult struct {
	ManifestDigest string
	Components     []string
	Warnings       []string
	Errors         []string
}

func newBundleRuleModule() bundleRuleModule {
	return bundleRuleModule{}
}

func (bundleRuleModule) prepareWorkspace(workspacePath string, input PrepareRequest) (preparedWorkspaceOutcome, error) {
	warnings := []string{}
	if input.Source.Type != "platform_authoring" && !input.Options.Strict {
		normalizationWarnings, err := normalizeImportedWorkspace(workspacePath)
		if err != nil {
			return preparedWorkspaceOutcome{}, err
		}
		warnings = append(warnings, normalizationWarnings...)
	}

	identityRewriteWarning, err := detectPackageIdentityRewrite(workspacePath, input.Target)
	if err != nil {
		return preparedWorkspaceOutcome{}, err
	}
	if identityRewriteWarning != "" {
		warnings = append(warnings, identityRewriteWarning)
	}

	if _, err := os.Stat(filepath.Join(workspacePath, "SKILL.md")); err != nil {
		return preparedWorkspaceOutcome{}, catalog.NewError(422, "missing_skill_md", "Root SKILL.md is required.", nil)
	}
	if err := writePackageJSON(
		workspacePath,
		input.Target,
		input.Metadata,
		input.Options,
		shouldMarkNormalizedFromImport(warnings),
	); err != nil {
		return preparedWorkspaceOutcome{}, catalog.NewError(500, "internal_error", "Failed to write package.json.", nil)
	}

	report := buildImportReport(input.Metadata.SourceType, input.Options.Strict, warnings)
	if err := writeImportReport(workspacePath, report); err != nil {
		return preparedWorkspaceOutcome{}, err
	}

	return preparedWorkspaceOutcome{
		Warnings:     warnings,
		ImportReport: report,
	}, nil
}

func (bundleRuleModule) validate(workspacePath string, target Target) (validationResult, error) {
	return validateWorkspace(workspacePath, target)
}

func (bundleRuleModule) buildBundle(workspacePath string, bundlePath string, includeBundle bool) (string, error) {
	bundleDir := filepath.Dir(bundlePath)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return "", catalog.NewError(500, "internal_error", "Failed to create bundle output directory.", nil)
	}
	if err := createBundleArchive(workspacePath, bundlePath); err != nil {
		return "", catalog.NewError(500, "internal_error", "Failed to create bundle archive.", nil)
	}
	if !includeBundle {
		return "", nil
	}

	bundleContent, err := os.ReadFile(bundlePath)
	if err != nil {
		return "", catalog.NewError(500, "internal_error", "Failed to read bundle archive.", nil)
	}
	return base64.StdEncoding.EncodeToString(bundleContent), nil
}

func validateWorkspace(workspacePath string, target Target) (validationResult, error) {
	info, err := os.Stat(workspacePath)
	if err != nil || !info.IsDir() {
		return validationResult{}, catalog.NewError(422, "invalid_request", "workspacePath must reference an existing directory.", map[string]any{"workspacePath": workspacePath})
	}

	skillPath := filepath.Join(workspacePath, "SKILL.md")
	skillContent, err := os.ReadFile(skillPath)
	if err != nil {
		return validationResult{Errors: []string{"missing_skill_md"}}, nil
	}

	packageJSONPath := filepath.Join(workspacePath, "package.json")
	packageJSONContent, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return validationResult{Errors: []string{"missing_package_json"}}, nil
	}
	if validationError := validatePackageJSON(packageJSONContent, target.PackageName, target.Version); validationError != "" {
		return validationResult{Errors: []string{validationError}}, nil
	}

	components, nestedSkill, hasCapability, err := scanWorkspaceComponents(workspacePath)
	if err != nil {
		return validationResult{}, err
	}
	if nestedSkill {
		return validationResult{Errors: []string{"nested_skill_md"}}, nil
	}
	if !hasCapability {
		return validationResult{Errors: []string{"missing_capability_content"}}, nil
	}

	sum := sha256.Sum256(skillContent)
	return validationResult{
		ManifestDigest: "sha256:" + hex.EncodeToString(sum[:]),
		Components:     components,
		Warnings:       []string{},
		Errors:         []string{},
	}, nil
}

func scanWorkspaceComponents(workspacePath string) ([]string, bool, bool, error) {
	componentSet := map[string]struct{}{}
	lowercasePaths := map[string]string{}
	nestedSkill := false
	hasCapability := false

	err := filepath.WalkDir(workspacePath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relativePath, err := filepath.Rel(workspacePath, path)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(relativePath)
		if relativePath == "." {
			return nil
		}

		if entry.Type()&os.ModeSymlink != 0 {
			return catalog.NewError(422, "unsafe_path_entry", "workspace must not contain symbolic links.", map[string]any{"path": relativePath})
		}
		lowercasePath := strings.ToLower(relativePath)
		if existingPath, ok := lowercasePaths[lowercasePath]; ok && existingPath != relativePath {
			return catalog.NewError(422, "unsafe_path_entry", "workspace must not contain case-conflicting paths.", map[string]any{"path": relativePath, "conflictsWith": existingPath})
		}
		lowercasePaths[lowercasePath] = relativePath

		if entry.IsDir() {
			switch strings.Split(relativePath, "/")[0] {
			case "commands", "agents", "hooks", "scripts", "docs":
				componentSet[strings.Split(relativePath, "/")[0]] = struct{}{}
				hasCapability = true
			}
			return nil
		}

		if relativePath == "SKILL.md" {
			hasCapability = true
		}
		if strings.HasSuffix(relativePath, "/SKILL.md") {
			nestedSkill = true
		}
		switch relativePath {
		case ".mcp.json":
			componentSet["mcp"] = struct{}{}
			hasCapability = true
		case "plugin.json":
			componentSet["plugin"] = struct{}{}
			hasCapability = true
		}
		return nil
	})
	if err != nil {
		return nil, false, false, err
	}

	components := make([]string, 0, len(componentSet))
	for component := range componentSet {
		components = append(components, component)
	}
	sort.Strings(components)
	return components, nestedSkill, hasCapability, nil
}

func createBundleArchive(workspacePath string, bundlePath string) error {
	file, err := os.Create(bundlePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	return filepath.WalkDir(workspacePath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == workspacePath {
			return nil
		}

		relativePath, err := filepath.Rel(workspacePath, path)
		if err != nil {
			return err
		}
		if shouldIgnorePackagingPath(relativePath) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return catalog.NewError(422, "unsafe_path_entry", "workspace must not contain symbolic links.", map[string]any{"path": path})
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		header := &tar.Header{
			Name: "package/" + filepath.ToSlash(relativePath),
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		_, err = tarWriter.Write(content)
		return err
	})
}

func normalizeImportedWorkspace(workspacePath string) ([]string, error) {
	warnings := []string{}

	if _, err := os.Stat(filepath.Join(workspacePath, "package.json")); err != nil {
		if !os.IsNotExist(err) {
			return nil, catalog.NewError(500, "internal_error", "Failed to inspect package.json.", nil)
		}
		warnings = append(warnings, "normalized_missing_package_json")
	}

	_, _, hasCapability, err := scanWorkspaceComponents(workspacePath)
	if err != nil {
		return nil, err
	}
	if !hasCapability {
		warnings = append(warnings, "missing_capability_content")
	}

	return warnings, nil
}

func detectPackageIdentityRewrite(workspacePath string, target Target) (string, error) {
	content, err := os.ReadFile(filepath.Join(workspacePath, "package.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", catalog.NewError(500, "internal_error", "Failed to inspect package.json.", nil)
	}

	var payload struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(content, &payload); err != nil {
		return "", nil
	}
	if payload.Name != "" && payload.Name != target.PackageName {
		return "normalized_rewrote_package_json_identity", nil
	}
	if payload.Version != "" && payload.Version != target.Version {
		return "normalized_rewrote_package_json_identity", nil
	}
	return "", nil
}

func buildImportReport(sourceType string, strictMode bool, warnings []string) *ImportReport {
	if strings.TrimSpace(sourceType) == "" && len(warnings) == 0 && !strictMode {
		return nil
	}

	report := &ImportReport{
		SourceType: sourceType,
		StrictMode: strictMode,
	}
	for _, warning := range warnings {
		switch warning {
		case "normalized_missing_package_json":
			report.Normalized = true
			report.GeneratedPackageJSON = true
			report.GeneratedPaths = append(report.GeneratedPaths, "package.json")
		case "normalized_rewrote_package_json_identity":
			report.Normalized = true
			report.RewrotePackageIdentity = true
		}
	}
	if len(warnings) > 0 {
		report.WarningCodes = append([]string(nil), warnings...)
	}
	if len(report.GeneratedPaths) > 0 {
		report.GeneratedPaths = uniqueStrings(report.GeneratedPaths)
	}
	return report
}

func writeImportReport(workspacePath string, report *ImportReport) error {
	if report == nil {
		return nil
	}
	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return catalog.NewError(500, "internal_error", "Failed to write import report.", nil)
	}
	if err := os.WriteFile(filepath.Join(workspacePath, importReportPath), content, 0o644); err != nil {
		return catalog.NewError(500, "internal_error", "Failed to write import report.", nil)
	}
	return nil
}

func readImportReport(workspacePath string, warnings []string) *ImportReport {
	content, err := os.ReadFile(filepath.Join(workspacePath, importReportPath))
	if err == nil {
		var report ImportReport
		if json.Unmarshal(content, &report) == nil {
			if len(warnings) > 0 && len(report.WarningCodes) == 0 {
				report.WarningCodes = append([]string(nil), warnings...)
			}
			return &report
		}
	}
	return inferImportReportFromWorkspace(workspacePath, warnings)
}

func inferImportReportFromWorkspace(workspacePath string, warnings []string) *ImportReport {
	content, err := os.ReadFile(filepath.Join(workspacePath, "package.json"))
	if err != nil {
		return buildImportReport("", false, warnings)
	}

	var payload struct {
		Skill struct {
			SourceType                     string `json:"sourceType"`
			NormalizedFromImport           bool   `json:"normalizedFromImport"`
			GeneratedPackageJSON           bool   `json:"generatedPackageJson"`
			GeneratedCapabilityPlaceholder bool   `json:"generatedCapabilityPlaceholder"`
			RewrotePackageIdentity         bool   `json:"rewrotePackageIdentity"`
		} `json:"skill"`
	}
	if err := json.Unmarshal(content, &payload); err != nil {
		return buildImportReport("", false, warnings)
	}

	report := buildImportReport(payload.Skill.SourceType, false, warnings)
	if report == nil && !payload.Skill.NormalizedFromImport && !payload.Skill.GeneratedPackageJSON &&
		!payload.Skill.GeneratedCapabilityPlaceholder && !payload.Skill.RewrotePackageIdentity {
		return nil
	}
	if report == nil {
		report = &ImportReport{SourceType: payload.Skill.SourceType}
	}
	if payload.Skill.NormalizedFromImport {
		report.Normalized = true
	}
	if payload.Skill.GeneratedPackageJSON {
		report.GeneratedPackageJSON = true
		report.GeneratedPaths = append(report.GeneratedPaths, "package.json")
	}
	if payload.Skill.GeneratedCapabilityPlaceholder {
		report.Normalized = true
		report.GeneratedCapabilityPlaceholder = true
		report.GeneratedPaths = append(report.GeneratedPaths, "docs/README.md")
	}
	if payload.Skill.RewrotePackageIdentity {
		report.Normalized = true
		report.RewrotePackageIdentity = true
	}
	if len(report.GeneratedPaths) > 0 {
		report.GeneratedPaths = uniqueStrings(report.GeneratedPaths)
	}
	return report
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func shouldMarkNormalizedFromImport(warnings []string) bool {
	for _, warning := range warnings {
		switch warning {
		case "normalized_missing_package_json", "normalized_rewrote_package_json_identity":
			return true
		}
	}
	return false
}

func writePackageJSON(workspacePath string, target Target, metadata Metadata, options Options, normalizedFromImport bool) error {
	payload := map[string]any{
		"name":    target.PackageName,
		"version": target.Version,
	}
	if strings.TrimSpace(metadata.Description) != "" {
		payload["description"] = strings.TrimSpace(metadata.Description)
	}
	skillMetadata := map[string]any{}
	if strings.TrimSpace(metadata.DisplayName) != "" {
		skillMetadata["displayName"] = strings.TrimSpace(metadata.DisplayName)
	}
	if strings.TrimSpace(metadata.SourceType) != "" {
		skillMetadata["sourceType"] = strings.TrimSpace(metadata.SourceType)
	}
	if normalizedFromImport {
		skillMetadata["normalizedFromImport"] = true
	}
	if len(skillMetadata) > 0 {
		payload["skill"] = skillMetadata
	}
	if strings.TrimSpace(options.PublishRegistry) != "" {
		payload["publishConfig"] = map[string]any{"registry": options.PublishRegistry}
	}
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	return os.WriteFile(filepath.Join(workspacePath, "package.json"), content, 0o644)
}

func validateTarget(target Target) error {
	switch {
	case !packageNameRE.MatchString(target.PackageName):
		return catalog.NewError(422, "invalid_package_name", "packageName must be a valid package name.", map[string]any{"packageName": target.PackageName})
	case !hasSkillPackageScope(target.PackageName):
		return catalog.NewError(422, "invalid_package_name", "packageName must use a scoped @<company>.skill/<skill-name> package name.", map[string]any{"packageName": target.PackageName})
	case !exactVersionRE.MatchString(target.Version):
		return catalog.NewError(422, "invalid_version", "version must be an exact semver string.", map[string]any{"version": target.Version})
	default:
		return nil
	}
}

func hasSkillPackageScope(packageName string) bool {
	if !strings.HasPrefix(packageName, "@") {
		return false
	}
	slashIndex := strings.Index(packageName, "/")
	if slashIndex <= 1 {
		return false
	}
	scope := packageName[1:slashIndex]
	return strings.HasSuffix(scope, ".skill")
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

func packageBasename(packageName string) string {
	parts := strings.Split(packageName, "/")
	return parts[len(parts)-1]
}
