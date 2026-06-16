package http

import (
	"context"
	stdhttp "net/http"
)

type Pinger interface {
	Ping(context.Context) error
}

type NamedDependency struct {
	Name   string
	Pinger Pinger
}

type Module struct {
	repository  NamedDependency
	audit       NamedDependency
	idempotency NamedDependency
}

func NewModule(repository NamedDependency, audit NamedDependency, idempotency NamedDependency) *Module {
	return &Module{repository: repository, audit: audit, idempotency: idempotency}
}

func (m *Module) Readyz(ctx context.Context) (Readyz, int) {
	checks := map[string]string{
		"repository":  "ok",
		"audit":       "ok",
		"idempotency": "ok",
	}
	status := stdhttp.StatusOK

	for _, dependency := range []NamedDependency{m.repository, m.audit, m.idempotency} {
		if dependency.Name == "" || dependency.Pinger == nil {
			continue
		}
		if err := dependency.Pinger.Ping(ctx); err != nil {
			checks[dependency.Name] = err.Error()
			status = stdhttp.StatusServiceUnavailable
		}
	}

	response := Readyz{
		Status: "ready",
		Checks: checks,
	}
	if status != stdhttp.StatusOK {
		response.Status = "degraded"
	}
	return response, status
}

func DependencyPinger(value any) Pinger {
	pinger, _ := value.(Pinger)
	return pinger
}
