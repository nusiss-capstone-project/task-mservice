#!/usr/bin/env bash
# Tag and push the common module so client/server can go get it remotely.
set -euo pipefail

TEMPLATE_ORG="${1:?Usage: $0 <org> <repo> [version]}"
TEMPLATE_REPO="${2:?}"
COMMON_VERSION="${3:-v0.0.1}"
COMMON_TAG="common/${COMMON_VERSION}"

git config user.name "${GIT_USER_NAME:-github-actions[bot]}"
git config user.email "${GIT_USER_EMAIL:-41898282+github-actions[bot]@users.noreply.github.com}"

echo "Publishing common module tag: ${COMMON_TAG}"

(cd common && go mod tidy)

git add -A
if git diff --cached --quiet; then
  echo "No changes to commit before tagging common."
else
  git commit -m "chore: prepare common module for ${COMMON_TAG}"
fi

git push origin HEAD

if git rev-parse "${COMMON_TAG}" >/dev/null 2>&1; then
  echo "Tag ${COMMON_TAG} already exists locally."
else
  git tag -a "${COMMON_TAG}" -m "common ${COMMON_VERSION}"
fi

git push origin "${COMMON_TAG}"

echo "Published ${COMMON_TAG}"
