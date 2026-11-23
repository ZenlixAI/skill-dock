package identity

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("identity not found")

type Store interface {
	UpsertUser(ctx context.Context, user User) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)
	GetUserBySubject(ctx context.Context, issuer string, subject string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	UpdateUserIdentity(ctx context.Context, userID string, issuer string, subject string) (User, error)
	ListUsers(ctx context.Context, query string, limit int) ([]User, error)

	UpsertOrganization(ctx context.Context, organization Organization) (Organization, error)
	GetOrganizationByID(ctx context.Context, id string) (Organization, error)
	GetOrganizationByLogtoID(ctx context.Context, logtoOrgID string) (Organization, error)
	GetOrganizationBySlug(ctx context.Context, slug string) (Organization, error)
	ListOrganizationsForUser(ctx context.Context, userID string) ([]OrganizationRole, error)

	UpsertMembership(ctx context.Context, membership Membership) error
	RemoveMembership(ctx context.Context, organizationID string, userID string) error
	IsMember(ctx context.Context, organizationID string, userID string) (bool, error)
}
