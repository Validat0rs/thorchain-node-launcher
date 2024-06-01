#!/usr/bin/env bash

source ./scripts/core.sh

get_node_info_short
get_mimir_key
get_mimir_value

kubectl exec -it -n "$NAME" -c thornode deploy/thornode -- /kube-scripts/retry.sh /kube-scripts/mimir.sh "$MIMIR_KEY" "$MIMIR_VALUE"
