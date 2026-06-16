package catalogapi

import (
	"strings"

	catalog "skill_registry/internal/catalog/domain"
)

func (s *Server) publicInstallTarget(response catalog.InstallResolution) catalog.InstallResolution {
	if s.publicNPMRegistryURL == "" {
		return response
	}

	internalRegistry := strings.TrimRight(strings.TrimSpace(response.NPMRegistry), "/")
	response.NPMRegistry = s.publicNPMRegistryURL
	response.TarballURL = rewriteRegistryBaseURL(response.TarballURL, internalRegistry, s.publicNPMRegistryURL)
	return response
}

func rewriteRegistryBaseURL(rawURL string, fromBase string, toBase string) string {
	trimmedURL := strings.TrimSpace(rawURL)
	normalizedFrom := strings.TrimRight(strings.TrimSpace(fromBase), "/")
	normalizedTo := strings.TrimRight(strings.TrimSpace(toBase), "/")

	switch {
	case trimmedURL == "":
		return rawURL
	case normalizedFrom == "":
		return trimmedURL
	case trimmedURL == normalizedFrom:
		return normalizedTo
	case strings.HasPrefix(trimmedURL, normalizedFrom+"/"):
		return normalizedTo + strings.TrimPrefix(trimmedURL, normalizedFrom)
	default:
		return trimmedURL
	}
}
