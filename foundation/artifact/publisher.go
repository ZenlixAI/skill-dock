package artifact

import platformartifact "skill_registry/internal/platform/artifact"

type NPMPublisher = platformartifact.NPMPublisher
type NPMPublishInput = platformartifact.NPMPublishInput

var NewNPMPublisher = platformartifact.NewNPMPublisher

func buildNPMPublishPayload(baseURL string, input NPMPublishInput) ([]byte, error) {
	return platformartifact.BuildNPMPublishPayloadForTest(baseURL, input)
}
