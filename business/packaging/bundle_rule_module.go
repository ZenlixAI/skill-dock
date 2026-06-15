package packaging

import packagingdomain "skill_registry/internal/packaging/domain"

type preparedWorkspaceOutcome = packagingdomain.PreparedWorkspaceOutcome
type validationResult = packagingdomain.ValidationResult

type bundleRuleModule struct {
	impl *packagingdomain.BundleRules
}

func newBundleRuleModule() bundleRuleModule {
	impl := packagingdomain.NewBundleRules()
	return bundleRuleModule{impl: &impl}
}

func (m bundleRuleModule) prepareWorkspace(workspacePath string, input PrepareRequest) (preparedWorkspaceOutcome, error) {
	return m.impl.PrepareWorkspace(workspacePath, input)
}

func (m bundleRuleModule) validate(workspacePath string, target Target) (validationResult, error) {
	return m.impl.Validate(workspacePath, target)
}

func (m bundleRuleModule) buildBundle(workspacePath string, bundlePath string, includeBundle bool) (string, error) {
	return m.impl.BuildBundle(workspacePath, bundlePath, includeBundle)
}

const importReportPath = packagingdomain.ImportReportPath

func writePackageJSON(workspacePath string, target Target, metadata Metadata, options Options, normalizedFromImport bool) error {
	return packagingdomain.WritePackageJSON(workspacePath, target, metadata, options, normalizedFromImport)
}

func validateTarget(target Target) error {
	return packagingdomain.ValidateTarget(target)
}

func createBundleArchive(workspacePath string, bundlePath string) error {
	return packagingdomain.CreateBundleArchive(workspacePath, bundlePath)
}
