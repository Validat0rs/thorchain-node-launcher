FROM debian:11-slim as installer

WORKDIR /work

RUN apt-get update \
  && apt-get -y install --no-install-recommends wget ca-certificates gnupg

ENV BITCOIN_SHA256=2ab81515a763162435f7ea28bb1f10f69b6143f469278fc52c0b8cbaec6cf238
COPY version /version

# trunk-ignore(hadolint/DL4006): sh does not support -o pipefail
RUN set -ex && \
  BITCOIN_URL=https://github.com/bitcoin-cash-node/bitcoin-cash-node/releases/download/v$(cat /version)/bitcoin-cash-node-$(cat /version)-x86_64-linux-gnu.tar.gz && \
  wget -qO bitcoin.tar.gz "$BITCOIN_URL" && \
  sha256sum bitcoin.tar.gz && \
  echo "$BITCOIN_SHA256  bitcoin.tar.gz" | sha256sum -c - && \
  mkdir -p /work/bitcoin && \
  tar -xzvf bitcoin.tar.gz -C /work/bitcoin --strip-components=1 --exclude=*-qt


FROM debian:11-slim

COPY --from=installer /work/bitcoin/bin/* /usr/local/bin/
COPY --from=installer /work/bitcoin/lib/* /usr/local/lib/
COPY --from=installer /work/bitcoin/share/* /usr/local/share/

COPY ./scripts /scripts

RUN useradd --create-home bitcoin
USER bitcoin

EXPOSE 8332 8333 18332 18333 18443 18444
VOLUME ["/home/bitcoin/.bitcoin"]

ENTRYPOINT ["/scripts/entrypoint.sh"]
CMD ["bitcoind"]
