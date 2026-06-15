package catalogapi

import (
	"skill_registry/business/catalog"
	platformhttp "skill_registry/internal/platform/http"
)

type requestLimiter struct {
	*platformhttp.RequestLimiter
}

func newRequestLimiter(limit int) *requestLimiter {
	return &requestLimiter{RequestLimiter: platformhttp.NewRequestLimiter(limit)}
}

func (l *requestLimiter) acquire() (func(), *catalog.Error) {
	if l == nil {
		return func() {}, nil
	}
	return l.RequestLimiter.Acquire()
}
