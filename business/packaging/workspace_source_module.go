package packaging

import (
	"time"

	packaginginfra "skill_registry/internal/packaging/infra"
)

type workspaceSourceModule struct {
	impl *packaginginfra.WorkspaceSourceModule
}

func newWorkspaceSourceModule(tempRoot string, workspaceTTL time.Duration) workspaceSourceModule {
	impl := packaginginfra.NewWorkspaceSourceModule(tempRoot, workspaceTTL)
	return workspaceSourceModule{impl: &impl}
}

func (m workspaceSourceModule) prepare(input PrepareRequest) (string, error) {
	return m.impl.Prepare(input)
}

func (m workspaceSourceModule) cleanup(workspacePath string) error {
	return m.impl.Cleanup(workspacePath)
}

func (m workspaceSourceModule) bundlePath(workspacePath string, target Target) (string, error) {
	return m.impl.BundlePath(workspacePath, target)
}

func (m workspaceSourceModule) pruneExpiredWorkspaces(ttl time.Duration) error {
	return m.impl.PruneExpiredWorkspaces(ttl)
}

func (m workspaceSourceModule) managesWorkspace(workspacePath string) bool {
	return m.impl.ManagesWorkspace(workspacePath)
}

func copySourceDirectory(sourcePath string, workspacePath string) error {
	return packaginginfra.CopySourceDirectory(sourcePath, workspacePath)
}

func shouldIgnorePackagingPath(relativePath string) bool {
	return packaginginfra.ShouldIgnorePackagingPath(relativePath)
}

func runGitCommand(directory string, args ...string) error {
	return packaginginfra.RunGitCommand(directory, args...)
}
