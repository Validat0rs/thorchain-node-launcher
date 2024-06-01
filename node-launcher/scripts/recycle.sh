#!/usr/bin/env bash

set -eo pipefail

source ./scripts/core.sh

get_node_info

if ! node_exists; then
  die "No existing THORNode found, make sure this is the correct name"
fi

if [ "$TYPE" != "validator" ]; then
  die "Only validators should be recycled"
fi

display_status

echo -e "=> Recycling a $boldgreen$TYPE$reset THORNode on $boldgreen$NET$reset named $boldgreen$NAME$reset"
echo
echo
warn "!!! Make sure your got your BOND back before recycling your THORNode !!!"
confirm

# delete gateway resources
echo -e "$beige=> Recycling THORNode - deleting gateway resources...$reset"
kubectl -n "$NAME" delete deployment gateway
kubectl -n "$NAME" delete service gateway
kubectl -n "$NAME" delete configmap gateway-external-ip

# delete thornode resources
echo -e "$beige=> Recycling THORNode - deleting thornode resources...$reset"
kubectl -n "$NAME" delete deployment thornode
kubectl -n "$NAME" delete configmap thornode-external-ip
kubectl -n "$NAME" delete secret thornode-password
kubectl -n "$NAME" delete secret thornode-mnemonic

# delete all key material from thornode while preserving chain data
echo -e "$beige=> Recycling THORNode - deleting thornode derived keys...$reset"
IMAGE=alpine:latest@sha256:4edbd2beb5f78b1014028f4fbb99f3237d9561100b6881aabbf5acce2c4f9454
SPEC=$(
  cat <<EOF
{
  "apiVersion": "v1",
  "spec": {
    "containers": [
      {
        "command": [
          "rm",
          "-rf",
          "/root/.thornode/THORChain-ED25519",
          "/root/.thornode/data/priv_validator_state.json",
          "/root/.thornode/keyring-file/",
          "/root/.thornode/config/node_key.json",
          "/root/.thornode/config/priv_validator_key.json",
          "/root/.thornode/config/genesis.json"
        ],
        "name": "reset-thornode-keys",
        "stdin": true,
        "tty": true,
        "image": "$IMAGE",
        "volumeMounts": [{"mountPath": "/root", "name":"data"}]
      }
    ],
    "volumes": [{"name": "data", "persistentVolumeClaim": {"claimName": "thornode"}}]
  }
}
EOF
)
kubectl -n "$NAME" run -it --rm reset-thornode-keys --restart=Never --image="$IMAGE" --overrides="$SPEC"

# delete bifrost resources
echo -e "$beige=> Recycling THORNode - deleting bifrost resources...$reset"
kubectl -n "$NAME" delete deployment bifrost
kubectl -n "$NAME" delete pvc bifrost
kubectl -n "$NAME" delete configmap bifrost-external-ip

# recreate resources
echo -e "$green=> Recycling THORNode - recreating deleted resources...$reset"
NET=$NET TYPE=$TYPE NAME=$NAME ./scripts/install.sh

echo -e "$green=> Recycle complete.$reset"
