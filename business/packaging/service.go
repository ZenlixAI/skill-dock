package packaging

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"os"
	"os/exec"
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

const requiredPackageScope = "@platform.skill/"

type Service struct {
	tempRoot string
}

func NewService(tempRoot string) *Service {
	if strings.TrimSpace(tempRoot) == "" {
		tempRoot = os.TempDir()
	}
	return &Service{tempRoot: tempRoot}
}

func (s *Service) Prepare(input PrepareRequest) (PrepareResponse, error) {
	if err := validateTarget(input.Target); err != nil {
		return PrepareResponse{}, err
	}

	workspacePath, err := os.MkdirTemp(s.tempRoot, "skill-packaging-*")
	if err != nil {
		return PrepareResponse{}, catalog.NewError(500, "internal_error", "Failed to create packaging workspace.", nil)
	}

	switch input.Source.Type {
	case "local":
		if err := copySourceDirectory(input.Source.Location, workspacePath); err != nil {
			return PrepareResponse{}, err
		}
	case "git":
		if err := prepareGitSource(s.tempRoot, input.Source, workspacePath); err != nil {
			return PrepareResponse{}, err
		}
	case "platform_authoring":
		if err := materializeInlineFiles(workspacePath, input.Source.Files); err != nil {
			return PrepareResponse{}, err
		}
	default:
		return PrepareResponse{}, catalog.NewError(422, "invalid_request", "source.type is invalid.", map[string]any{"sourceType": input.Source.Type})
	}

	if input.Source.Type == "platform_authoring" {
		if err := writeSkillMarkdown(workspacePath, input.Metadata.DisplayName, input.Metadata.Description); err != nil {
			return PrepareResponse{}, catalog.NewError(500, "internal_error", "Failed to write SKILL.md.", nil)
		}
	}

	warnings := []string{}
	if input.Source.Type != "platform_authoring" && !input.Options.Strict {
		normalizationWarnings, err := normalizeImportedWorkspace(workspacePath)
		if err != nil {
			return PrepareResponse{}, err
		}
		warnings = append(warnings, normalizationWarnings...)
	}

	if _, err := os.Stat(filepath.Join(workspacePath, "SKILL.md")); err != nil {
		return PrepareResponse{}, catalog.NewError(422, "missing_skill_md", "Root SKILL.md is required.", nil)
	}
	if err := writePackageJSON(workspacePath, input.Target, input.Metadata, input.Options, len(warnings) > 0); err != nil {
		return PrepareResponse{}, catalog.NewError(500, "internal_error", "Failed to write package.json.", nil)
	}

	return PrepareResponse{
		WorkspacePath: workspacePath,
		PackageName:   input.Target.PackageName,
		Version:       input.Target.Version,
		Warnings:      warnings,
		Errors:        []string{},
	}, nil
}

func (s *Service) Validate(input ValidateRequest) (ValidateResponse, error) {
	if err := validateTarget(input.Target); err != nil {
		return ValidateResponse{}, err
	}
	result, err := validateWorkspace(input.WorkspacePath, input.Target)
	if err != nil {
		return ValidateResponse{}, err
	}
	return ValidateResponse{
		OK:             len(result.Errors) == 0,
		WorkspacePath:  input.WorkspacePath,
		PackageName:    input.Target.PackageName,
		Version:        input.Target.Version,
		ManifestDigest: result.ManifestDigest,
		Components:     result.Components,
		Warnings:       result.Warnings,
		Errors:         result.Errors,
	}, nil
}

func (s *Service) Pack(input PackRequest) (PackResponse, error) {
	if err := validateTarget(input.Target); err != nil {
		return PackResponse{}, err
	}
	result, err := validateWorkspace(input.WorkspacePath, input.Target)
	if err != nil {
		return PackResponse{}, err
	}
	if len(result.Errors) > 0 {
		return PackResponse{
			WorkspacePath: input.WorkspacePath,
			PackageName:   input.Target.PackageName,
			Version:       input.Target.Version,
			Warnings:      result.Warnings,
			Errors:        result.Errors,
		}, nil
	}

	bundlePath := filepath.Join(filepath.Dir(input.WorkspacePath), packageBasename(input.Target.PackageName)+"-"+input.Target.Version+".tgz")
	if err := createBundleArchive(input.WorkspacePath, bundlePath); err != nil {
		return PackResponse{}, catalog.NewError(500, "internal_error", "Failed to create bundle archive.", nil)
	}

	bundleBase64 := ""
	if input.IncludeBundle {
		bundleContent, err := os.ReadFile(bundlePath)
		if err != nil {
			return PackResponse{}, catalog.NewError(500, "internal_error", "Failed to read bundle archive.", nil)
		}
		bundleBase64 = base64.StdEncoding.EncodeToString(bundleContent)
	}

	return PackResponse{
		WorkspacePath:  input.WorkspacePath,
		BundlePath:     bundlePath,
		BundleBase64:   bundleBase64,
		PackageName:    input.Target.PackageName,
		Version:        input.Target.Version,
		ManifestDigest: result.ManifestDigest,
		Components:     result.Components,
		Warnings:       result.Warnings,
		Errors:         result.Errors,
	}, nil
}

type validationResult struct {
	ManifestDigest string
	Components     []string
	Warnings       []string
	Errors         []string
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
			}
			return nil
		}

		if strings.HasSuffix(relativePath, "/SKILL.md") {
			nestedSkill = true
		}
		switch relativePath {
		case ".mcp.json":
			componentSet["mcp"] = struct{}{}
		case "plugin.json":
			componentSet["plugin"] = struct{}{}
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
	return components, nestedSkill, len(componentSet) > 0, nil
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
		if path == workspacePath || entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return catalog.NewError(422, "unsafe_path_entry", "workspace must not contain symbolic links.", map[string]any{"path": path})
		}

		relativePath, err := filepath.Rel(workspacePath, path)
		if err != nil {
			return err
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

func copySourceDirectory(sourcePath string, workspacePath string) error {
	sourcePath = strings.TrimSpace(strings.TrimPrefix(sourcePath, "file://"))
	if sourcePath == "" {
		return catalog.NewError(422, "invalid_request", "source.location is required.", nil)
	}
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil || !sourceInfo.IsDir() {
		return catalog.NewError(422, "invalid_request", "source.location must reference an existing directory.", map[string]any{"location": sourcePath})
	}

	return filepath.WalkDir(sourcePath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == sourcePath {
			return nil
		}
		relativePath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}
		if entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}
		targetPath := filepath.Join(workspacePath, relativePath)
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return catalog.NewError(422, "unsafe_path_entry", "source directory must not contain symbolic links.", map[string]any{"path": relativePath})
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(targetPath, content, 0o644)
	})
}

func materializeInlineFiles(workspacePath string, files map[string]string) error {
	for name, content := range files {
		targetPath, err := resolveWorkspacePath(workspacePath, name)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func writeSkillMarkdown(workspacePath string, displayName string, description string) error {
	content := "---\nname: " + strings.TrimSpace(displayName) + "\ndescription: " + strings.TrimSpace(description) + "\n---\n"
	return os.WriteFile(filepath.Join(workspacePath, "SKILL.md"), []byte(content), 0o644)
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
		readmePath := filepath.Join(workspacePath, "docs", "README.md")
		if err := os.MkdirAll(filepath.Dir(readmePath), 0o755); err != nil {
			return nil, catalog.NewError(500, "internal_error", "Failed to create placeholder capability content.", nil)
		}
		placeholder := `# Imported Skill Placeholder

This file was generated during packaging normalization because the imported skill
did not include any capability directories or compatibility manifests.
`
		if err := os.WriteFile(readmePath, []byte(placeholder), 0o644); err != nil {
			return nil, catalog.NewError(500, "internal_error", "Failed to create placeholder capability content.", nil)
		}
		warnings = append(warnings, "normalized_missing_capability_content")
	}

	return warnings, nil
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
	case !strings.HasPrefix(target.PackageName, requiredPackageScope):
		return catalog.NewError(422, "invalid_package_name", "packageName must use the reserved @platform.skill/ scope.", map[string]any{"packageName": target.PackageName})
	case !exactVersionRE.MatchString(target.Version):
		return catalog.NewError(422, "invalid_version", "version must be an exact semver string.", map[string]any{"version": target.Version})
	default:
		return nil
	}
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

func prepareGitSource(tempRoot string, source Source, workspacePath string) error {
	location := strings.TrimSpace(source.Location)
	if location == "" {
		return catalog.NewError(422, "invalid_request", "source.location is required.", nil)
	}

	cloneURL, subdirectory := normalizeGitLocation(location)
	if source.Ref == "" && subdirectory == "" && isExistingDirectory(location) {
		return copySourceDirectory(location, workspacePath)
	}

	cloneDir, err := os.MkdirTemp(tempRoot, "skill-git-source-*")
	if err != nil {
		return catalog.NewError(500, "internal_error", "Failed to create git clone workspace.", nil)
	}
	defer os.RemoveAll(cloneDir)

	if err := runGitCommand("", "clone", cloneURL, cloneDir); err != nil {
		return catalog.NewError(422, "invalid_request", "Failed to clone git source.", map[string]any{"location": location, "error": err.Error()})
	}
	if strings.TrimSpace(source.Ref) != "" {
		if err := runGitCommand(cloneDir, "checkout", strings.TrimSpace(source.Ref)); err != nil {
			return catalog.NewError(422, "invalid_request", "Failed to checkout requested git ref.", map[string]any{"location": location, "ref": source.Ref, "error": err.Error()})
		}
	}

	sourceDir := cloneDir
	if subdirectory != "" {
		targetDir, err := resolveWorkspacePath(cloneDir, subdirectory)
		if err != nil {
			return err
		}
		info, err := os.Stat(targetDir)
		if err != nil || !info.IsDir() {
			return catalog.NewError(422, "invalid_request", "source.location path must reference an existing directory inside the git repository.", map[string]any{"location": location, "path": subdirectory})
		}
		sourceDir = targetDir
	}
	return copySourceDirectory(sourceDir, workspacePath)
}

func normalizeGitLocation(location string) (string, string) {
	parsed, err := url.Parse(location)
	if err != nil {
		return location, ""
	}
	if parsed.Scheme == "" && parsed.Host == "" {
		return location, ""
	}

	subdirectory := strings.Trim(strings.TrimSpace(parsed.Query().Get("path")), "/")
	parsed.RawQuery = ""
	if subdirectory == "" {
		return parsed.String(), ""
	}
	return parsed.String(), filepath.Clean(subdirectory)
}

func isExistingDirectory(path string) bool {
	info, err := os.Stat(strings.TrimSpace(strings.TrimPrefix(path, "file://")))
	return err == nil && info.IsDir()
}

func runGitCommand(directory string, args ...string) error {
	command := exec.Command("git", args...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
	if output, err := command.CombinedOutput(); err != nil {
		return catalog.NewError(500, "internal_error", "git command failed.", map[string]any{"args": args, "output": strings.TrimSpace(string(output))})
	}
	return nil
}

func resolveWorkspacePath(root string, name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", catalog.NewError(422, "unsafe_path_entry", "file path must not be empty.", map[string]any{"path": name})
	}
	cleanPath := filepath.Clean(trimmed)
	if filepath.IsAbs(cleanPath) {
		return "", catalog.NewError(422, "unsafe_path_entry", "file path must be relative.", map[string]any{"path": name})
	}
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", catalog.NewError(422, "unsafe_path_entry", "file path must stay within the workspace.", map[string]any{"path": name})
	}
	targetPath := filepath.Join(root, cleanPath)
	relativePath, err := filepath.Rel(root, targetPath)
	if err != nil {
		return "", catalog.NewError(422, "unsafe_path_entry", "file path must stay within the workspace.", map[string]any{"path": name})
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", catalog.NewError(422, "unsafe_path_entry", "file path must stay within the workspace.", map[string]any{"path": name})
	}
	return targetPath, nil
}
