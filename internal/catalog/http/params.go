package http

import (
	"net/url"
	"strconv"
	"strings"

	catalog "skill_registry/internal/catalog/domain"
	"skill_registry/internal/platform/observability"
)

func DecodeVersion(raw string) (string, *catalog.Error) {
	version, err := url.PathUnescape(raw)
	if err != nil {
		return "", catalog.NewError(400, "invalid_request", "version path is not valid URL encoding.", map[string]any{"version": raw})
	}
	if !IsExactVersionString(version) {
		return "", catalog.NewError(422, "invalid_request", "version must be an exact semver string.", map[string]any{"version": version})
	}
	return version, nil
}

func ParsePageSize(values url.Values) (int, *catalog.Error) {
	return ParsePageSizeValue(values.Get("pageSize"))
}

func ParsePageSizeValue(raw string) (int, *catalog.Error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 20, nil
	}
	pageSize, err := strconv.Atoi(raw)
	if err != nil {
		return 0, catalog.NewError(400, "invalid_request", "pageSize must be an integer.", map[string]any{"pageSize": raw})
	}
	if pageSize <= 0 {
		return 0, catalog.NewError(400, "invalid_request", "pageSize must be greater than or equal to 1.", map[string]any{"pageSize": raw})
	}
	return pageSize, nil
}

func ParseCursor(values url.Values) (int, *catalog.Error) {
	return ParseCursorValue(values.Get("cursor"))
}

func ParseCursorValue(raw string) (int, *catalog.Error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	cursor, err := parseCursorToken(raw)
	if err != nil {
		return 0, catalog.NewError(400, "invalid_request", "cursor must be a valid cursor token.", map[string]any{"cursor": raw})
	}
	return cursor, nil
}

func parseCursorToken(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if strings.HasPrefix(value, "cursor_") {
		value = strings.TrimPrefix(value, "cursor_")
	}
	return strconv.Atoi(value)
}

func ParseBooleanQueryParam(raw string, field string, defaultValue bool) (bool, *catalog.Error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return defaultValue, nil
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes":
		return true, nil
	case "0", "false", "no":
		return false, nil
	default:
		return false, catalog.NewError(400, "invalid_request", field+" must be a boolean.", map[string]any{field: value})
	}
}

func ParseGovernanceTargetTypeFilter(raw string) (string, *catalog.Error) {
	value := strings.TrimSpace(raw)
	switch value {
	case "", string(catalog.GovernanceTargetSkill), string(catalog.GovernanceTargetVersion):
		return value, nil
	default:
		return "", catalog.NewError(400, "invalid_request", "target_type is invalid.", map[string]any{"target_type": value})
	}
}

func ParseAuthDecisionFilter(raw string) (string, *catalog.Error) {
	value := strings.TrimSpace(raw)
	switch value {
	case "", string(observability.AuthDecisionAllowed), string(observability.AuthDecisionDenied):
		return value, nil
	default:
		return "", catalog.NewError(400, "invalid_request", "auth_decision must be one of: allowed, denied.", map[string]any{"auth_decision": value})
	}
}

func ParseExactVersionQueryParam(raw string, field string) (string, *catalog.Error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if !IsExactVersionString(value) {
		return "", catalog.NewError(400, "invalid_request", field+" must be an exact semver string.", map[string]any{field: value})
	}
	return value, nil
}

func IsExactVersionString(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}
	return true
}
