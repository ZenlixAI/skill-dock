#!/bin/zsh
set -euo pipefail

container_name=${ARTIFACTKEEPER_CONTAINER_NAME:-skill-registry-artifact-keeper}
image=${ARTIFACTKEEPER_IMAGE:-verdaccio/verdaccio:6}
state_dir=${ARTIFACTKEEPER_STATE_DIR:-/private/tmp/skill-registry-artifactkeeper}
config_dir=${ARTIFACTKEEPER_CONFIG_DIR:-$state_dir/conf}
storage_dir=${ARTIFACTKEEPER_STORAGE_DIR:-$state_dir/storage}
# This script bootstraps the keeper itself. Client installs use the Catalog proxy.
registry_url=${ARTIFACTKEEPER_KEEPER_NPM_URL:-http://127.0.0.1:30080/}
registry_url=${registry_url%/}
npmrc_path=${ARTIFACTKEEPER_NPMRC:-$state_dir/.npmrc}
auth_file=${ARTIFACTKEEPER_TOKEN_FILE:-$state_dir/publish.auth}
username=${ARTIFACTKEEPER_PUBLISH_USERNAME:-publisher}
password=${ARTIFACTKEEPER_PUBLISH_PASSWORD:-publisher-secret}
email=${ARTIFACTKEEPER_PUBLISH_EMAIL:-publisher@example.local}
curl_bin=${CURL_BIN:-$(command -v curl)}

if [[ -z "$curl_bin" || ! -x "$curl_bin" ]]; then
  print -r -- "curl is required" >&2
  exit 1
fi

mkdir -p "$config_dir" "$storage_dir" "${npmrc_path:h}" "${auth_file:h}"

cat >"$config_dir/config.yaml" <<'EOF'
storage: /verdaccio/storage
listen: 0.0.0.0:4873
url_prefix: /
auth:
  htpasswd:
    file: /verdaccio/storage/htpasswd
    max_users: -1
uplinks:
  npmjs:
    url: https://registry.npmjs.org/
packages:
  '@*/*':
    access: $all
    # Local-only dev registry (127.0.0.1): avoid client npm credential setup.
    publish: $all
    proxy: npmjs
  '**':
    access: $all
    publish: $all
    proxy: npmjs
middlewares:
  audit:
    enabled: true
logs:
  - {type: stdout, format: pretty, level: http}
EOF

if docker ps --format '{{.Names}}' | grep -qx "$container_name"; then
	# Reload the generated config; the persistent storage remains untouched.
	docker restart "$container_name" >/dev/null
elif docker ps -a --format '{{.Names}}' | grep -qx "$container_name"; then
  docker start "$container_name" >/dev/null
else
  docker pull "$image" >/dev/null
  docker run -d \
    --name "$container_name" \
    --restart unless-stopped \
    -p 127.0.0.1:30080:4873 \
    -v "$storage_dir:/verdaccio/storage" \
    -v "$config_dir:/verdaccio/conf" \
    "$image" >/dev/null
fi

for attempt in {1..60}; do
  if "$curl_bin" -fsS --max-time 2 "$registry_url/-/ping" >/dev/null 2>&1; then
    break
  fi
  sleep 1
  if [[ "$attempt" -eq 60 ]]; then
    print -r -- "Artifact Keeper did not become ready at ${registry_url}/-/ping" >&2
    exit 1
  fi
done

extract_token() {
  local path=$1
  local line token
  if [[ ! -f "$path" ]]; then
    return 1
  fi
  while IFS= read -r line; do
    line=${line//[$'\r\n']/}
    case "$line" in
      *':_authToken='*)
        token=${line#*=}
        # ponytail: ignore the old bootstrap placeholder so stale local state self-heals.
        if [[ -n "$token" && "$token" != "dummy-token" ]]; then
          print -r -- "Bearer ${token}"
          return 0
        fi
        ;;
      *':_auth='*)
        token=${line#*=}
        # ponytail: ignore the old bootstrap placeholder so stale local state self-heals.
        if [[ -n "$token" && "$token" != "dummy-token" ]]; then
          print -r -- "Bearer ${token}"
          return 0
        fi
        ;;
    esac
  done < "$path"
  return 1
}

if auth_value=$(extract_token "$npmrc_path"); then
  print -r -- "$auth_value" > "$auth_file"
else
  # ponytail: local dev bootstrap falls back to the fixed dummy token so stack startup stays non-interactive.
  print -r -- "Bearer dummy-token" > "$auth_file"
fi
chmod 600 "$auth_file"
print -r -- "Bootstrapped Artifact Keeper at ${registry_url} and wrote auth token: ${auth_file}"
