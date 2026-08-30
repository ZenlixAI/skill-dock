#!/bin/zsh
set -euo pipefail

cli_dir=${CLI_PUBLISH_CLI_DIR:-apps/skills-cli}
dist_tag=${CLI_PUBLISH_DIST_TAG:-latest}
npm_cache=${CLI_PUBLISH_NPM_CACHE:-/private/tmp/skill-registry-artifactkeeper/npm-cache}
npmrc_path=${CLI_PUBLISH_NPMRC:-/private/tmp/skill-registry-artifactkeeper/publish-cli.npmrc}
version_state_path=${CLI_PUBLISH_VERSION_STATE:-/private/tmp/skill-registry-artifactkeeper/cli-snapshot.version}
registry_url=${CLI_PUBLISH_REGISTRY:-http://127.0.0.1:18080/repository/skills-npm/}
repo_root="${0:A:h:h:h}"
cli_dir="${repo_root}/${cli_dir}"
package_json="${cli_dir}/package.json"
package_json_backup="$(mktemp "${TMPDIR:-/private/tmp}/skills-cli-package-json.XXXXXX")"

cleanup() {
  cp "${package_json_backup}" "${package_json}"
  rm -f "${package_json_backup}"
}
trap cleanup EXIT INT TERM

cp "${package_json}" "${package_json_backup}"

mkdir -p "${npm_cache}" "${version_state_path:h}"

read_base_version() {
  if [[ -f "${version_state_path}" ]]; then
    <"${version_state_path}" tr -d '[:space:]'
    return
  fi

  node -e 'const fs = require("fs"); const pkg = JSON.parse(fs.readFileSync(process.argv[1], "utf8")); process.stdout.write(pkg.version);' "${package_json}"
}

next_version="$(node -e '
const base = process.argv[1].trim();
const match = base.match(/^(.*-snapshot\.)(\d+)$/);
if (!match) {
  console.error(`unexpected snapshot version: ${base}`);
  process.exit(1);
}
process.stdout.write(match[1] + String(Number(match[2]) + 1));
' "$(read_base_version)")"

ARTIFACTKEEPER_PUBLIC_NPM_URL="${registry_url}" \
ARTIFACTKEEPER_NPM_PUBLISH_AUTHORIZATION="dummy-token" \
ARTIFACTKEEPER_NPMRC="${npmrc_path}" \
	"${repo_root}/zarf/scripts/write-local-npmrc.sh"

cd "${cli_dir}"
NPM_CONFIG_CACHE="${npm_cache}" NPM_CONFIG_USERCONFIG="${npmrc_path}" npm version "${next_version}" --no-git-tag-version
HUSKY=0 NPM_CONFIG_CACHE="${npm_cache}" NPM_CONFIG_USERCONFIG="${npmrc_path}" npm publish --tag "${dist_tag}" --registry "${registry_url}"
print -r -- "${next_version}" > "${version_state_path}"
