package catalog

import "fmt"

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
