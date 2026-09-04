package httpserver

import "strings"

type ServerMode string

type InstallTargetMode string

const (
	ServerModeAll   ServerMode = "all"
	ServerModeRead  ServerMode = "read"
	ServerModeWrite ServerMode = "write"

	InstallTargetModeProxy  InstallTargetMode = "proxy"
	InstallTargetModeDirect InstallTargetMode = "direct"
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

func normalizeInstallTargetMode(raw InstallTargetMode) InstallTargetMode {
	switch InstallTargetMode(strings.ToLower(strings.TrimSpace(string(raw)))) {
	case InstallTargetModeDirect:
		return InstallTargetModeDirect
	default:
		return InstallTargetModeProxy
	}
}

func (m ServerMode) servesReadRoutes() bool {
	return m == ServerModeAll || m == ServerModeRead
}

func (m ServerMode) servesWriteRoutes() bool {
	return m == ServerModeAll || m == ServerModeWrite
}
