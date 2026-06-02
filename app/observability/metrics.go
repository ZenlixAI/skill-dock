package observability

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Collector struct {
	mu                   sync.Mutex
	requestCounts        map[string]int64
	durationCounts       map[string]int64
	durationSumMS        map[string]int64
	actionRequestCounts  map[string]int64
	actionDurationCounts map[string]int64
	actionDurationSumMS  map[string]int64
}

func NewCollector() *Collector {
	return &Collector{
		requestCounts:        make(map[string]int64),
		durationCounts:       make(map[string]int64),
		durationSumMS:        make(map[string]int64),
		actionRequestCounts:  make(map[string]int64),
		actionDurationCounts: make(map[string]int64),
		actionDurationSumMS:  make(map[string]int64),
	}
}

func (m *Collector) RecordRequest(method string, path string, status int, duration time.Duration, action string, authDecision AuthDecision, outcome string) {
	if m == nil {
		return
	}
	countKey := fmt.Sprintf("%s\x00%s\x00%d", method, path, status)
	durationKey := fmt.Sprintf("%s\x00%s", method, path)
	actionCountKey := fmt.Sprintf("%s\x00%s\x00%s\x00%d", action, authDecision, outcome, status)
	actionDurationKey := fmt.Sprintf("%s\x00%s", action, authDecision)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.requestCounts[countKey]++
	m.durationCounts[durationKey]++
	m.durationSumMS[durationKey] += duration.Milliseconds()
	m.actionRequestCounts[actionCountKey]++
	m.actionDurationCounts[actionDurationKey]++
	m.actionDurationSumMS[actionDurationKey] += duration.Milliseconds()
}

func (m *Collector) RenderPrometheus() string {
	if m == nil {
		return ""
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var lines []string
	lines = append(lines, "# HELP skill_registry_http_requests_total Total HTTP requests by method, path, and status.")
	lines = append(lines, "# TYPE skill_registry_http_requests_total counter")

	requestKeys := make([]string, 0, len(m.requestCounts))
	for key := range m.requestCounts {
		requestKeys = append(requestKeys, key)
	}
	sort.Strings(requestKeys)
	for _, key := range requestKeys {
		parts := strings.Split(key, "\x00")
		lines = append(lines, fmt.Sprintf(
			`skill_registry_http_requests_total{method=%q,path=%q,status=%q} %d`,
			parts[0],
			parts[1],
			parts[2],
			m.requestCounts[key],
		))
	}

	lines = append(lines, "# HELP skill_registry_http_request_duration_ms_count Request duration count by method and path.")
	lines = append(lines, "# TYPE skill_registry_http_request_duration_ms_count counter")
	lines = append(lines, "# HELP skill_registry_http_request_duration_ms_sum Request duration sum in milliseconds by method and path.")
	lines = append(lines, "# TYPE skill_registry_http_request_duration_ms_sum counter")

	durationKeys := make([]string, 0, len(m.durationCounts))
	for key := range m.durationCounts {
		durationKeys = append(durationKeys, key)
	}
	sort.Strings(durationKeys)
	for _, key := range durationKeys {
		parts := strings.Split(key, "\x00")
		lines = append(lines, fmt.Sprintf(
			`skill_registry_http_request_duration_ms_count{method=%q,path=%q} %d`,
			parts[0],
			parts[1],
			m.durationCounts[key],
		))
		lines = append(lines, fmt.Sprintf(
			`skill_registry_http_request_duration_ms_sum{method=%q,path=%q} %d`,
			parts[0],
			parts[1],
			m.durationSumMS[key],
		))
	}

	lines = append(lines, "# HELP skill_registry_http_action_requests_total Total HTTP requests by action, auth decision, outcome, and status.")
	lines = append(lines, "# TYPE skill_registry_http_action_requests_total counter")
	actionRequestKeys := make([]string, 0, len(m.actionRequestCounts))
	for key := range m.actionRequestCounts {
		actionRequestKeys = append(actionRequestKeys, key)
	}
	sort.Strings(actionRequestKeys)
	for _, key := range actionRequestKeys {
		parts := strings.Split(key, "\x00")
		lines = append(lines, fmt.Sprintf(
			`skill_registry_http_action_requests_total{action=%q,auth_decision=%q,outcome=%q,status=%q} %d`,
			parts[0],
			parts[1],
			parts[2],
			parts[3],
			m.actionRequestCounts[key],
		))
	}

	lines = append(lines, "# HELP skill_registry_http_action_duration_ms_count Request duration count by action and auth decision.")
	lines = append(lines, "# TYPE skill_registry_http_action_duration_ms_count counter")
	lines = append(lines, "# HELP skill_registry_http_action_duration_ms_sum Request duration sum in milliseconds by action and auth decision.")
	lines = append(lines, "# TYPE skill_registry_http_action_duration_ms_sum counter")
	actionDurationKeys := make([]string, 0, len(m.actionDurationCounts))
	for key := range m.actionDurationCounts {
		actionDurationKeys = append(actionDurationKeys, key)
	}
	sort.Strings(actionDurationKeys)
	for _, key := range actionDurationKeys {
		parts := strings.Split(key, "\x00")
		lines = append(lines, fmt.Sprintf(
			`skill_registry_http_action_duration_ms_count{action=%q,auth_decision=%q} %d`,
			parts[0],
			parts[1],
			m.actionDurationCounts[key],
		))
		lines = append(lines, fmt.Sprintf(
			`skill_registry_http_action_duration_ms_sum{action=%q,auth_decision=%q} %d`,
			parts[0],
			parts[1],
			m.actionDurationSumMS[key],
		))
	}

	return strings.Join(lines, "\n") + "\n"
}
