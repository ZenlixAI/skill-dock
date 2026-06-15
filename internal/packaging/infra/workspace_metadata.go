package infra

import (
	"encoding/json"
	"os"
	"path/filepath"

	"skill_registry/business/catalog"
)

type workspacePackageMetadata struct {
	Description string `json:"description"`
	Skill       struct {
		DisplayName string             `json:"displayName"`
		SourceType  catalog.SourceType `json:"sourceType"`
	} `json:"skill"`
}

type WorkspacePackageMetadata = workspacePackageMetadata

func ReadWorkspacePackageMetadataFile(workspaceHandle string, workspacePath string) (WorkspacePackageMetadata, error) {
	return readWorkspacePackageMetadataFile(workspaceHandle, workspacePath)
}

func DecodeWorkspacePackageMetadataJSON(workspaceHandle string, content []byte) (map[string]any, error) {
	return decodeWorkspacePackageMetadataJSON(workspaceHandle, content)
}

func readWorkspacePackageMetadataFile(workspaceHandle string, workspacePath string) (workspacePackageMetadata, error) {
	content, err := readWorkspacePackageMetadataContent(workspaceHandle, workspacePath)
	if err != nil {
		return workspacePackageMetadata{}, err
	}

	var payload workspacePackageMetadata
	if err := json.Unmarshal(content, &payload); err != nil {
		return workspacePackageMetadata{}, catalog.NewError(422, "invalid_request", "workspace package.json is invalid JSON.", map[string]any{
			"workspacePath": workspaceHandle,
		})
	}
	return payload, nil
}

func readWorkspacePackageMetadataContent(workspaceHandle string, workspacePath string) ([]byte, error) {
	content, err := os.ReadFile(filepath.Join(workspacePath, "package.json"))
	if err != nil {
		return nil, catalog.NewError(422, "invalid_request", "workspace package.json could not be read.", map[string]any{
			"workspacePath": workspaceHandle,
		})
	}
	return content, nil
}

func decodeWorkspacePackageMetadataJSON(workspaceHandle string, content []byte) (map[string]any, error) {
	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		return nil, catalog.NewError(422, "invalid_request", "workspace package.json is invalid JSON.", map[string]any{
			"workspacePath": workspaceHandle,
		})
	}
	return payload, nil
}
