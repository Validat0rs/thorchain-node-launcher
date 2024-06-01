#!/bin/sh

. ./scripts/gitlab.sh

# get auth for container registry
REGISTRY=thorchain/devops/node-launcher
TOKEN=$(gitlab_registry_token $REGISTRY)

# check <chart>
check() {
  echo "Checking $1..."

  VALUES_FILE="$1/values.yaml"
  TAG=$(yq -r '.image.tag' "$VALUES_FILE")
  HASH=$(yq -r '.image.hash' "$VALUES_FILE")
  DIGEST=$(gitlab_registry_digest $REGISTRY "$TAG" "$TOKEN")

  if [ "sha256:$HASH" != "$DIGEST" ]; then
    echo "Hash mismatch!"
    echo "Expected: ${DIGEST}"
    echo "  Actual: sha256:${HASH}"
    exit 1
  fi
}

# check all images hosted in node-launcher registry
for CHART in *-daemon; do
  [ "$CHART" = "ethereum-daemon" ] && continue
  check "${CHART%/}"
done
