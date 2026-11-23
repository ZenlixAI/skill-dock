package httpserver

import (
	"context"
	"errors"
	"net/http"

	"skill_registry/internal/access"
	catalog "skill_registry/internal/catalog/domain"
	"skill_registry/internal/platform/auth"
)

func (s *Server) isPrivateSkill(ctx context.Context, packageName string) bool {
	if s == nil || s.accessStore == nil {
		return false
	}
	record, err := s.accessStore.GetSkillAccess(ctx, packageName)
	if err != nil {
		return false
	}
	return record.Visibility == "private" || record.Visibility == "organization"
}

func (s *Server) checkSkillAccess(ctx context.Context, packageName string, action access.Action) error {
	if s == nil || s.access == nil || s.accessStore == nil {
		return nil
	}
	record, err := s.accessStore.GetSkillAccess(ctx, packageName)
	if errors.Is(err, access.ErrNotFound) {
		return nil
	}
	if err != nil {
		return catalog.NewError(http.StatusInternalServerError, "internal_error", "Internal server error.", nil)
	}
	decision := s.access.Decide(ctx, auth.PrincipalFromContext(ctx), action, record)
	if decision.Allowed {
		return nil
	}
	if action == access.ActionRead || action == access.ActionInstall {
		return catalog.NewError(http.StatusNotFound, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}
	return catalog.NewError(http.StatusForbidden, "forbidden", "Action is not allowed for this skill.", map[string]any{
		"packageName": packageName,
		"action":      string(action),
		"reason":      decision.Reason,
	})
}
