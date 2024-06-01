#!/usr/bin/env bash

source ./scripts/core.sh

get_node_net

if [ -n "$HARDFORK_BLOCK_HEIGHT" ]; then
  EXTRA_ARGS="$EXTRA_ARGS --set thornode.haltHeight=$HARDFORK_BLOCK_HEIGHT"
fi

source ./scripts/pre-install.sh

if [ "$0" == "./scripts/update.sh" ] && snapshot_available; then
  make_snapshot "thornode"
  if [ "$TYPE" != "fullnode" ]; then
    make_snapshot "bifrost"
  fi
fi

# check to ensure required CRDs are created before deploying
if ! kubectl get crd servicemonitors.monitoring.coreos.com >/dev/null 2>&1; then
  echo "=> Required ServiceMonitor CRD not found - run 'make tools' before proceeding."
  exit 1
fi

case $TYPE in
  genesis)
    deploy_genesis
    ;;
  validator)
    deploy_validator
    ;;
  fullnode)
    deploy_fullnode
    ;;
  daemons)
    EXTRA_ARGS="$EXTRA_ARGS --set thornode.enabled=false"
    EXTRA_ARGS="$EXTRA_ARGS --set bifrost.enabled=false"
    EXTRA_ARGS="$EXTRA_ARGS --set gateway.enabled=false"
    deploy_validator
    ;;
esac
