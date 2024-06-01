#!/usr/bin/env bash

set -e

source ./scripts/core.sh

: "${NAMESPACE:="provider"}"

helm get all provider -n "${NAMESPACE}"
echo -n "The above resources will be deleted "
confirm

echo "=> Deleting Provider"
helm delete provider -n "${NAMESPACE}"
echo
