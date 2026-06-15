#!/usr/bin/env bash
# Point client/server at the published common module (remote go get).
set -euo pipefail

TEMPLATE_ORG="${1:?Usage: $0 <org> <repo> [version]}"
TEMPLATE_REPO="${2:?}"
COMMON_VERSION="${3:-v0.0.1}"
COMMON_MODULE="github.com/${TEMPLATE_ORG}/${TEMPLATE_REPO}/common"
COMMON_TAG="common/${COMMON_VERSION}"

remove_common_replace() {
  local mod_dir="$1"
  perl -pi -e '
    if (/^replace github.com\/.*\/common => \.\.\/common\s*$/) {
      $_ = "";
    }
  ' "${mod_dir}/go.mod"
}

echo "Linking client/server to ${COMMON_MODULE}@${COMMON_TAG}"

for module in client server; do
  remove_common_replace "${module}"
  (
    cd "${module}"
    GOPROXY=direct go get "${COMMON_MODULE}@${COMMON_TAG}"
    go mod tidy
  )
done

echo "Client/server linked to remote common module."
