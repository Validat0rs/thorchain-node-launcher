#!/usr/bin/env bash

source ./scripts/core.sh

get_node_info_short
echo "=> Select a THORNode service to reset"
SERVICE=thornode

if node_exists; then
  echo
  warn "Found an existing THORNode, make sure this is the node you want to update:"
  display_status
  echo
fi

echo "=> Resetting service $boldyellow$SERVICE$reset of a THORNode named $boldyellow$NAME$reset"
echo
warn "Destructive command, be careful, your service data volume data will be wiped out and restarted to sync from scratch"
confirm

IMAGE=$(kubectl -n "$NAME" get deploy/thornode -o jsonpath='{$.spec.template.spec.containers[:1].image}')
SPEC=$(
  cat <<EOF
{
  "apiVersion": "v1",
  "spec": {
    "containers": [
      {
        "command": [
          "sh",
          "-c",
          "thornode unsafe-reset-all"
        ],
        "name": "debug-thornode",
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

kubectl scale -n "$NAME" --replicas=0 deploy/thornode --timeout=5m
kubectl wait --for=delete pods -l app.kubernetes.io/name=thornode -n "$NAME" --timeout=5m >/dev/null 2>&1 || true
kubectl run -n "$NAME" -it reset-thornode --rm --restart=Never --image="$IMAGE" --overrides="$SPEC"

kubectl scale -n "$NAME" --replicas=1 deploy/thornode --timeout=5m
