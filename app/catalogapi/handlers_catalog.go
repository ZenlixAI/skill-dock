package catalogapi

import (
	"net/http"
	"strings"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/common"
	"skill_registry/business/catalog"
)

func (s *Server) handleSkills(writer http.ResponseWriter, request *http.Request) {
	if packageName := request.URL.Query().Get("package_name"); packageName != "" {
		if !s.authorizePackageAccess(writer, request, packageName) {
			return
		}
	}

	s.serveCachedReadJSON(writer, request, s.readListCacheTTL, func() (any, *catalog.Error) {
		return s.readAPIModule.listSkills(request.Context(), auth.PrincipalFromContext(request.Context()), request.URL.Query())
	})
}

func (s *Server) handleCuratedSkills(writer http.ResponseWriter, request *http.Request) {
	pageSize, err := common.ParsePageSize(request.URL.Query())
	if err != nil {
		common.WriteCatalogError(writer, request, err)
		return
	}
	cursor, err := common.ParseCursor(request.URL.Query())
	if err != nil {
		common.WriteCatalogError(writer, request, err)
		return
	}

	s.serveCachedReadJSON(writer, request, s.readListCacheTTL, func() (any, *catalog.Error) {
		return s.catalogReadModule.listCuratedSkills(request.Context(), auth.PrincipalFromContext(request.Context()), pageSize, cursor)
	})
}

func (s *Server) handleAuditEvents(writer http.ResponseWriter, request *http.Request) {
	packageName := request.URL.Query().Get("package_name")
	if packageName != "" && !s.authorizePackageAccess(writer, request, packageName) {
		return
	}

	s.serveCachedReadJSON(writer, request, s.readListCacheTTL, func() (any, *catalog.Error) {
		return s.auditQueryModule.ListEvents(request.Context(), auth.PrincipalFromContext(request.Context()), request.URL.Query())
	})
}

func (s *Server) handleGovernanceChanges(writer http.ResponseWriter, request *http.Request) {
	pageSize, err := common.ParsePageSize(request.URL.Query())
	if err != nil {
		common.WriteCatalogError(writer, request, err)
		return
	}
	cursor, err := common.ParseCursor(request.URL.Query())
	if err != nil {
		common.WriteCatalogError(writer, request, err)
		return
	}
	if packageName := request.URL.Query().Get("package_name"); packageName != "" {
		if !s.authorizePackageAccess(writer, request, packageName) {
			return
		}
	}
	targetType, targetTypeErr := common.ParseGovernanceTargetTypeFilter(request.URL.Query().Get("target_type"))
	if targetTypeErr != nil {
		common.WriteCatalogError(writer, request, targetTypeErr)
		return
	}
	versionFilter, versionFilterErr := common.ParseExactVersionQueryParam(request.URL.Query().Get("version"), "version")
	if versionFilterErr != nil {
		common.WriteCatalogError(writer, request, versionFilterErr)
		return
	}

	s.serveCachedReadJSON(writer, request, s.readListCacheTTL, func() (any, *catalog.Error) {
		return s.governanceModule.listChanges(request.Context(), auth.PrincipalFromContext(request.Context()), catalog.GovernanceQuery{
			PackageName: request.URL.Query().Get("package_name"),
			Version:     versionFilter,
			TargetType:  targetType,
			PageSize:    pageSize,
			Cursor:      cursor,
		})
	})
}

func (s *Server) handleSkillDetail(writer http.ResponseWriter, request *http.Request, packageName string) {
	s.serveCachedReadJSON(writer, request, s.readDetailCacheTTL, func() (any, *catalog.Error) {
		response, err := s.repo.GetSkill(request.Context(), packageName)
		if err != nil {
			return nil, common.CatalogErrorFromError(err)
		}
		return response, nil
	})
}

func (s *Server) handleVersionList(writer http.ResponseWriter, request *http.Request, packageName string) {
	includeDeprecated, includeDeprecatedErr := common.ParseBooleanQueryParam(request.URL.Query().Get("include_deprecated"), "include_deprecated", true)
	if includeDeprecatedErr != nil {
		common.WriteCatalogError(writer, request, includeDeprecatedErr)
		return
	}
	includeYanked, includeYankedErr := common.ParseBooleanQueryParam(request.URL.Query().Get("include_yanked"), "include_yanked", false)
	if includeYankedErr != nil {
		common.WriteCatalogError(writer, request, includeYankedErr)
		return
	}
	pageSize, err := common.ParsePageSize(request.URL.Query())
	if err != nil {
		common.WriteCatalogError(writer, request, err)
		return
	}
	cursor, err := common.ParseCursor(request.URL.Query())
	if err != nil {
		common.WriteCatalogError(writer, request, err)
		return
	}

	s.serveCachedReadJSON(writer, request, s.readDetailCacheTTL, func() (any, *catalog.Error) {
		response, err := s.repo.ListVersions(request.Context(), catalog.VersionQuery{
			PackageName:       packageName,
			IncludeDeprecated: includeDeprecated,
			IncludeYanked:     includeYanked,
			PageSize:          pageSize,
			Cursor:            cursor,
		})
		if err != nil {
			return nil, common.CatalogErrorFromError(err)
		}
		return response, nil
	})
}

func (s *Server) handleVersionDetail(writer http.ResponseWriter, request *http.Request, packageName string, version string) {
	decodedVersion, decodeErr := common.DecodeVersion(version)
	if decodeErr != nil {
		common.WriteCatalogError(writer, request, decodeErr)
		return
	}

	s.serveCachedReadJSON(writer, request, s.readDetailCacheTTL, func() (any, *catalog.Error) {
		response, err := s.repo.GetVersion(request.Context(), packageName, decodedVersion)
		if err != nil {
			return nil, common.CatalogErrorFromError(err)
		}
		return response, nil
	})
}

func (s *Server) handleInstallTarget(writer http.ResponseWriter, request *http.Request, packageName string) {
	s.serveCachedReadJSON(writer, request, s.readDetailCacheTTL, func() (any, *catalog.Error) {
		response, err := s.repo.ResolveInstallTarget(request.Context(), packageName, request.URL.Query().Get("version"))
		if err != nil {
			return nil, common.CatalogErrorFromError(err)
		}
		return s.publicInstallTarget(response), nil
	})
}

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
