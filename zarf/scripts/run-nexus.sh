#!/bin/zsh
set -euo pipefail

IMAGE=${NEXUS_IMAGE:-sonatype/nexus3:latest}
LISTEN_ADDR=${NEXUS_ADDR:-127.0.0.1:8081}
STATE_DIR=${NEXUS_STATE_DIR:-/private/tmp/skill-registry-nexus}
JAVA_OPTS=${NEXUS_JAVA_OPTS:-"-Xms1024m -Xmx1024m -XX:MaxDirectMemorySize=1024m -Djava.util.prefs.userRoot=/nexus-data/javaprefs"}

mkdir -p "$STATE_DIR"

exec docker run \
  --rm \
  --name skill-registry-nexus \
  -p "${LISTEN_ADDR}:8081" \
  -e INSTALL4J_ADD_VM_PARAMS="$JAVA_OPTS" \
  -v "${STATE_DIR}:/nexus-data" \
  "$IMAGE"
