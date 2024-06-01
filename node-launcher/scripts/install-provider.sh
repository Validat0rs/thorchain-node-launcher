#!/usr/bin/env bash

set -e

source ./scripts/core.sh

: "${NAMESPACE:=provider}"

if helm -n "${NAMESPACE}" status provider >/dev/null 2>&1; then
  helm diff -C 3 upgrade --install provider ./provider -n "${NAMESPACE}" -f ./provider/values.yaml
  confirm
fi

echo "=> Installing Provider"
helm upgrade --install provider ./provider -n "${NAMESPACE}" --create-namespace --wait -f ./provider/values.yaml
echo
