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

# get the node vault address
NODES=$(kubectl exec -it -n "$NAME" deploy/thornode -c thornode -- curl -s http://localhost:1317/thorchain/nodes)
PUB_KEY=$(echo "$NODES" | jq -r ".[] | select(.node_address == \"$NODE_ADDRESS\") | .pub_key_set.secp256k1")
echo "=> Pub key: $PUB_KEY"

# get the yggdrasil address
VAULTS=$(kubectl exec -it -n "$NAME" deploy/thornode -c thornode -- curl -s http://localhost:1317/thorchain/vaults/yggdrasil)
YGG_ADDR=$(echo "$VAULTS" | jq -r ".[] | select(.pub_key == \"$PUB_KEY\") | .addresses[] | select(.chain == \"$CHAIN\") | .address")
YGG_BALANCE=$(echo "$VAULTS" | jq -r ".[] | select(.pub_key == \"$PUB_KEY\") | .coins[] | select(.asset == \"$CHAIN.$CHAIN\") | .amount")

# get the chain listunspent output
LIST_UNSPENT=$(
  cat <<EOF | jq -c | sed 's/"/\\"/g'
{
  "method": "listunspent",
  "params": [0, 9999999, ["$YGG_ADDR"]]
}
EOF
)
UNSPENT=$(kubectl exec -it -n "$NAME" deploy/bifrost -c bifrost -- sh -c "curl --user thorchain:password -s --data-binary \"$LIST_UNSPENT\" http://\$${CHAIN}_HOST")
CHAIN_BALANCE=$(echo "$UNSPENT" | jq '([.result[]|.amount]|add//0)*1e8')

echo "=> Yggdrasil address: $YGG_ADDR"
echo "=> Thorchain balance: $YGG_BALANCE"
echo "=> Chain balance: $CHAIN_BALANCE"

# confirm
echo "=> Starting import and rescan - this may appear to hang and take over an hour..."
confirm

# import and rescan
IMPORT_ADDRESS=$(
  cat <<EOF | jq -c | sed 's/"/\\"/g'
{
  "method": "importaddress",
  "params": ["$YGG_ADDR", "", true]
}
EOF
)
kubectl exec -it -n "$NAME" deploy/bifrost -c bifrost -- sh -c "curl --user thorchain:password -s --data-binary \"$IMPORT_ADDRESS\" http://\$${CHAIN}_HOST"
