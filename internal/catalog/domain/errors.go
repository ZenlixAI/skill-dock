package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNoResolvableVersion         = errors.New("no resolvable version")
	ErrAgentSkillAggregateNotFound = errors.New("agent skill aggregate not found")
	ErrAggregateRevisionConflict   = errors.New("agent skill aggregate revision conflict")
	ErrAggregateSnapshotNotFound   = errors.New("aggregate snapshot not found")
)

type Error struct {
	Status  int
	Code    string
	Message string
	Details map[string]any
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewError(status int, code string, message string, details map[string]any) *Error {
	return &Error{
		Status:  status,
		Code:    code,
		Message: message,
		Details: details,
	}
}
