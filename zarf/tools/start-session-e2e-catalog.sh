#!/bin/sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"
DB_PATH="${SKILL_REGISTRY_E2E_DB_PATH:-/private/tmp/skill-registry-session-e2e.db}"
rm -f "${DB_PATH}"

export GOCACHE="${GOCACHE:-/private/tmp/go-cache}"
export SKILL_REGISTRY_ADDR="${SKILL_REGISTRY_ADDR:-127.0.0.1:18081}"
export SKILL_REGISTRY_DATABASE_URL="${SKILL_REGISTRY_DATABASE_URL:-file:${DB_PATH}}"
export SKILL_REGISTRY_ENABLE_SAMPLE_DATA="${SKILL_REGISTRY_ENABLE_SAMPLE_DATA:-true}"
export SKILL_REGISTRY_ENABLE_MOCK_REGISTRY="${SKILL_REGISTRY_ENABLE_MOCK_REGISTRY:-false}"
export SKILL_REGISTRY_SESSION_SECRET="${SKILL_REGISTRY_SESSION_SECRET:-playwright-session-secret}"
if [ -z "${SKILL_REGISTRY_SESSION_USERS_JSON:-}" ]; then
  export SKILL_REGISTRY_SESSION_USERS_JSON='{"reader":{"password":"reader-secret","role":"reader","actor":"reader"},"operator":{"password":"op-secret","role":"operator","actor":"operator"}}'
fi

cd "${ROOT_DIR}"
exec go run ./cmd/skill-registry
