package packaging

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"skill_registry/business/catalog"
	"skill_registry/foundation/persist"
)

func (s *Service) persistWorkspaceSnapshot(ctx context.Context, workspacePath string) (string, error) {
	if err := packagingContextError(ctx); err != nil {
		return "", err
	}
	snapshot, err := archiveWorkspace(ctx, workspacePath)
	if err != nil {
		return "", catalog.NewError(500, "internal_error", "Failed to persist packaging workspace.", nil)
	}
	workspaceID := workspaceHandle()
	if err := s.workspaceStore.SaveWorkspace(ctx, workspaceID, snapshot, time.Now().UTC()); err != nil {
		return "", catalog.NewError(500, "internal_error", "Failed to persist packaging workspace.", nil)
	}
	return workspaceID, nil
}

func (s *Service) materializeWorkspace(ctx context.Context, workspacePath string) (string, func(), error) {
	if err := packagingContextError(ctx); err != nil {
		return "", nil, err
	}
	if s.workspaceStore == nil {
		return workspacePath, func() {}, nil
	}

	record, err := s.workspaceStore.LoadWorkspace(ctx, workspacePath)
	if err != nil {
		if err == persist.ErrWorkspaceRecordNotFound {
			return "", nil, catalog.NewError(422, "invalid_request", "workspacePath must reference an existing packaging workspace.", map[string]any{"workspacePath": workspacePath})
		}
		return "", nil, catalog.NewError(500, "internal_error", "Failed to load packaging workspace.", nil)
	}

	tempWorkspace, err := os.MkdirTemp(s.workspaceSource.tempRoot, "skill-packaging-materialized-*")
	if err != nil {
		return "", nil, catalog.NewError(500, "internal_error", "Failed to materialize packaging workspace.", nil)
	}
	if err := extractWorkspaceArchive(ctx, record.Snapshot, tempWorkspace); err != nil {
		_ = os.RemoveAll(tempWorkspace)
		return "", nil, err
	}

	return tempWorkspace, func() { _ = os.RemoveAll(tempWorkspace) }, nil
}

func archiveWorkspace(ctx context.Context, workspacePath string) ([]byte, error) {
	var compressed bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressed)
	tarWriter := tar.NewWriter(gzipWriter)

	walkErr := filepath.WalkDir(workspacePath, func(path string, entry os.DirEntry, err error) error {
		if ctxErr := packagingContextError(ctx); ctxErr != nil {
			return ctxErr
		}
		if err != nil {
			return err
		}
		if path == workspacePath {
			return nil
		}
		relativePath, err := filepath.Rel(workspacePath, path)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(relativePath)
		if shouldIgnoreWorkspaceSnapshotPath(relativePath) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relativePath
		if entry.IsDir() && !strings.HasSuffix(header.Name, "/") {
			header.Name += "/"
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		if ctxErr := packagingContextError(ctx); ctxErr != nil {
			return ctxErr
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = tarWriter.Write(content)
		return err
	})
	if walkErr != nil {
		_ = tarWriter.Close()
		_ = gzipWriter.Close()
		return nil, walkErr
	}
	if err := tarWriter.Close(); err != nil {
		_ = gzipWriter.Close()
		return nil, err
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}
	return compressed.Bytes(), nil
}

func extractWorkspaceArchive(ctx context.Context, archive []byte, workspacePath string) error {
	gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return catalog.NewError(500, "internal_error", "Persisted packaging workspace is unreadable.", nil)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		if ctxErr := packagingContextError(ctx); ctxErr != nil {
			return ctxErr
		}
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return catalog.NewError(500, "internal_error", "Persisted packaging workspace is unreadable.", nil)
		}

		targetPath, resolveErr := resolveWorkspacePath(workspacePath, header.Name)
		if resolveErr != nil {
			return resolveErr
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return catalog.NewError(500, "internal_error", "Failed to materialize packaging workspace.", nil)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return catalog.NewError(500, "internal_error", "Failed to materialize packaging workspace.", nil)
			}
			if ctxErr := packagingContextError(ctx); ctxErr != nil {
				return ctxErr
			}
			content, err := io.ReadAll(tarReader)
			if err != nil {
				return catalog.NewError(500, "internal_error", "Failed to materialize packaging workspace.", nil)
			}
			if ctxErr := packagingContextError(ctx); ctxErr != nil {
				return ctxErr
			}
			if err := os.WriteFile(targetPath, content, 0o644); err != nil {
				return catalog.NewError(500, "internal_error", "Failed to materialize packaging workspace.", nil)
			}
		default:
			return catalog.NewError(500, "internal_error", "Persisted packaging workspace contains an unsupported entry.", nil)
		}
	}
}

func shouldIgnoreWorkspaceSnapshotPath(relativePath string) bool {
	cleanPath := filepath.ToSlash(filepath.Clean(strings.TrimSpace(relativePath)))
	if cleanPath == "." || cleanPath == "" {
		return false
	}
	for _, segment := range strings.Split(cleanPath, "/") {
		if segment == ".dist" {
			return true
		}
	}
	return false
}

func workspaceHandle() string {
	dir, _ := os.MkdirTemp("", "ws-")
	if dir == "" {
		return "workspace-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	_ = os.RemoveAll(dir)
	return filepath.Base(dir)
}

func workspaceBundleHandle(workspacePath string, target Target) string {
	return "workspace://" + workspacePath + "/" + packageBasename(target.PackageName) + "-" + target.Version + ".tgz"
}
