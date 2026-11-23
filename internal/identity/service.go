package identity

import (
	"context"
	"errors"
	"strings"

	"skill_registry/internal/platform/auth"
)

type Service struct {
	verifier *Verifier
	store    Store
}

func NewService(verifier *Verifier, store Store) *Service {
	return &Service{verifier: verifier, store: store}
}

func (s *Service) AuthenticateBearer(ctx context.Context, token string) (auth.Principal, bool, error) {
	if s == nil || s.verifier == nil || s.store == nil || strings.TrimSpace(token) == "" {
		return auth.Principal{}, false, nil
	}
	claims, err := s.verifier.Verify(ctx, token)
	if err != nil {
		return auth.Principal{}, true, err
	}

	user, err := s.store.GetUserBySubject(ctx, claims.Issuer, claims.Subject)
	if errors.Is(err, ErrNotFound) && strings.TrimSpace(claims.Email) != "" {
		if existing, emailErr := s.store.GetUserByEmail(ctx, claims.Email); emailErr == nil {
			user, err = s.store.UpdateUserIdentity(ctx, existing.ID, claims.Issuer, claims.Subject)
		}
	}
	if errors.Is(err, ErrNotFound) || user.ID == "" {
		user, err = s.store.UpsertUser(ctx, User{
			Issuer:      claims.Issuer,
			Subject:     claims.Subject,
			Email:       claims.Email,
			DisplayName: firstNonEmpty(claims.Name, claims.Email, claims.Subject),
			AvatarURL:   claims.Picture,
			Status:      "active",
		})
	}
	if err != nil {
		return auth.Principal{}, true, err
	}

	allOrganizations := uniqueStrings(append(append([]string{}, claims.Organizations...), claims.OrganizationID))
	for _, logtoOrgID := range allOrganizations {
		if strings.TrimSpace(logtoOrgID) == "" {
			continue
		}
		organization, err := s.store.GetOrganizationByLogtoID(ctx, logtoOrgID)
		if errors.Is(err, ErrNotFound) {
			organization, err = s.store.UpsertOrganization(ctx, Organization{
				LogtoOrgID: logtoOrgID,
				Slug:       sanitizeSlug(logtoOrgID),
				Name:       logtoOrgID,
				Status:     "active",
			})
		}
		if err != nil {
			return auth.Principal{}, true, err
		}
		roles := rolesForOrganization(claims.OrganizationRoles, logtoOrgID)
		if err := s.store.UpsertMembership(ctx, Membership{
			OrganizationID: organization.ID,
			UserID:         user.ID,
			Roles:          roles,
		}); err != nil {
			return auth.Principal{}, true, err
		}
	}

	principal := auth.Principal{
		Kind:          auth.PrincipalKindUser,
		UserID:        user.ID,
		Subject:       claims.Subject,
		Role:          "",
		Actor:         firstNonEmpty(user.DisplayName, user.Email, user.Subject),
		AuthMechanism: "logto_jwt",
		Scopes:        splitScope(claims.Scope),
		DisplayName:   user.DisplayName,
		Email:         user.Email,
		ClientID:      claims.ClientID,
		PlatformAdmin: hasScope(claims.Scope, "platform:admin") || contains(claims.Roles, "platform-admin"),
	}
	if claims.OrganizationID != "" {
		organization, err := s.store.GetOrganizationByLogtoID(ctx, claims.OrganizationID)
		if err == nil {
			principal.TenantID = organization.ID
			principal.TenantSlug = organization.Slug
		}
	}
	return principal, true, nil
}

func (s *Service) Me(ctx context.Context, principal auth.Principal) (Me, error) {
	if s == nil || s.store == nil {
		return Me{}, errors.New("identity store is not configured")
	}
	user, err := s.store.GetUserByID(ctx, principal.UserID)
	if err != nil {
		return Me{}, err
	}
	organizations, err := s.store.ListOrganizationsForUser(ctx, principal.UserID)
	if err != nil {
		return Me{}, err
	}
	if organizations == nil {
		organizations = []OrganizationRole{}
	}
	var active *Organization
	for index := range organizations {
		if organizations[index].ID == principal.TenantID {
			copy := organizations[index].Organization
			active = &copy
			break
		}
	}
	scopes := append([]string(nil), principal.Scopes...)
	if scopes == nil {
		scopes = []string{}
	}
	return Me{
		User:               user,
		ActiveOrganization: active,
		Organizations:      organizations,
		Scopes:             scopes,
		Personal:           PersonalContext{DraftsEnabled: true},
	}, nil
}

func splitScope(scope string) []string {
	if strings.TrimSpace(scope) == "" {
		return []string{}
	}
	return strings.Fields(scope)
}

func hasScope(scope string, wanted string) bool {
	for _, item := range splitScope(scope) {
		if item == wanted {
			return true
		}
	}
	return false
}

func rolesForOrganization(roles []string, logtoOrgID string) []string {
	prefix := logtoOrgID + ":"
	result := []string{}
	for _, role := range roles {
		if strings.HasPrefix(role, prefix) {
			result = append(result, strings.TrimPrefix(role, prefix))
		}
	}
	return result
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func sanitizeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "organization"
	}
	var builder strings.Builder
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(char)
		case char >= '0' && char <= '9':
			builder.WriteRune(char)
		case char == '-' || char == '_':
			builder.WriteRune(char)
		default:
			builder.WriteRune('-')
		}
	}
	return strings.Trim(builder.String(), "-")
}
