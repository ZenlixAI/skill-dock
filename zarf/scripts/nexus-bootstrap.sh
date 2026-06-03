#!/bin/zsh
set -euo pipefail

export PATH="/usr/bin:/bin:/usr/sbin:/sbin:${PATH:-}"

NEXUS_URL=${NEXUS_URL:-http://127.0.0.1:8081}
STATE_DIR=${NEXUS_STATE_DIR:-/private/tmp/skill-registry-nexus}
REPOSITORY_NAME=${NEXUS_REPOSITORY_NAME:-skills-npm}
REPOSITORY_URL=${NEXUS_REPOSITORY_URL:-$NEXUS_URL/repository/$REPOSITORY_NAME}
ADMIN_USER=${NEXUS_ADMIN_USER:-admin}
ADMIN_PASSWORD=${NEXUS_ADMIN_PASSWORD:-admin123456}
INITIAL_PASSWORD_FILE=${NEXUS_INITIAL_PASSWORD_FILE:-$STATE_DIR/admin.password}
NPMRC_PATH=${NEXUS_NPMRC_PATH:-$STATE_DIR/.npmrc}
ANONYMOUS_REALM=${NEXUS_ANONYMOUS_REALM:-NexusAuthorizingRealm}
ANONYMOUS_USER=${NEXUS_ANONYMOUS_USER:-anonymous}
ENABLE_ANONYMOUS_READ=${NEXUS_ENABLE_ANONYMOUS_READ:-true}
LOGIN_EMAIL=${NEXUS_LOGIN_EMAIL:-nexus@example.local}
COMMUNITY_EULA_DISCLAIMER=${NEXUS_COMMUNITY_EULA_DISCLAIMER:-Use of Sonatype Nexus Repository - Community Edition is governed by the End User License Agreement at https://links.sonatype.com/products/nxrm/ce-eula. By returning the value from ‘accepted:false’ to ‘accepted:true’, you acknowledge that you have read and agree to the End User License Agreement at https://links.sonatype.com/products/nxrm/ce-eula.}
CURL_BIN=${CURL_BIN:-$(command -v curl)}
JQ_BIN=${JQ_BIN:-$(command -v jq)}
TR_BIN=${TR_BIN:-$(command -v tr)}
EXPECT_BIN=${EXPECT_BIN:-$(command -v expect)}

mkdir -p "$STATE_DIR"

for required_bin in "$CURL_BIN" "$JQ_BIN" "$TR_BIN" "$EXPECT_BIN"; do
  if [[ -z "$required_bin" || ! -x "$required_bin" ]]; then
    printf 'Required tool is missing or not executable: %s\n' "$required_bin" >&2
    exit 1
  fi
done

api_get() {
  local auth=$1
  local path=$2
  "$CURL_BIN" -fsS -u "$auth" "$NEXUS_URL$path"
}

api_put_json() {
  local auth=$1
  local path=$2
  local body=$3
  "$CURL_BIN" -fsS -u "$auth" -X PUT -H 'Content-Type: application/json' --data "$body" "$NEXUS_URL$path"
}

api_put_text() {
  local auth=$1
  local path=$2
  local body=$3
  "$CURL_BIN" -fsS -u "$auth" -X PUT -H 'Content-Type: text/plain' --data "$body" "$NEXUS_URL$path"
}

api_post_json() {
  local auth=$1
  local path=$2
  local body=$3
  "$CURL_BIN" -fsS -u "$auth" -X POST -H 'Content-Type: application/json' --data "$body" "$NEXUS_URL$path"
}

printf 'Waiting for Nexus API at %s ...\n' "$NEXUS_URL"
until "$CURL_BIN" -fsS "$NEXUS_URL/service/rest/v1/status" >/dev/null 2>&1; do
  sleep 5
done

current_auth="$ADMIN_USER:$ADMIN_PASSWORD"

if ! api_get "$current_auth" "/service/rest/v1/repositories" >/dev/null 2>&1; then
  if [[ ! -f "$INITIAL_PASSWORD_FILE" ]]; then
    printf 'Missing initial admin password file: %s\n' "$INITIAL_PASSWORD_FILE" >&2
    exit 1
  fi

  initial_password=$("$TR_BIN" -d '\r\n' <"$INITIAL_PASSWORD_FILE")
  initial_auth="$ADMIN_USER:$initial_password"
  printf 'Changing Nexus admin password to the local test default.\n'
  api_put_text "$initial_auth" "/service/rest/v1/security/users/$ADMIN_USER/change-password" "$ADMIN_PASSWORD" >/dev/null
else
  printf 'Admin password already initialized.\n'
fi

current_auth="$ADMIN_USER:$ADMIN_PASSWORD"

active_realms=$(api_get "$current_auth" "/service/rest/v1/security/realms/active")
if ! printf '%s' "$active_realms" | "$JQ_BIN" -e '.[] | select(. == "NpmToken")' >/dev/null; then
  updated_realms=$(printf '%s' "$active_realms" | "$JQ_BIN" -c '["NpmToken"] + .')
  printf 'Enabling NpmToken realm for npm publish.\n'
  api_put_json "$current_auth" "/service/rest/v1/security/realms/active" "$updated_realms" >/dev/null
fi

printf 'Accepting the Community Edition EULA for local automation.\n'
api_post_json "$current_auth" "/service/rest/v1/system/eula" "$("$JQ_BIN" -nc \
  --arg disclaimer "$COMMUNITY_EULA_DISCLAIMER" \
  '{accepted:true,disclaimer:$disclaimer}')" >/dev/null

anonymous_settings=$(api_get "$current_auth" "/service/rest/v1/security/anonymous")
anonymous_realm=$(printf '%s' "$anonymous_settings" | "$JQ_BIN" -r '.realmName // empty')
if [[ -z "$anonymous_realm" || "$anonymous_realm" == "null" ]]; then
  anonymous_realm="$ANONYMOUS_REALM"
fi

if [[ "$ENABLE_ANONYMOUS_READ" == "true" ]]; then
  printf 'Ensuring anonymous read is enabled.\n'
else
  printf 'Disabling anonymous read for strict npm auth verification.\n'
fi
api_put_json "$current_auth" "/service/rest/v1/security/anonymous" "$("$JQ_BIN" -nc \
  --argjson enabled "$([[ "$ENABLE_ANONYMOUS_READ" == "true" ]] && echo true || echo false)" \
  --arg realm "$anonymous_realm" \
  --arg user "$ANONYMOUS_USER" \
  '{enabled:$enabled,userId:$user,realmName:$realm}')" >/dev/null

if ! api_get "$current_auth" "/service/rest/v1/repositories" | "$JQ_BIN" -e --arg name "$REPOSITORY_NAME" '.[] | select(.name == $name)' >/dev/null; then
  printf 'Creating npm hosted repository %s.\n' "$REPOSITORY_NAME"
  api_post_json "$current_auth" "/service/rest/v1/repositories/npm/hosted" "$("$JQ_BIN" -nc \
    --arg name "$REPOSITORY_NAME" \
    '{name:$name,online:true,storage:{blobStoreName:"default",strictContentTypeValidation:true,writePolicy:"ALLOW_ONCE"}}')" >/dev/null
else
  printf 'Repository %s already exists.\n' "$REPOSITORY_NAME"
fi

repo_url="$REPOSITORY_URL"
if [[ "$repo_url" != */ ]]; then
  repo_url="${repo_url}/"
fi

rm -f "$NPMRC_PATH"
NEXUS_EXPECT_NPMRC_PATH="$NPMRC_PATH" \
NEXUS_EXPECT_REGISTRY_URL="$repo_url" \
NEXUS_EXPECT_ADMIN_USER="$ADMIN_USER" \
NEXUS_EXPECT_ADMIN_PASSWORD="$ADMIN_PASSWORD" \
NEXUS_EXPECT_LOGIN_EMAIL="$LOGIN_EMAIL" \
  "$EXPECT_BIN" <<'EOF'
set timeout 60
set npmrc $env(NEXUS_EXPECT_NPMRC_PATH)
set registry_url $env(NEXUS_EXPECT_REGISTRY_URL)
set admin_user $env(NEXUS_EXPECT_ADMIN_USER)
set admin_password $env(NEXUS_EXPECT_ADMIN_PASSWORD)
set login_email $env(NEXUS_EXPECT_LOGIN_EMAIL)

spawn npm adduser --registry $registry_url --auth-type=legacy --userconfig $npmrc
expect "Username:"
send "$admin_user\r"
expect "Password:"
send "$admin_password\r"
expect "Email (this will be public):"
send "$login_email\r"
expect "Logged in on $registry_url."
expect eof
EOF

printf 'Wrote isolated npm config: %s\n' "$NPMRC_PATH"
printf 'Nexus npm repository is ready at: %s\n' "$repo_url"
