#!/usr/bin/env bash

source ./scripts/core.sh

get_node_info_short
get_discord_channel
get_discord_message

# Function to ask for confirmation
ask_confirmation() {
  while true; do
    # Prompt the user
    read -rp "Is this message correct? \"${DISCORD_MESSAGE}\" (Y/N): " yn

    case ${yn} in
      [Yy]*) break ;; # If yes, exit the loop
      [Nn]*)
        echo "exiting..."
        exit
        ;;                                  # If no, exit the script or modify the behavior as needed
      *) echo "Please answer yes or no." ;; # If the input is invalid, ask again
    esac
  done
}

# trim leanding and trailing whitesapce
DISCORD_MESSAGE=$(echo "${DISCORD_MESSAGE}" | sed 's/^[[:blank:]]*//;s/[[:blank:]]*$//')

echo "Confirm Message: '${DISCORD_MESSAGE}'"
ask_confirmation

if [[ -z ${DISCORD_MESSAGE} ]]; then
  echo "no message, exiting"
  exit 1
fi

kubectl exec -it -n "$NAME" -c thornode deploy/thornode -- /kube-scripts/retry.sh /kube-scripts/relay.sh "$DISCORD_CHANNEL" "$DISCORD_MESSAGE"
