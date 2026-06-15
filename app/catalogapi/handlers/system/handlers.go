package systemhandlers

import internalsystemhttp "skill_registry/internal/system/http"

type ReadyzFunc = internalsystemhttp.ReadyzFunc
type MetricsFunc = internalsystemhttp.MetricsFunc
type Handlers = internalsystemhttp.Handlers

func New(readyz ReadyzFunc, metrics MetricsFunc) Handlers {
	return internalsystemhttp.New(readyz, metrics)
}
