package catalogapi

import (
	"context"
	"net/http"
)

type systemWorkflow interface {
	readyz(ctx context.Context) (readyzResponse, int)
}

type readyzResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

type namedReadinessDependency struct {
	name   string
	pinger readinessPinger
}

type systemModule struct {
	repository  namedReadinessDependency
	audit       namedReadinessDependency
	idempotency namedReadinessDependency
}

func (m *systemModule) readyz(ctx context.Context) (readyzResponse, int) {
	checks := map[string]string{
		"repository":  "ok",
		"audit":       "ok",
		"idempotency": "ok",
	}
	status := http.StatusOK

	for _, dependency := range []namedReadinessDependency{m.repository, m.audit, m.idempotency} {
		if dependency.name == "" || dependency.pinger == nil {
			continue
		}
		if err := dependency.pinger.Ping(ctx); err != nil {
			checks[dependency.name] = err.Error()
			status = http.StatusServiceUnavailable
		}
	}

	response := readyzResponse{
		Status: "ready",
		Checks: checks,
	}
	if status != http.StatusOK {
		response.Status = "degraded"
	}
	return response, status
}

func dependencyPinger(value any) readinessPinger {
	pinger, _ := value.(readinessPinger)
	return pinger
}
