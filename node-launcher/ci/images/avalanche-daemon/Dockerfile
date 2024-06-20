# Upstream: https://github.com/ava-labs/avalanchego/blob/v1.7.14/Dockerfile

# Changes to the minimum golang version must also be replicated in
# scripts/ansible/roles/golang_base/defaults/main.yml
# scripts/build_avalanche.sh
# scripts/local.Dockerfile
# Dockerfile (here)
# README.md
# go.mod
# ============= Compilation Stage ================
FROM golang:1.21.11-bullseye AS builder

COPY version /version
ARG COMMIT=41e46d1178116eb7421fe2adbf2eaa7ef97b0f8b
WORKDIR /build
RUN TAG="v$(cat /version)" && \
  git init && \
  git remote add origin https://github.com/ava-labs/avalanchego && \
  git fetch --depth 1 origin "$TAG" && \
  git checkout $COMMIT && \
  ./scripts/build.sh

# ============= Cleanup Stage ================
FROM debian:11-slim AS execution

# Install curl and jq for startup and liveness probes
RUN apt-get update && apt-get install -y curl jq

# Maintain compatibility with previous images
RUN mkdir -p /avalanchego/build
WORKDIR /avalanchego/build

# Copy the executables into the container
COPY --from=builder /build/build/ .

CMD [ "./avalanchego" ]
