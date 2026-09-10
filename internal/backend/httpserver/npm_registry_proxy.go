package httpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	neturl "net/url"
	"strconv"
	"strings"
)

const npmRegistryMountPath = "/repository/skills-npm"

func newNPMRegistryProxy(targetBaseURL string, publicBaseURL string, readAuthorization string, publishAuthorization string) http.Handler {
	targetBaseURL = strings.TrimSpace(targetBaseURL)
	publicBaseURL = strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")
	if targetBaseURL == "" {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			http.Error(writer, "Catalog backend is not configured.", http.StatusServiceUnavailable)
		})
	}

	target, err := neturl.Parse(targetBaseURL)
	if err != nil {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			http.Error(writer, fmt.Sprintf("Invalid Catalog backend URL: %v", err), http.StatusServiceUnavailable)
		})
	}

	proxy := &httputil.ReverseProxy{
		Director: func(request *http.Request) {
			request.Header.Set("Accept-Encoding", "identity")
			// Never forward the caller's Catalog token to the artifact repository.
			request.Header.Del("Authorization")
			switch request.Method {
			case http.MethodPut, http.MethodPost, http.MethodPatch, http.MethodDelete:
				if publishAuthorization != "" {
					request.Header.Set("Authorization", publishAuthorization)
				}
			default:
				if readAuthorization != "" {
					request.Header.Set("Authorization", readAuthorization)
				}
			}
			request.URL.Scheme = target.Scheme
			request.URL.Host = target.Host
			request.Host = target.Host

			pathSuffix := strings.TrimPrefix(request.URL.Path, npmRegistryMountPath)
			rawPathSuffix := strings.TrimPrefix(request.URL.RawPath, npmRegistryMountPath)

			request.URL.Path = joinRegistryProxyPath(target.Path, pathSuffix)
			if rawPathSuffix != "" {
				request.URL.RawPath = joinRegistryProxyPath(target.EscapedPath(), rawPathSuffix)
			} else {
				request.URL.RawPath = ""
			}
		},
		ModifyResponse: func(response *http.Response) error {
			if response == nil || response.Body == nil {
				return nil
			}
			if publicBaseURL == "" {
				return nil
			}
			body, err := io.ReadAll(response.Body)
			if err != nil {
				return err
			}
			_ = response.Body.Close()
			rewritten := rewriteRegistryTarballBase(body, publicBaseURL, target.Path)
			response.Body = io.NopCloser(bytes.NewReader(rewritten))
			response.ContentLength = int64(len(rewritten))
			response.Header.Set("Content-Length", strconv.Itoa(len(rewritten)))
			response.Header.Del("Content-Encoding")
			return nil
		},
		ErrorHandler: func(writer http.ResponseWriter, request *http.Request, err error) {
			http.Error(writer, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		},
	}
	return proxy
}

func rewriteRegistryTarballBase(body []byte, publicBaseURL string, upstreamBasePath string) []byte {
	publicBase := strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")
	if publicBase == "" {
		return body
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}
	rewriteTarballURLs(payload, publicBase, upstreamBasePath)
	rewritten, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return rewritten
}

func rewriteTarballURLs(node any, publicBase string, upstreamBasePath string) {
	switch value := node.(type) {
	case map[string]any:
		for key, child := range value {
			if key == "tarball" {
				if tarball, ok := child.(string); ok {
					value[key] = rewriteTarballURL(tarball, publicBase, upstreamBasePath)
				}
				continue
			}
			rewriteTarballURLs(child, publicBase, upstreamBasePath)
		}
	case []any:
		for _, child := range value {
			rewriteTarballURLs(child, publicBase, upstreamBasePath)
		}
	}
}

func rewriteTarballURL(tarballURL string, publicBase string, upstreamBasePath string) string {
	parsed, err := neturl.Parse(strings.TrimSpace(tarballURL))
	if err != nil || parsed.Path == "" {
		return tarballURL
	}
	path := stripRegistryBasePath(parsed.Path, upstreamBasePath)
	rewritten := strings.TrimRight(publicBase, "/") + path
	if parsed.RawQuery != "" {
		rewritten += "?" + parsed.RawQuery
	}
	if parsed.Fragment != "" {
		rewritten += "#" + parsed.Fragment
	}
	return rewritten
}

func stripRegistryBasePath(path string, basePath string) string {
	basePath = strings.TrimRight(basePath, "/")
	if basePath == "" || basePath == "/" {
		return path
	}
	if path == basePath {
		return "/"
	}
	if relativePath, ok := strings.CutPrefix(path, basePath+"/"); ok {
		return "/" + relativePath
	}
	return path
}

func joinRegistryProxyPath(basePath string, suffix string) string {
	switch {
	case basePath == "" && suffix == "":
		return "/"
	case basePath == "":
		if strings.HasPrefix(suffix, "/") {
			return suffix
		}
		return "/" + suffix
	case suffix == "":
		return basePath
	case strings.HasSuffix(basePath, "/"):
		return basePath + strings.TrimPrefix(suffix, "/")
	case strings.HasPrefix(suffix, "/"):
		return basePath + suffix
	default:
		return basePath + "/" + suffix
	}
}
