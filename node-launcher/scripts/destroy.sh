#!/usr/bin/env bash

source ./scripts/core.sh

get_node_info_short

node_exists || die "No existing THORNode found, make sure this is the correct name"

display_status

echo -e "=> Destroying a $boldgreen$TYPE$reset THORNode on $boldgreen$NET$reset named $boldgreen$NAME$reset"
echo
echo
warn "!!! Make sure your got your BOND back before destroying your THORNode !!!"
confirm
echo "=> Deleting THORNode"
helm delete "$NAME" -n "$NAME"
kubectl delete namespace "$NAME"
