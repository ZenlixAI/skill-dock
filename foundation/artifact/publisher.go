package artifact

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"skill_registry/business/catalog"
)

type NPMPublisher struct {
	baseURL       string
	authorization string
	client        *http.Client
}

type NPMPublishInput struct {
	PackageName string
	Version     string
	Description string
	DisplayName string
	SourceType  catalog.SourceType
	Tarball     []byte
}

func NewNPMPublisher(baseURL string, authorization string, client *http.Client) *NPMPublisher {
	if client == nil {
		client = http.DefaultClient
	}
	return &NPMPublisher{
		baseURL:       strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		authorization: strings.TrimSpace(authorization),
		client:        client,
	}
}

func (p *NPMPublisher) PublishPackage(ctx context.Context, input NPMPublishInput) error {
	if p == nil || p.baseURL == "" {
		return catalog.NewError(503, "registry_publish_unavailable", "Registry publish target is not configured.", map[string]any{
			"packageName": input.PackageName,
			"version":     input.Version,
		})
	}

	requestBody, err := buildNPMPublishPayload(p.baseURL, input)
	if err != nil {
		return catalog.NewError(500, "internal_error", "Registry publish payload could not be built.", map[string]any{
			"packageName": input.PackageName,
			"version":     input.Version,
		})
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		p.baseURL+"/"+url.PathEscape(input.PackageName),
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return catalog.NewError(503, "registry_publish_unavailable", "Registry publish request could not be created.", map[string]any{
			"packageName": input.PackageName,
			"version":     input.Version,
		})
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	if p.authorization != "" {
		request.Header.Set("Authorization", p.authorization)
	}

	response, err := p.client.Do(request)
	if err != nil {
		return catalog.NewError(503, "registry_unavailable", "Registry is unavailable.", map[string]any{
			"packageName": input.PackageName,
			"version":     input.Version,
		})
	}
	defer response.Body.Close()

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}

	responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	details := map[string]any{
		"packageName": input.PackageName,
		"version":     input.Version,
		"status":      response.StatusCode,
	}
	if trimmed := strings.TrimSpace(string(responseBody)); trimmed != "" {
		details["responseBody"] = trimmed
	}
	if response.StatusCode == http.StatusConflict || response.StatusCode == http.StatusBadRequest {
		return catalog.NewError(409, "registry_version_conflict", "Registry already contains this package version or rejected an overwrite.", details)
	}
	return catalog.NewError(503, "registry_publish_failed", "Registry publish request failed.", details)
}

func buildNPMPublishPayload(baseURL string, input NPMPublishInput) ([]byte, error) {
	integrity, shasum := digestTarball(input.Tarball)
	attachmentName := packageAttachmentName(input.PackageName, input.Version)
	tarballURL := strings.TrimRight(baseURL, "/") + "/" + strings.TrimPrefix(input.PackageName, "@") + "/-/" + packageBasename(input.PackageName) + "-" + input.Version + ".tgz"

	versionPayload := map[string]any{
		"name":        input.PackageName,
		"version":     input.Version,
		"description": strings.TrimSpace(input.Description),
		"dist": map[string]any{
			"integrity": integrity,
			"shasum":    shasum,
			"tarball":   tarballURL,
		},
	}
	skillPayload := map[string]any{}
	if strings.TrimSpace(input.DisplayName) != "" {
		skillPayload["displayName"] = strings.TrimSpace(input.DisplayName)
	}
	if strings.TrimSpace(string(input.SourceType)) != "" {
		skillPayload["sourceType"] = input.SourceType
	}
	if len(skillPayload) > 0 {
		versionPayload["skill"] = skillPayload
	}

	payload := map[string]any{
		"_id":         input.PackageName,
		"name":        input.PackageName,
		"description": strings.TrimSpace(input.Description),
		"dist-tags": map[string]string{
			"latest": input.Version,
		},
		"versions": map[string]any{
			input.Version: versionPayload,
		},
		"_attachments": map[string]any{
			attachmentName: map[string]any{
				"content_type": "application/octet-stream",
				"data":         base64.StdEncoding.EncodeToString(input.Tarball),
				"length":       len(input.Tarball),
			},
		},
	}

	return json.Marshal(payload)
}

func digestTarball(content []byte) (string, string) {
	sha1Sum := sha1.Sum(content)
	sha512Sum := sha512.Sum512(content)
	return "sha512-" + base64.StdEncoding.EncodeToString(sha512Sum[:]), hex.EncodeToString(sha1Sum[:])
}

func packageAttachmentName(packageName string, version string) string {
	normalized := strings.TrimPrefix(packageName, "@")
	normalized = strings.ReplaceAll(normalized, "/", "-")
	return normalized + "-" + version + ".tgz"
}
