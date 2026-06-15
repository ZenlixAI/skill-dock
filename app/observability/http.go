package observability

import platformobservability "skill_registry/internal/platform/observability"

type AuthDecision = platformobservability.AuthDecision
type StatusRecorder = platformobservability.StatusRecorder

const (
	AuthDecisionAllowed = platformobservability.AuthDecisionAllowed
	AuthDecisionDenied  = platformobservability.AuthDecisionDenied
)

var (
	NewStatusRecorder       = platformobservability.NewStatusRecorder
	SetObservedPrincipal    = platformobservability.SetObservedPrincipal
	SetObservedAction       = platformobservability.SetObservedAction
	SetObservedAuthDecision = platformobservability.SetObservedAuthDecision
	SetObservedOutcome      = platformobservability.SetObservedOutcome
)
