#!/usr/bin/env bash

source ./scripts/core.sh

if ! snapshot_available; then
  warn "Snapshot not available in this cluster"
  echo
  exit 0
fi

get_node_info_short
if [ "$SERVICE" == "" ]; then
  echo "=> Select a THORNode service to snapshot"
  menu thornode thornode bifrost midgard binance-smart-daemon bitcoin-daemon bitcoin-cash-daemon dogecoin-daemon ethereum-daemon litecoin-daemon gaia-daemon avalanche-daemon
  SERVICE=$MENU_SELECTED
fi

make_snapshot "$SERVICE"
