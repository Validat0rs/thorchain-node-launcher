FROM debian:11-slim

COPY version /version
WORKDIR /home/bsc

RUN TAG="v$(sed 's/-.*//' /version)" && \
  apt-get update && \
  apt-get install -y --no-install-recommends ca-certificates=20210119 openssl=1.1.1w-0+deb11u1 wget=1.21-1+deb11u1 unzip=6.0-26+deb11u1 && \
  rm -rf /var/lib/apt/lists/* && \
  useradd --create-home bsc && \
  wget --progress=dot:giga "https://github.com/bnb-chain/bsc/releases/download/$TAG/geth_linux" && \
  wget --progress=dot:giga "https://github.com/bnb-chain/bsc/releases/download/$TAG/mainnet.zip" && \
  unzip mainnet.zip -d config && \
  rm mainnet.zip && \
  sed -i '/\[Node\.LogConfig\]/,$ d' config/config.toml && \
  sed -i '/HTTPHost/c\HTTPHost = "0.0.0.0"' config/config.toml && \
  sed -i '/HTTPVirtualHosts/c\HTTPVirtualHosts = ["*"]' config/config.toml && \
  sed -i '/HTTPModules/c\HTTPModules = ["eth","net","engine","web3","miner","personal","txpool","debug"]' config/config.toml && \
  chmod +x geth_linux && \
  mv geth_linux /usr/local/bin/geth && \
  chown -R bsc:bsc /home/bsc

USER bsc

ENTRYPOINT ["geth", "--config", "/home/bsc/config/config.toml", "--datadir", "/home/bsc/data"]
