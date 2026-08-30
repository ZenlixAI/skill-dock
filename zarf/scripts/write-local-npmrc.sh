#!/bin/zsh
set -euo pipefail

registry_url=${ARTIFACTKEEPER_PUBLIC_NPM_URL:-http://127.0.0.1:30080/npm/skills-npm/}
registry_url=${registry_url%/}/
read_auth=${ARTIFACTKEEPER_NPM_READ_AUTHORIZATION:-}
publish_auth=${ARTIFACTKEEPER_NPM_PUBLISH_AUTHORIZATION:-}
npmrc_path=${ARTIFACTKEEPER_NPMRC:-/private/tmp/skill-registry-artifactkeeper/.npmrc}

mkdir -p "${npmrc_path:h}"
registry_host="${registry_url#http://}"
registry_host="${registry_host#https://}"
registry_host="${registry_host%/}"

{
  print -r -- "registry=${registry_url}"
  if [[ -n "${publish_auth}" ]]; then
    auth_token=${publish_auth#Basic }
    auth_token=${auth_token#Bearer }
    print -r -- "//${registry_host}/:_authToken=${auth_token}"
  elif [[ -n "${read_auth}" ]]; then
    auth_token=${read_auth#Basic }
    auth_token=${auth_token#Bearer }
    print -r -- "//${registry_host}/:_authToken=${auth_token}"
  fi
} > "${npmrc_path}"

chmod 600 "${npmrc_path}"
print -r -- "Wrote npm config: ${npmrc_path}"
