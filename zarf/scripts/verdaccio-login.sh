#!/bin/zsh
set -euo pipefail

REGISTRY_URL=${1:-${VERDACCIO_URL:-http://127.0.0.1:4873}}
STATE_DIR=${VERDACCIO_STATE_DIR:-/private/tmp/skill-registry-verdaccio-auth}
NPMRC_PATH=${VERDACCIO_NPMRC_PATH:-$STATE_DIR/.npmrc}

mkdir -p "$STATE_DIR"

cat >"$NPMRC_PATH" <<EOF
registry=$REGISTRY_URL
always-auth=true
EOF

echo "Using isolated npm config: $NPMRC_PATH"
echo "Running interactive npm adduser against $REGISTRY_URL"

exec env NPM_CONFIG_USERCONFIG="$NPMRC_PATH" npm adduser --registry "$REGISTRY_URL" --auth-type=legacy
