package catalogapi

import (
	"net/http"

	"skill_registry/business/catalog"
)

type requestLimiter struct {
	tokens chan struct{}
}

func newRequestLimiter(limit int) *requestLimiter {
	if limit <= 0 {
		return &requestLimiter{}
	}
	return &requestLimiter{tokens: make(chan struct{}, limit)}
}

func (l *requestLimiter) acquire() (func(), *catalog.Error) {
	if l == nil || l.tokens == nil {
		return func() {}, nil
	}

	select {
	case l.tokens <- struct{}{}:
		return func() {
			<-l.tokens
		}, nil
	default:
		return nil, catalog.NewError(http.StatusTooManyRequests, "request_limit_exceeded", "Request concurrency limit exceeded.", nil)
	}
}
