package access

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("access record not found")

type SkillAccess struct {
	PackageName    string
	OrganizationID string
	OwnerType      string
	OwnerID        string
	Visibility     string
	AccessRevision int64
	LegacyUnowned  bool
}

type Grant struct {
	ID             string
	OrganizationID string
	PackageName    string
	SubjectType    string
	SubjectID      string
	Permission     string
	GrantedBy      string
	ExpiresAt      *time.Time
	CreatedAt      time.Time
}

type Namespace struct {
	ID             string
	Prefix         string
	OrganizationID string
	Status         string
	Verification   string
	VerifiedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ArtifactConfig struct {
	OrganizationID string
	UpstreamURL    string
	ScopePrefix    string
	ProxyMountPath string
	AuthMode       string
}

type Store interface {
	GetSkillAccess(ctx context.Context, packageName string) (SkillAccess, error)
	PutSkillAccess(ctx context.Context, access SkillAccess) error
	DeleteSkillAccess(ctx context.Context, packageName string) error
	ListGrants(ctx context.Context, organizationID string, packageName string) ([]Grant, error)
	PutGrant(ctx context.Context, grant Grant) error
	DeleteGrant(ctx context.Context, organizationID string, packageName string, grantID string) error
	IsMaintainer(ctx context.Context, packageName string, userID string) (bool, error)
	ListMaintainers(ctx context.Context, packageName string) ([]string, error)
	AddMaintainer(ctx context.Context, packageName string, organizationID string, userID string, grantedBy string) error
	RemoveMaintainer(ctx context.Context, packageName string, userID string) error
	GetNamespaceForPackage(ctx context.Context, packageName string) (Namespace, error)
	GetNamespaceByPrefix(ctx context.Context, prefix string) (Namespace, error)
	ListNamespaces(ctx context.Context, organizationID string) ([]Namespace, error)
	PutNamespace(ctx context.Context, namespace Namespace) (Namespace, error)
	VerifyNamespace(ctx context.Context, organizationID string, id string) (Namespace, error)
	GetArtifactConfig(ctx context.Context, organizationID string) (ArtifactConfig, error)
	PutArtifactConfig(ctx context.Context, config ArtifactConfig) (ArtifactConfig, error)
}
