#!/bin/sh
set -eu

publish_skill() {
  skills import-local "$1" \
    --registry "${SKILLS_REGISTRY_URL}" \
    --publish-registry "${SKILLS_NPM_REGISTRY}"
}

publish_skill /runtime-test-skills/runtime-basic
publish_skill /runtime-test-skills/runtime-command-suite
publish_skill /runtime-test-skills/runtime-agent-mcp
publish_skill /runtime-test-skills/runtime-versioned-v0.1.0
publish_skill /runtime-test-skills/runtime-versioned-v0.2.0
publish_skill /runtime-test-skills/runtime-collision-alpha
publish_skill /runtime-test-skills/runtime-collision-beta
