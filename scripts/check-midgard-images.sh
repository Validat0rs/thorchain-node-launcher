#!/bin/sh

. ./scripts/gitlab.sh

# get auth for container registry
REGISTRY=thorchain/midgard
TOKEN=$(gitlab_registry_token $REGISTRY)

echo "Checking midgard..."

TAG=$(yq -r '.appVersion' "midgard/Chart.yaml")
HASH=$(yq -r '.image.hash' "midgard/values.yaml")
DIGEST=$(gitlab_registry_digest $REGISTRY "$TAG" "$TOKEN")

if [ "sha256:$HASH" != "$DIGEST" ]; then
  echo "Hash mismatch!"
  echo "Expected: ${DIGEST}"
  echo "  Actual: sha256:${HASH}"
  exit 1
fi
