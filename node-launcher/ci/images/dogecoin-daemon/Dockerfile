FROM debian:11-slim as installer

WORKDIR /work

RUN apt-get update \
    && apt-get -y install --no-install-recommends wget ca-certificates gnupg

COPY version /version

RUN VERSION="$(cat /version)" && \
  wget  --progress=dot:giga "https://github.com/dogecoin/dogecoin/releases/download/v${VERSION}/dogecoin-${VERSION}-x86_64-linux-gnu.tar.gz" && \
  mkdir dogecoin && \
  tar xzvf "dogecoin-${VERSION}-x86_64-linux-gnu.tar.gz" --strip-components=1 -C dogecoin

FROM debian:11-slim

COPY --from=installer /work/dogecoin/bin/* /usr/local/bin/
COPY --from=installer /work/dogecoin/lib/* /usr/local/lib/
COPY --from=installer /work/dogecoin/share/* /usr/local/share/

COPY ./scripts /scripts

RUN useradd --create-home dogecoin
USER dogecoin

EXPOSE 22555 22556 44555 44556 18332 18444
VOLUME ["/home/dogecoin/.dogecoin"]

ENTRYPOINT ["/scripts/entrypoint.sh"]
CMD ["dogecoind"]
