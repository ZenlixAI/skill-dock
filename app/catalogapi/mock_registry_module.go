package catalogapi

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	artifactadapter "skill_registry/foundation/artifact"
)

type mockRegistryWorkflow interface {
	handleRequest(writer http.ResponseWriter, request *http.Request)
}

type mockRegistryModule struct {
	registry *artifactadapter.MockRegistry
}

func (m *mockRegistryModule) handleRequest(writer http.ResponseWriter, request *http.Request) {
	if m.registry == nil {
		http.NotFound(writer, request)
		return
	}

	remainder := strings.TrimPrefix(request.URL.EscapedPath(), "/mock-npm/")
	if strings.Contains(remainder, "/-/") {
		m.handleTarballRequest(writer, request, remainder)
		return
	}

	packageName, err := url.PathUnescape(remainder)
	if err != nil {
		http.NotFound(writer, request)
		return
	}

	switch request.Method {
	case http.MethodGet:
		payload, found := m.registry.PackageMetadata(mockRegistryBaseURL(request), packageName)
		if !found {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(payload)
	case http.MethodPut:
		body, readErr := io.ReadAll(request.Body)
		if readErr != nil {
			http.Error(writer, "failed to read request body", http.StatusBadRequest)
			return
		}
		if !m.registry.PublishPackage(packageName, body) {
			http.Error(writer, "invalid publish payload", http.StatusBadRequest)
			return
		}
		writer.WriteHeader(http.StatusCreated)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *mockRegistryModule) handleTarballRequest(writer http.ResponseWriter, request *http.Request, remainder string) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	packagePart, filePart, ok := strings.Cut(remainder, "/-/")
	if !ok {
		http.NotFound(writer, request)
		return
	}
	packageName := "@" + packagePart
	version := strings.TrimSuffix(strings.TrimPrefix(filePart, packageBasename(packageName)+"-"), ".tgz")
	tarball, found := m.registry.Tarball(packageName, version)
	if !found {
		http.NotFound(writer, request)
		return
	}
	writer.Header().Set("Content-Type", "application/octet-stream")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(tarball)
}
