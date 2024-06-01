#!/usr/bin/env bash

source ./scripts/core.sh

set -eo pipefail

export TYPE=validator
get_node_info_short
if ! node_exists; then
  die "No existing THORNode found, make sure this is the correct name"
fi
display_status

# prompt for the daemon
echo -e "\n=> Select UTXO daemon"
menu bitcoin bitcoin bitcoin-cash dogecoin litecoin
DAEMON="$MENU_SELECTED"

# set chain
case "$DAEMON" in
  bitcoin) CHAIN=BTC ;;
  bitcoin-cash) CHAIN=BCH ;;
  dogecoin) CHAIN=DOGE ;;
  litecoin) CHAIN=LTC ;;
  *) die "Unknown daemon $DAEMON" ;;
esac

# select rescan duration
echo -e "\n=> Select rescan duration"
menu "7 days" "7 days" "30 days" "90 days" "forever"
DURATION="$MENU_SELECTED"

# get unix start timestamp
case "$DURATION" in
  "7 days") START=$(date -d "7 days ago" +%s) ;;
  "30 days") START=$(date -d "30 days ago" +%s) ;;
  "90 days") START=$(date -d "90 days ago" +%s) ;;
  forever) START=0 ;;
  *) die "Unknown duration $DURATION" ;;
esac

# get all vault addresses for the chain
VAULTS=$(kubectl exec -it -n "$NAME" deploy/thornode -c thornode -- curl -s http://localhost:1317/thorchain/vaults/asgard)
ADDRS=$(echo "$VAULTS" | jq -r ".[] | .addresses[] | select(.chain == \"$CHAIN\") | .address")

IMPORT_PARAMS='[]'

for addr in $ADDRS; do
  echo
  echo "=> Checking $addr..."
  ASGARD_BALANCE=$(echo "$VAULTS" | jq -r ".[] | select(any(.addresses[]; .chain == \"$CHAIN\" and .address == \"$addr\")) | .coins[] | select(.asset==\"$CHAIN.$CHAIN\") | .amount")

  # get the chain listunspent output
  LIST_UNSPENT=$(
    cat <<EOF | jq -c | sed 's/"/\\"/g'
{
  "method": "listunspent",
  "params": [0, 9999999, ["$addr"]]
}
EOF
  )
  UNSPENT=$(kubectl exec -it -n "$NAME" deploy/bifrost -c bifrost -- sh -c "curl --user thorchain:password -s --data-binary \"$LIST_UNSPENT\" http://\$${CHAIN}_HOST")
  DAEMON_BALANCE=$(echo "$UNSPENT" | jq '([.result[]|.amount]|add//0)*1e8')

  echo "=> Asgard balance: $ASGARD_BALANCE"
  echo "=> Daemon balance: $DAEMON_BALANCE"
  echo

  echo -n "$boldyellow:: Rescan $addr? Confirm [y/n]: $reset" && read -r ans && [ "${ans:-N}" != y ] && continue

  # append to import params
  IMPORT_PARAMS=$(echo "$IMPORT_PARAMS" | jq -c ". + [{\"scriptPubKey\": {\"address\": \"$addr\"}, \"timestamp\": $START}]")
done

# skip if no addresses
if [ "$IMPORT_PARAMS" = '[]' ]; then
  echo "=> No addresses to rescan"
  exit 0
fi

# import and rescan
echo "=> Starting import and rescan - this may appear to hang and take over an hour..."
IMPORT_ADDRESS=$(
  cat <<EOF | jq -c | sed 's/"/\\"/g'
{
  "method": "importmulti",
  "params": {
    "requests": $IMPORT_PARAMS
  }
}
EOF
)
kubectl exec -it -n "$NAME" deploy/bifrost -c bifrost -- sh -c "curl --user thorchain:password -s --data-binary \"$IMPORT_ADDRESS\" http://\$${CHAIN}_HOST"
