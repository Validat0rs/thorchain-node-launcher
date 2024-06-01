FROM debian:11-slim

# Build the Docker image used for running this repository's CI.

RUN apt-get update && apt-get upgrade -y && apt-get install -y \
    curl gpg ca-certificates git

COPY scripts /tmp/scripts
RUN cd /tmp && \
    ./scripts/install-helm.sh && \
    rm -rf /tmp/scripts

WORKDIR /repo
