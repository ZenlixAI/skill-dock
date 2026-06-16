package http

import (
	"strings"

	catalog "skill_registry/internal/catalog/domain"
)

func RewritePublicInstallTarget(response catalog.InstallResolution, publicRegistryURL string) catalog.InstallResolution {
	if strings.TrimSpace(publicRegistryURL) == "" {
		return response
	}

	internalRegistry := strings.TrimRight(strings.TrimSpace(response.NPMRegistry), "/")
	publicRegistry := strings.TrimRight(strings.TrimSpace(publicRegistryURL), "/")
	response.NPMRegistry = publicRegistry
	response.TarballURL = rewriteRegistryBaseURL(response.TarballURL, internalRegistry, publicRegistry)
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
