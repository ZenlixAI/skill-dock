package packaging

import packagingdomain "skill_registry/internal/packaging/domain"

func warningDetails(codes []string) []WarningDetail {
	return packagingdomain.WarningDetails(codes)
}
