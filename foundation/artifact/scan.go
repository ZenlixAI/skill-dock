package artifact

import platformartifact "skill_registry/internal/platform/artifact"

type BundleScanResult = platformartifact.BundleScanResult

var (
	ScanNPMPackageTarballForTest      = platformartifact.ScanNPMPackageTarballForTest
	BundleDigestFromStringForTest     = platformartifact.BundleDigestFromStringForTest
	ExpectedPackageTarballPathForTest = platformartifact.ExpectedPackageTarballPathForTest
)
