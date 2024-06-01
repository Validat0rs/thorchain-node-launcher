#!/usr/bin/env bash

source ./scripts/core.sh

# pull the current pinned alpine/k8s version from thornode chart
get_alpine_img() {
  # shellcheck disable=SC2046,SC2312
  yq -r '.global.images.alpineK8s | "alpine/k8s:" + .tag + "@sha256:" + .hash' \
    <"$(dirname $(readlink -f $0))"/../thornode/values.yaml
}

# prompt for namespace if unset
if [[ -z ${NAME} ]]; then
  read -r -p "=> Enter namespace: " NAME
fi

K="kubectl -n ${NAME}"

# select deployment in namespace
DEPLOYMENTS=$(${K} get deployments --no-headers -o custom-columns=":metadata.name")
echo "=> Select deployment to debug"
menu_default ${DEPLOYMENTS}
DEPLOYMENT_NAME="${MENU_SELECTED}"
DEPLOYMENT=$(${K} get deploy/"${DEPLOYMENT_NAME}" -o json)

# select container in deployment if multiple exist
CONTAINERS=$(jq '.spec.template.spec.containers' <<<"${DEPLOYMENT}")
if jq --exit-status 'length == 1' <<<"${CONTAINERS}" >/dev/null; then
  CONTAINER_NAME=$(jq -r '.[0].name' <<<"${CONTAINERS}")
  CONTAINER=$(jq -r '.[0]' <<<"${CONTAINERS}")
else
  echo "=> Select container to debug"
  CONTAINER_NAMES=$(jq -r '.[].name' <<<"${CONTAINERS}")
  menu_default ${CONTAINER_NAMES}
  CONTAINER_NAME="${MENU_SELECTED}"
  CONTAINER=$(jq -r '.[] | select(.name == "'${CONTAINER_NAME}'")' <<<"${CONTAINERS}")
fi

# select image for debug container
echo "=> Select image for debug container"
DEPLOY_IMAGE=$(jq -r '.image' <<<"${CONTAINER}")
menu_default "${DEPLOY_IMAGE}" "alpine"
IMAGE="${MENU_SELECTED}"
if [[ ${IMAGE} == "alpine" ]]; then
  IMAGE=$(get_alpine_img)
fi

# confirm deployment and container to debug
echo "=> Debugging ${red}${NAME}${reset}"
echo "Deployment: ${DEPLOYMENT_NAME}"
echo "Container: ${CONTAINER_NAME}"
echo "Image: ${IMAGE}"
confirm

# update container json
CONTAINER=$(jq '.name = "debug-'${DEPLOYMENT_NAME}'"' <<<"${CONTAINER}")
CONTAINER=$(jq '.stdin = true' <<<"${CONTAINER}")
CONTAINER=$(jq '.tty = true' <<<"${CONTAINER}")
CONTAINER=$(jq '.image = "'${IMAGE}'"' <<<"${CONTAINER}")
CONTAINER=$(jq '.command = ["sh"]' <<<"${CONTAINER}")
CONTAINER=$(jq '.args = []' <<<"${CONTAINER}")
CONTAINER=$(jq 'del(.startupProbe)' <<<"${CONTAINER}")
CONTAINER=$(jq 'del(.readinessProbe)' <<<"${CONTAINER}")
CONTAINER=$(jq 'del(.livenessProbe)' <<<"${CONTAINER}")

# create the run container spec
VOLUMES=$(jq '.spec.template.spec.volumes' <<<"${DEPLOYMENT}")
SPEC=$(
  cat <<EOF
{
  "apiVersion": "v1",
  "spec": {
    "containers": [${CONTAINER}],
    "volumes": ${VOLUMES}
  }
}
EOF
)

${K} scale --replicas=0 deploy/"${DEPLOYMENT_NAME}" --timeout=5m
${K} wait --for=delete pods -l app.kubernetes.io/name="${DEPLOYMENT_NAME}" --timeout=5m >/dev/null 2>&1 || true
${K} run -it --rm debug-"${DEPLOYMENT_NAME}" --restart=Never --image="${IMAGE}" --overrides="${SPEC}"
${K} scale --replicas=1 deploy/"${DEPLOYMENT_NAME}" --timeout=5m
