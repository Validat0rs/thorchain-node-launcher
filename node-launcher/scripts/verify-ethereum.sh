#!/usr/bin/env bash

source ./scripts/core.sh

get_node_info_short

# get finalized slot and state root
finalized=$(
  kubectl exec -n "$NAME" deploy/ethereum-daemon -- \
    wget -qO- http://localhost:3500/eth/v1/beacon/headers/finalized
)
slot=$(echo "$finalized" | jq -r '.data.header.message.slot')
state_root=$(echo "$finalized" | jq -r '.data.header.message.state_root')

cat <<EOF

Slot: $slot
State Root: $state_root
EOF

# prompt operator to check explorers
cat <<EOF

Navigate to multiple explorers (following are examples) and verify slot state root:
https://beaconcha.in/slot/$slot
https://beaconscan.com/slot/$slot
EOF
