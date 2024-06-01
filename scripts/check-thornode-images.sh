#!/bin/sh

. ./scripts/gitlab.sh

# get auth for container registry
REGISTRY=thorchain/thornode
TOKEN=$(gitlab_registry_token $REGISTRY)

# check <values-file> <network>
check() {
  echo "Checking thornode $2..."

  VALUES_FILE="$1"
  EXPECTED_NETWORK="$2"

  TAG=$(yq -r '.global.tag' "$VALUES_FILE")
  HASH=$(yq -r '.global.hash' "$VALUES_FILE")
  IMAGE="registry.gitlab.com/thorchain/thornode:${TAG}@sha256:${HASH}"
  VERSION=$(yq -r '.global.tag|split("-")[-1]' "$VALUES_FILE")

  # verify the tag matches the digest
  DIGEST=$(gitlab_registry_digest $REGISTRY "$TAG" "$TOKEN")
  if [ "sha256:$HASH" != "$DIGEST" ]; then
    echo "Error: $IMAGE"
    echo "Digest mismatch: $HASH != $DIGEST"
    exit 1
  fi

  IMAGE_VERSION_LONG=$(docker run --rm "$IMAGE" thornode version --long)

  IMAGE_VERSION=$(echo "$IMAGE_VERSION_LONG" | yq -r '.version')
  if [ "$IMAGE_VERSION" != "$VERSION" ]; then
    echo "Error: $IMAGE"
    echo "Version mismatch: $IMAGE_VERSION != $VERSION"
    exit 1
  fi

  IMAGE_NETWORK=$(echo "$IMAGE_VERSION_LONG" | yq -r '.build_tags')
  if [ "$IMAGE_NETWORK" != "$EXPECTED_NETWORK" ]; then
    echo "Error: $IMAGE"
    echo "Network mismatch: $IMAGE_NETWORK != $EXPECTED_NETWORK"
    exit 1
  fi

  IMAGE_COMMIT=$(echo "$IMAGE_VERSION_LONG" | yq -r '.commit')
  if [ "$IMAGE_NETWORK" = "mainnet" ]; then
    # check that the tag explicitly matches the image commit
    REPO_COMMIT=$(curl -s "https://gitlab.com/api/v4/projects/13422983/repository/commits/v$VERSION" | jq -r .id)
    if [ "$IMAGE_COMMIT" != "$REPO_COMMIT" ]; then
      echo "Warning: $IMAGE"
      echo "Commit mismatch: image=$IMAGE_COMMIT tag=$REPO_COMMIT"
      echo
    fi
  else
    # check that the version on the repo commit matches
    REPO_VERSION=$(curl -s "https://gitlab.com/thorchain/thornode/-/raw/$IMAGE_COMMIT/version")
    if [ "$IMAGE_VERSION" != "$REPO_VERSION" ]; then
      echo "Error: $IMAGE"
      echo "Image Commit: $IMAGE_COMMIT"
      echo "Image Version: $IMAGE_VERSION"
      echo "Repo Version: $REPO_VERSION"
      exit 1
    fi
  fi

  echo "Image $IMAGE is valid"
  echo "$IMAGE_VERSION_LONG" | head -n5
  echo
}

check thornode-stack/mainnet.yaml mainnet
check thornode-stack/stagenet.yaml stagenet
