FROM ubuntu:22.04

COPY version /version
ARG HOST_USER_UID=1000
ARG HOST_USER_GID=1000

ENV BNCHOME=/opt/bnbchaind

WORKDIR /release
RUN VERSION="$(cat /version)" && \
  apt-get update && apt-get install -y --no-install-recommends curl=7.81.0-1ubuntu1.16 jq=1.6-2.1ubuntu3 ca-certificates=20230311ubuntu0.22.04.1 unzip=6.0-26ubuntu3.1 && \
  rm -rf /var/lib/apt/lists/* && \
  curl -LO "https://github.com/bnb-chain/node/releases/download/v${VERSION}/linux_binary.zip" && \
  curl -LO "https://github.com/bnb-chain/node/releases/download/v${VERSION}/mainnet_config.zip" && \
  unzip linux_binary.zip && \
  unzip mainnet_config.zip && mv asset/mainnet . && \
  chmod 755 linux/* && \
  rmdir asset && rm ./*.zip

RUN set -ex && \
  mkdir -p "$BNCHOME" && \
  groupadd --gid "$HOST_USER_GID" bnbchaind && \
  useradd --uid "$HOST_USER_UID" --gid "$HOST_USER_GID" --shell /bin/bash --no-create-home bnbchaind && \
  chown -R bnbchaind:bnbchaind "$BNCHOME" /release

VOLUME ${BNCHOME}

# RPC service listen on port 27147 and P2P service listens on port 27146 by default.
# Prometheus is enabled on port 26660 by default, and the endpoint is /metrics.
EXPOSE 27146 27147 26656 26657 26660

ENTRYPOINT ["sh", "-c", "cp -r /release/mainnet $BNCHOME/config; /release/linux/bnbchaind start --home ${BNCHOME}"]
