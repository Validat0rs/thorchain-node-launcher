#!/usr/bin/env bash

set -e

source ./scripts/core.sh

get_node_info_short

echo "=> Resuming node global halt from a THORNode named $boldyellow$NAME$reset"
confirm

kubectl exec -it -n "$NAME" -c thornode deploy/thornode -- /kube-scripts/resume.sh
sleep 5
echo THORChain resumed

display_status
