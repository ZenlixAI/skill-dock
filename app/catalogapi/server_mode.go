package catalogapi

import "strings"

type ServerMode string

const (
	ServerModeAll   ServerMode = "all"
	ServerModeRead  ServerMode = "read"
	ServerModeWrite ServerMode = "write"
)

func normalizeServerMode(raw ServerMode) ServerMode {
	switch ServerMode(strings.ToLower(strings.TrimSpace(string(raw)))) {
	case ServerModeRead:
		return ServerModeRead
	case ServerModeWrite:
		return ServerModeWrite
	default:
		return ServerModeAll
	}
}

func (m ServerMode) servesReadRoutes() bool {
	return m == ServerModeAll || m == ServerModeRead
}

func (m ServerMode) servesWriteRoutes() bool {
	return m == ServerModeAll || m == ServerModeWrite
}
