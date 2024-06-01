#!/bin/bash
set -eo pipefail

bitcoin-cli -rpcuser=thorchain \
  -rpcpassword=password \
  ping
