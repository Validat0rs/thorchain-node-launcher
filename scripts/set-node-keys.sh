#!/usr/bin/env bash

set -e

source ./scripts/core.sh

get_node_info_short

echo "=> Setting THORNode keys"
kubectl exec -it -n "$NAME" -c thornode deploy/thornode -- /kube-scripts/set-node-keys.sh
sleep 5
echo THORNode Keys updated
