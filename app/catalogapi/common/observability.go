package common

import platformhttp "skill_registry/internal/platform/http"

type ObservedState = platformhttp.ObservedState

var (
	EnsureObservedState      = platformhttp.EnsureObservedState
	ObservedStateFromContext = platformhttp.ObservedStateFromContext
	SetObservedPrincipal     = platformhttp.SetObservedPrincipal
	SetObservedAction        = platformhttp.SetObservedAction
	SetObservedAuthDecision  = platformhttp.SetObservedAuthDecision
	SetObservedOutcome       = platformhttp.SetObservedOutcome
	ObservedOutcome          = platformhttp.ObservedOutcome
)
