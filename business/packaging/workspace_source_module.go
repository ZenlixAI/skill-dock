package packaging

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"skill_registry/business/catalog"
)

const importReportPath = ".skill-packaging-import-report.json"

var packagingIgnoredDirectories = map[string]struct{}{
	".dist": {},
	".git":  {},
}

const defaultGitCloneTimeout = 45 * time.Second

type workspaceSourceModule struct {
	tempRoot     string
	workspaceTTL time.Duration
}

func newWorkspaceSourceModule(tempRoot string, workspaceTTL time.Duration) workspaceSourceModule {
	return workspaceSourceModule{
		tempRoot:     tempRoot,
		workspaceTTL: workspaceTTL,
	}
}

func (m workspaceSourceModule) prepare(input PrepareRequest) (string, error) {
	if m.workspaceTTL > 0 {
		if err := m.pruneExpiredWorkspaces(m.workspaceTTL); err != nil {
			return "", catalog.NewError(500, "internal_error", "Failed to prune expired packaging workspaces.", nil)
		}
	}

	workspacePath, err := os.MkdirTemp(m.tempRoot, "skill-packaging-*")
	if err != nil {
		return "", catalog.NewError(500, "internal_error", "Failed to create packaging workspace.", nil)
	}

	switch input.Source.Type {
	case "local":
		return "", catalog.NewError(422, "unsupported_source_type", "source.type=local has been removed. Upload the source archive content instead of sending a local filesystem path.", map[string]any{"sourceType": input.Source.Type})
	case "git":
		if err := m.prepareGitSource(input.Source, workspacePath); err != nil {
			return "", err
		}
	case "upload_archive":
		if err := extractUploadedArchive(input.Source.ArchiveBase64, workspacePath); err != nil {
			return "", err
		}
	case "platform_authoring":
		if err := materializeInlineFiles(workspacePath, input.Source.Files); err != nil {
			return "", err
		}
	default:
		return "", catalog.NewError(422, "invalid_request", "source.type is invalid.", map[string]any{"sourceType": input.Source.Type})
	}

	if input.Source.Type == "platform_authoring" {
		if err := writeSkillMarkdown(workspacePath, input.Metadata.DisplayName, input.Metadata.Description); err != nil {
			return "", catalog.NewError(500, "internal_error", "Failed to write SKILL.md.", nil)
		}
	}

	return workspacePath, nil
}

func (m workspaceSourceModule) cleanup(workspacePath string) error {
	if err := ensureManagedWorkspace(workspacePath, m.managesWorkspace(workspacePath)); err != nil {
		return err
	}
	if err := os.RemoveAll(workspacePath); err != nil {
		return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
	}
	return nil
}

func (m workspaceSourceModule) bundlePath(workspacePath string, target Target) (string, error) {
	if err := ensureManagedWorkspace(workspacePath, m.managesWorkspace(workspacePath)); err != nil {
		return "", err
	}
	return filepath.Join(workspacePath, ".dist", packageBasename(target.PackageName)+"-"+target.Version+".tgz"), nil
}

func (m workspaceSourceModule) pruneExpiredWorkspaces(ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}

	entries, err := os.ReadDir(m.tempRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cutoff := time.Now().Add(-ttl)
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "skill-packaging-") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.ModTime().Before(cutoff) {
			if err := os.RemoveAll(filepath.Join(m.tempRoot, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m workspaceSourceModule) managesWorkspace(workspacePath string) bool {
	cleanWorkspacePath := filepath.Clean(workspacePath)
	relativePath, err := filepath.Rel(m.tempRoot, cleanWorkspacePath)
	if err != nil || relativePath == "." || strings.HasPrefix(relativePath, "..") {
		return false
	}
	firstSegment := strings.Split(filepath.ToSlash(relativePath), "/")[0]
	return strings.HasPrefix(firstSegment, "skill-packaging-")
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
		if shouldIgnorePackagingPath(relativePath) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
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

func shouldIgnorePackagingPath(relativePath string) bool {
	cleanPath := filepath.ToSlash(filepath.Clean(strings.TrimSpace(relativePath)))
	if cleanPath == "." || cleanPath == "" {
		return false
	}
	if cleanPath == importReportPath {
		return true
	}
	for _, segment := range strings.Split(cleanPath, "/") {
		if _, ok := packagingIgnoredDirectories[segment]; ok {
			return true
		}
	}
	return false
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

func extractUploadedArchive(archiveBase64 string, workspacePath string) error {
	trimmed := strings.TrimSpace(archiveBase64)
	if trimmed == "" {
		return catalog.NewError(422, "invalid_request", "source.archiveBase64 is required for source.type=upload_archive.", nil)
	}

	decoded, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return catalog.NewError(422, "invalid_request", "source.archiveBase64 must be valid base64.", nil)
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return catalog.NewError(422, "invalid_request", "source archive must be a valid gzip stream.", nil)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return catalog.NewError(422, "invalid_request", "source archive must be a valid tar stream.", nil)
		}

		relativePath := filepath.ToSlash(filepath.Clean(strings.TrimSpace(header.Name)))
		if relativePath == "." || relativePath == "" {
			continue
		}
		if shouldIgnorePackagingPath(relativePath) {
			continue
		}

		targetPath, resolveErr := resolveWorkspacePath(workspacePath, relativePath)
		if resolveErr != nil {
			return resolveErr
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return catalog.NewError(500, "internal_error", "Failed to materialize source archive directory.", nil)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return catalog.NewError(500, "internal_error", "Failed to materialize source archive file.", nil)
			}
			content, readErr := io.ReadAll(tarReader)
			if readErr != nil {
				return catalog.NewError(422, "invalid_request", "source archive contains unreadable file content.", map[string]any{"path": relativePath})
			}
			if err := os.WriteFile(targetPath, content, 0o644); err != nil {
				return catalog.NewError(500, "internal_error", "Failed to write extracted source archive file.", nil)
			}
		case tar.TypeSymlink, tar.TypeLink:
			return catalog.NewError(422, "unsafe_path_entry", "source archive must not contain symbolic links.", map[string]any{"path": relativePath})
		default:
			return catalog.NewError(422, "unsafe_path_entry", "source archive contains unsupported entry types.", map[string]any{"path": relativePath, "typeflag": header.Typeflag})
		}
	}

	return nil
}

func writeSkillMarkdown(workspacePath string, displayName string, description string) error {
	content := "---\nname: " + strings.TrimSpace(displayName) + "\ndescription: " + strings.TrimSpace(description) + "\n---\n"
	return os.WriteFile(filepath.Join(workspacePath, "SKILL.md"), []byte(content), 0o644)
}

func (m workspaceSourceModule) prepareGitSource(source Source, workspacePath string) error {
	location := strings.TrimSpace(source.Location)
	if location == "" {
		return catalog.NewError(422, "invalid_request", "source.location is required.", nil)
	}

	cloneURL, subdirectory := normalizeGitLocation(location)
	if source.Ref == "" && subdirectory == "" && isExistingDirectory(location) {
		return copySourceDirectory(location, workspacePath)
	}

	cloneDir, err := os.MkdirTemp(m.tempRoot, "skill-git-source-*")
	if err != nil {
		return catalog.NewError(500, "internal_error", "Failed to create git clone workspace.", nil)
	}
	defer os.RemoveAll(cloneDir)

	cloneArgs := []string{"clone"}
	if strings.TrimSpace(source.Ref) == "" {
		cloneArgs = append(cloneArgs, "--depth", "1")
	}
	cloneArgs = append(cloneArgs, cloneURL, cloneDir)
	if err := runGitCommand("", cloneArgs...); err != nil {
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
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout())
	defer cancel()

	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
	if output, err := command.CombinedOutput(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return catalog.NewError(500, "internal_error", "git command timed out.", map[string]any{"args": args, "timeoutMs": gitCommandTimeout().Milliseconds()})
		}
		return catalog.NewError(500, "internal_error", "git command failed.", map[string]any{"args": args, "output": strings.TrimSpace(string(output))})
	}
	return nil
}

func gitCommandTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("SKILL_REGISTRY_GIT_CLONE_TIMEOUT_MS"))
	if raw == "" {
		return defaultGitCloneTimeout
	}
	timeoutMs, err := strconv.Atoi(raw)
	if err != nil || timeoutMs <= 0 {
		return defaultGitCloneTimeout
	}
	return time.Duration(timeoutMs) * time.Millisecond
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
