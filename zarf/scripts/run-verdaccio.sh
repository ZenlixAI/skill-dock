#!/bin/zsh
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
ZARF_DIR=$(cd "$SCRIPT_DIR/.." && pwd)
STATE_DIR=${VERDACCIO_STATE_DIR:-/private/tmp/skill-registry-verdaccio}
LISTEN_ADDR=${VERDACCIO_ADDR:-127.0.0.1:4873}
CONFIG_PATH=${VERDACCIO_CONFIG:-$ZARF_DIR/docker/verdaccio/public.yaml}

mkdir -p "$STATE_DIR/storage"
touch "$STATE_DIR/htpasswd"

exec npx --yes verdaccio@6 --config "$CONFIG_PATH" --listen "$LISTEN_ADDR"
