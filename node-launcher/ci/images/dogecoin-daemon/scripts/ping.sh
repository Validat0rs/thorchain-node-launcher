#!/bin/bash
set -eo pipefail

dogecoin-cli -rpcuser=thorchain \
  -rpcpassword=password \
  ping
