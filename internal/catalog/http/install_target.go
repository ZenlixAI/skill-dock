package http

import (
	"net/http"
	"net/url"
	"strings"

	catalog "skill_registry/internal/catalog/domain"
)

const installProxyPath = "/repository/skills-npm"

func RewriteInstallTargetToProxy(request *http.Request, response catalog.InstallResolution) catalog.InstallResolution {
	if request == nil || strings.TrimSpace(response.TarballURL) == "" {
		return response
	}
	from, err := url.Parse(response.TarballURL)
	if err != nil {
		return response
	}

	internalRegistryPath := ""
	if internalRegistry, parseErr := url.Parse(strings.TrimSpace(response.NPMRegistry)); parseErr == nil {
		internalRegistryPath = internalRegistry.Path
	}

	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	proxyBase := strings.TrimRight(scheme+"://"+request.Host+installProxyPath, "/")
	proxyPath := rewriteProxyPath(from.Path, internalRegistryPath)
	proxyRawPath := rewriteProxyPath(from.RawPath, internalRegistryPath)

	proxy := *request.URL
	proxy.Scheme = scheme
	proxy.Host = request.Host
	proxy.Path = proxyPath
	proxy.RawPath = proxyRawPath
	proxy.RawQuery = from.RawQuery
	response.NPMRegistry = proxyBase
	response.TarballURL = proxy.String()
	return response
}

func rewriteProxyPath(path string, internalBasePath string) string {
	path = strings.TrimSpace(path)
	internalBasePath = normalizeRegistryPath(internalBasePath)
	if internalBasePath != "" {
		if path == internalBasePath {
			path = ""
		} else if strings.HasPrefix(path, internalBasePath+"/") {
			path = strings.TrimPrefix(path, internalBasePath)
		}
	}

	switch {
	case path == "":
		return installProxyPath
	case path == installProxyPath || strings.HasPrefix(path, installProxyPath+"/"):
		return path
	case strings.HasPrefix(path, "/"):
		return installProxyPath + path
	default:
		return installProxyPath + "/" + path
	}
}

func normalizeRegistryPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return strings.TrimRight(trimmed, "/")
}

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
