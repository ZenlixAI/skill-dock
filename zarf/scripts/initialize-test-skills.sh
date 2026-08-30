#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
skills_cli="${SKILLS_CLI_BIN:-$repo_root/apps/skills-cli/bin/cli.mjs}"
registry_url=${SKILLS_REGISTRY_URL:-http://127.0.0.1:18080}
publisher_token=${SKILLS_REGISTRY_TOKEN:-publisher-secret}
npm_registry=${SKILLS_NPM_REGISTRY:-${ARTIFACTKEEPER_INTERNAL_NPM_URL:-http://host.docker.internal:30080/}}

wait_for_registry() {
  ready_url="$registry_url/readyz"
  for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30; do
    if curl -sS -o /dev/null "$ready_url"; then
      return 0
    fi
    sleep 1
  done
  echo "catalog did not become ready at ${ready_url}" >&2
  return 1
}

publish_skill() {
  fixture_path="$1"
  fixture_metadata="$fixture_path/package.json"
  set -- $(node -e 'const fs = require("node:fs"); const path = process.argv[1]; const pkg = JSON.parse(fs.readFileSync(path, "utf8")); process.stdout.write(pkg.name + "\n" + pkg.version + "\n");' "$fixture_metadata")
  package_name=${1:-}
  package_version=${2:-}
  if [ -z "$package_name" ] || [ -z "$package_version" ]; then
    echo "failed to read fixture metadata from ${fixture_path}" >&2
    return 11
  fi

  encoded_package_name=$(node -e 'process.stdout.write(encodeURIComponent(process.argv[1]))' "$package_name")
  install_target_url="$registry_url/api/v1/skills/${encoded_package_name}/install-target?version=${package_version}"
  install_payload=$(curl -sS -H "Authorization: Bearer ${publisher_token}" "$install_target_url")
  if [ -z "$install_payload" ]; then
    echo "failed to check existing fixture ${package_name}@${package_version}: empty install target response" >&2
    return 11
  fi
  tarball_url=$(printf '%s' "$install_payload" | node -e 'const fs = require("node:fs"); const raw = fs.readFileSync(0, "utf8"); const data = JSON.parse(raw); process.stdout.write(data.tarballUrl || "")')
  if [ -n "$tarball_url" ]; then
    tarball_status=$(curl -sS -o /dev/null -w '%{http_code}' -I "$tarball_url")
    if [ "$tarball_status" = "200" ]; then
      echo "Skipping existing fixture from ${fixture_path}"
      return 0
    fi
    if [ "$tarball_status" != "404" ]; then
      echo "failed to check existing fixture ${package_name}@${package_version}: tarball HTTP ${tarball_status}" >&2
      return 11
    fi
  fi

  echo "Importing fixture from ${fixture_path}"
  SKILLS_REGISTRY_URL="$registry_url" \
  SKILLS_REGISTRY_TOKEN="$publisher_token" \
  node "$skills_cli" import-local "${fixture_path}" \
    --registry "$registry_url" \
    --publish-registry "$npm_registry"
  echo "Imported fixture from ${fixture_path}"
}

wait_for_registry
publish_skill "$repo_root/my-test-skill"
