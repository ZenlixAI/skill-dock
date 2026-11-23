package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"skill_registry/internal/access"
	catalog "skill_registry/internal/catalog/domain"
	"skill_registry/internal/platform/auth"
)

func (s *Server) ensurePublishAccess(ctx context.Context, principal auth.Principal, packageName string, visibility string) (func(), error) {
	if s == nil || s.accessStore == nil || s.access == nil {
		return func() {}, nil
	}
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return func() {}, catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "packageName is required.", nil)
	}
	visibility = normalizeVisibility(visibility, principal)

	existing, err := s.accessStore.GetSkillAccess(ctx, packageName)
	if err == nil {
		if decision := s.access.Decide(ctx, principal, access.ActionPublish, existing); !decision.Allowed {
			return func() {}, catalog.NewError(http.StatusForbidden, "namespace_forbidden", "Publisher is not allowed to publish this skill.", map[string]any{
				"packageName": packageName,
				"reason":      decision.Reason,
			})
		}
		return func() {}, nil
	}
	if !errors.Is(err, access.ErrNotFound) {
		return func() {}, catalog.NewError(http.StatusInternalServerError, "internal_error", "Internal server error.", nil)
	}

	record := access.SkillAccess{
		PackageName: packageName,
		OwnerType:   "platform",
		Visibility:  visibility,
	}
	namespace, namespaceErr := s.accessStore.GetNamespaceForPackage(ctx, packageName)
	switch {
	case namespaceErr == nil:
		if principal.IsUser() && principal.TenantID != namespace.OrganizationID {
			return func() {}, catalog.NewError(http.StatusForbidden, "namespace_forbidden", "Package does not belong to the active tenant.", map[string]any{"packageName": packageName})
		}
		if visibility == "public" && namespace.Status != "verified" {
			return func() {}, catalog.NewError(http.StatusConflict, "namespace_unverified", "Namespace is not verified for public publishing.", map[string]any{"packageName": packageName})
		}
		record.OwnerType = "organization"
		record.OwnerID = namespace.OrganizationID
		record.OrganizationID = namespace.OrganizationID
	case errors.Is(namespaceErr, access.ErrNotFound):
		if visibility == "public" {
			if principal.IsUser() {
				return func() {}, catalog.NewError(http.StatusForbidden, "namespace_forbidden", "Public skills require a verified organization namespace.", map[string]any{"packageName": packageName})
			}
			record.OwnerType = "platform"
			record.Visibility = "public"
		} else if principal.IsUser() {
			if !strings.HasPrefix(packageName, "@draft."+principal.UserID+"/") {
				return func() {}, catalog.NewError(http.StatusForbidden, "namespace_forbidden", "Personal drafts must use the reserved @draft.<user-id>/ namespace.", map[string]any{"packageName": packageName})
			}
			record.OwnerType = "user"
			record.OwnerID = principal.UserID
			record.Visibility = "private"
		}
	default:
		return func() {}, catalog.NewError(http.StatusInternalServerError, "internal_error", "Internal server error.", nil)
	}

	if record.Visibility != "public" && record.OrganizationID != "" {
		config, configErr := s.accessStore.GetArtifactConfig(ctx, record.OrganizationID)
		if configErr != nil {
			if s.privateNPMRegistryURL == "" {
				return func() {}, catalog.NewError(http.StatusConflict, "private_artifact_scope_required", "A private artifact target is required for this tenant.", map[string]any{"packageName": packageName})
			}
		} else if config.ScopePrefix != "" && !strings.HasPrefix(packageName, config.ScopePrefix) {
			return func() {}, catalog.NewError(http.StatusForbidden, "namespace_forbidden", "Package is outside the tenant's private artifact scope.", map[string]any{
				"packageName": packageName,
				"scopePrefix": config.ScopePrefix,
			})
		}
	}

	if err := s.accessStore.PutSkillAccess(ctx, record); err != nil {
		return func() {}, catalog.NewError(http.StatusInternalServerError, "internal_error", "Internal server error.", nil)
	}
	return func() {
		_ = s.accessStore.DeleteSkillAccess(context.Background(), packageName)
	}, nil
}

func normalizeVisibility(value string, principal auth.Principal) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "public", "organization", "private":
		return strings.ToLower(strings.TrimSpace(value))
	}
	if principal.IsUser() {
		return "private"
	}
	return "public"
}
