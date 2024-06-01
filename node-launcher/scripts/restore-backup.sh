#!/usr/bin/env bash

source ./scripts/core.sh

get_node_info_short

node_exists || die "No existing THORNode found, make sure this is the correct name"

if [ "$SERVICE" = "" ]; then
  echo "=> Select a THORNode service to restore a backup from"
  menu thornode thornode bifrost
  SERVICE=$MENU_SELECTED
fi

if ! kubectl -n "$NAME" get pvc "$SERVICE" >/dev/null 2>&1; then
  warn "Volume $SERVICE not found"
  echo
  exit 0
fi

TIMES=$(find "$PWD/backups/$NAME/$SERVICE/" -name "*.tar.gz" | sort -r | awk -F "/" '{ match($NF, /[0-9]+/); print strftime("%F@%R:%S", substr($NF, RSTART, RLENGTH)) " "$0 }')

if [ "$TIMES" == "" ]; then
  warn "No backups found for service $SERVICE"
  echo
  exit 0
fi

echo "=> Select the backup time"
readarray -t TIMES <<<"$TIMES"
menu "${TIMES[0]}" "${TIMES[@]}"

FILE=$(echo "$MENU_SELECTED" | awk '{ print $2 }')

if [ "$SERVICE" = "bifrost" ]; then
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
            "sleep 300"
          ],
          "name": "$SERVICE",
          "image": "busybox:1.33",
          "volumeMounts": [
            {"mountPath": "/root/.thornode", "name": "data", "subPath": "thornode"},
            {"mountPath": "/var/data/bifrost", "name": "data", "subPath": "data"}
          ]
        }
      ],
      "volumes": [{"name": "data", "persistentVolumeClaim": {"claimName": "$SERVICE"}}]
    }
  }"
EOF
  )
else
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
            "sleep 300"
          ],
          "name": "$SERVICE",
          "image": "busybox:1.33",
          "volumeMounts": [{"mountPath": "/root", "name":"data"}]
        }
      ],
      "volumes": [{"name": "data", "persistentVolumeClaim": {"claimName": "$SERVICE"}}]
    }
  }
EOF
  )
fi

echo
echo "=> Restoring backup service $boldgreen$SERVICE$reset from THORNode in $boldgreen$NAME$reset using backup $boldgreen$FILE$reset"
confirm

POD="deploy/$SERVICE"
if (kubectl get pod -n "$NAME" -l "app.kubernetes.io/name=$SERVICE" 2>&1 | grep "No resources found") >/dev/null 2>&1; then
  kubectl run -n "$NAME" "backup-$SERVICE" --restart=Never --image="busybox:1.33" --overrides="$SPEC"
  kubectl wait --for=condition=ready pods "backup-$SERVICE" -n "$NAME" --timeout=5m >/dev/null 2>&1
  POD="pod/backup-$SERVICE"
fi

FILE_BASE=$(basename "$FILE")
FILE_DIR=$(dirname "$FILE")
tar -C "$FILE_DIR" -cf - "$FILE_BASE" | kubectl exec -i -n "$NAME" "$POD" -c "$SERVICE" -- tar xf - -C /root/.thornode

kubectl exec -it -n "$NAME" "$POD" -c "$SERVICE" -- sh -c "cd /root/.thornode && tar xf \"$FILE_BASE\""

if (kubectl get pod -n "$NAME" -l "app.kubernetes.io/name=$SERVICE" 2>&1 | grep "No resources found") >/dev/null 2>&1; then
  kubectl delete pod --now=true -n "$NAME" "backup-$SERVICE"
fi

echo "Restore backup successful for $SERVICE"
