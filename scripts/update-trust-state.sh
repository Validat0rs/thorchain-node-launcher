#!/usr/bin/env bash

set -euo pipefail

MIDGARD_HASHES="https://snapshots.ninerealms.com/snapshots/midgard-blockstore/hashes"

# update mainnet midgard hashes with latest
sed -i '/thorchain-blockstore-hashes/q' midgard/templates/configmap.yaml
curl -s "$MIDGARD_HASHES" | sed -e 's/^/    /' >>midgard/templates/configmap.yaml
