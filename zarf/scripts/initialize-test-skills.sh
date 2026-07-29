#!/bin/sh
set -eu

publish_skill() {
  fixture_path="$1"
  fixture_metadata="$fixture_path/package.json"
  if node - "${SKILLS_REGISTRY_URL}" "$fixture_metadata" <<'NODE'
(async () => {
  const fs = require('node:fs');

  const registry = process.argv[2].replace(/\/+$/, '');
  const metadataPath = process.argv[3];
  const metadata = JSON.parse(fs.readFileSync(metadataPath, 'utf8'));
  const packageName = encodeURIComponent(metadata.name);
  const version = encodeURIComponent(metadata.version);
  const token = process.env.SKILLS_REGISTRY_TOKEN;

  const response = await fetch(`${registry}/api/v1/skills/${packageName}/versions/${version}`, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (response.ok) {
    process.exit(0);
  }
  if (response.status === 404) {
    process.exit(10);
  }

  console.error(`failed to check existing fixture ${metadata.name}@${metadata.version}: HTTP ${response.status}`);
  process.exit(11);
})().catch((error) => {
  console.error(`failed to check existing fixture: ${error instanceof Error ? error.message : String(error)}`);
  process.exit(11);
});
NODE
  then
    echo "Skipping existing fixture from ${fixture_path}"
    return 0
  else
    check_status=$?
    if [ "$check_status" -ne 10 ]; then
      return "$check_status"
    fi
  fi

  echo "Importing fixture from ${fixture_path}"
  skills import-local "${fixture_path}" \
    --registry "${SKILLS_REGISTRY_URL}" \
    --publish-registry "${SKILLS_NPM_REGISTRY}"
  echo "Imported fixture from ${fixture_path}"
}

publish_skill /runtime-test-skills/runtime-basic
publish_skill /runtime-test-skills/runtime-command-suite
publish_skill /runtime-test-skills/runtime-agent-mcp
publish_skill /runtime-test-skills/runtime-versioned-v0.1.0
publish_skill /runtime-test-skills/runtime-versioned-v0.2.0
publish_skill /runtime-test-skills/runtime-collision-alpha
publish_skill /runtime-test-skills/runtime-collision-beta
