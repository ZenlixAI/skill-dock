package http

import (
	stdhttp "net/http"

	"skill_registry/business/catalog"
)

type RequestLimiter struct {
	tokens chan struct{}
}

func NewRequestLimiter(limit int) *RequestLimiter {
	if limit <= 0 {
		return &RequestLimiter{}
	}
	return &RequestLimiter{tokens: make(chan struct{}, limit)}
}

func (l *RequestLimiter) Acquire() (func(), *catalog.Error) {
	if l == nil || l.tokens == nil {
		return func() {}, nil
	}

	select {
	case l.tokens <- struct{}{}:
		return func() {
			<-l.tokens
		}, nil
	default:
		return nil, catalog.NewError(stdhttp.StatusTooManyRequests, "request_limit_exceeded", "Request concurrency limit exceeded.", nil)
	}
}
