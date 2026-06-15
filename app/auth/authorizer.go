package auth

import platformauth "skill_registry/internal/platform/auth"

type Config = platformauth.Config
type TokenConfig = platformauth.TokenConfig
type Principal = platformauth.Principal
type Action = platformauth.Action
type AuthorizationError = platformauth.AuthorizationError
type Authorizer = platformauth.Authorizer

const (
	ActionReadCatalog            = platformauth.ActionReadCatalog
	ActionReadAuditEvents        = platformauth.ActionReadAuditEvents
	ActionReadSession            = platformauth.ActionReadSession
	ActionReadHealthz            = platformauth.ActionReadHealthz
	ActionReadReadyz             = platformauth.ActionReadReadyz
	ActionReadMetrics            = platformauth.ActionReadMetrics
	ActionPackagingPrepare       = platformauth.ActionPackagingPrepare
	ActionPackagingValidate      = platformauth.ActionPackagingValidate
	ActionPackagingPack          = platformauth.ActionPackagingPack
	ActionPackagingPublish       = platformauth.ActionPackagingPublish
	ActionCreateReleaseRecord    = platformauth.ActionCreateReleaseRecord
	ActionApplySkillGovernance   = platformauth.ActionApplySkillGovernance
	ActionApplyVersionGovernance = platformauth.ActionApplyVersionGovernance
)

var (
	NewAuthorizer                          = platformauth.NewAuthorizer
	FilterSkillSummariesByPrincipal        = platformauth.FilterSkillSummariesByPrincipal
	FilterCuratedSkillSummariesByPrincipal = platformauth.FilterCuratedSkillSummariesByPrincipal
	FilterGovernanceRecordsByPrincipal     = platformauth.FilterGovernanceRecordsByPrincipal
)
