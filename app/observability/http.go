package observability

import (
	"net/http"
	"strings"

	"skill_registry/app/auth"
)

type AuthDecision string

const (
	AuthDecisionAllowed AuthDecision = "allowed"
	AuthDecisionDenied  AuthDecision = "denied"
)

type StatusRecorder struct {
	http.ResponseWriter
	Status               int
	ObservedPrincipal    auth.Principal
	ObservedAction       string
	ObservedAuthDecision AuthDecision
	observedOutcomeValue string
}

func NewStatusRecorder(writer http.ResponseWriter) *StatusRecorder {
	return &StatusRecorder{
		ResponseWriter: writer,
		Status:         http.StatusOK,
	}
}

func (r *StatusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *StatusRecorder) Outcome() string {
	if strings.TrimSpace(r.observedOutcomeValue) != "" {
		return r.observedOutcomeValue
	}
	if r.Status >= 200 && r.Status < 400 {
		return "succeeded"
	}
	if r.Status == http.StatusUnauthorized {
		return "unauthorized"
	}
	if r.Status == http.StatusForbidden {
		return "forbidden"
	}
	return "failed"
}

func SetObservedPrincipal(writer http.ResponseWriter, current auth.Principal) {
	if recorder, ok := writer.(*StatusRecorder); ok {
		recorder.ObservedPrincipal = current
	}
}

func SetObservedAction(writer http.ResponseWriter, action string) {
	if recorder, ok := writer.(*StatusRecorder); ok {
		recorder.ObservedAction = action
	}
}

func SetObservedAuthDecision(writer http.ResponseWriter, decision AuthDecision) {
	if recorder, ok := writer.(*StatusRecorder); ok {
		recorder.ObservedAuthDecision = decision
	}
}

func SetObservedOutcome(writer http.ResponseWriter, outcome string) {
	if recorder, ok := writer.(*StatusRecorder); ok {
		recorder.observedOutcomeValue = outcome
	}
}
