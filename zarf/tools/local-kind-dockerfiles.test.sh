#!/bin/sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"
CATALOG_LOCAL_DOCKERFILE="${ROOT_DIR}/zarf/docker/local/Dockerfile"
WEB_LOCAL_DOCKERFILE="${ROOT_DIR}/zarf/docker/local/skills-web.Dockerfile"

test -f "${CATALOG_LOCAL_DOCKERFILE}"
test -f "${WEB_LOCAL_DOCKERFILE}"

if rg -n "truth-ai-registry.cn-hangzhou.cr.aliyuncs.com/base/" "${CATALOG_LOCAL_DOCKERFILE}" "${WEB_LOCAL_DOCKERFILE}" >/dev/null; then
  echo "expected local kind Dockerfiles to omit the base registry prefix" >&2
  exit 1
fi

BUILD_CATALOG_OUTPUT="$(make -n build-catalog-image 2>/dev/null)"
if [ "${BUILD_CATALOG_OUTPUT}" != "docker build -f zarf/docker/local/Dockerfile -t skill-registry-catalog:dev ." ]; then
  echo "unexpected build-catalog-image expansion" >&2
  echo "actual: ${BUILD_CATALOG_OUTPUT}" >&2
  exit 1
fi

BUILD_WEB_OUTPUT="$(make -n build-web-image 2>/dev/null)"
if [ "${BUILD_WEB_OUTPUT}" != "docker build -f zarf/docker/local/skills-web.Dockerfile -t skill-registry-web:dev ." ]; then
  echo "unexpected build-web-image expansion" >&2
  echo "actual: ${BUILD_WEB_OUTPUT}" >&2
  exit 1
fi

DEV_UPDATE_OUTPUT="$(make -n dev-update 2>/dev/null)"
printf '%s\n' "${DEV_UPDATE_OUTPUT}" | rg "zarf/docker/local/(Dockerfile|skills-web.Dockerfile)" >/dev/null

if printf '%s\n' "${DEV_UPDATE_OUTPUT}" | rg "dev-reset-no-base-prefix|dev-reset-local" >/dev/null; then
  echo "dev-update should not depend on temporary prefix-stripping helpers" >&2
  exit 1
fi

printf 'local kind Dockerfile wiring test passed\n'
