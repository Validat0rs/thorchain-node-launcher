FROM debian:11-slim AS installer

# install dependencies
RUN apt-get update && \
  apt-get -y install --no-install-recommends \
  ca-certificates=20210119 \
  git=1:2.30.2-1+deb11u2 \
  gnupg=2.2.27-2+deb11u2 \
  wget=1.21-1+deb11u1

# import builder keys (https://github.com/bitcoin/bitcoin/tree/master/contrib/verify-binaries)
WORKDIR /keys
RUN git init && \
  git remote add origin https://github.com/bitcoin-core/guix.sigs.git && \
  git fetch --depth 1 origin ca2147e5f8e360bc42b7eff57f4759afe723e59a && \
  git checkout FETCH_HEAD && \
  for file in builder-keys/*; do gpg --import "$file"; done

# pull and verify binaries
COPY version /version
WORKDIR /work
RUN VERSION="$(sed 's/-.*//' /version)" && \
  wget --progress=dot:giga "https://bitcoincore.org/bin/bitcoin-core-${VERSION}/bitcoin-${VERSION}-x86_64-linux-gnu.tar.gz" && \
  wget --progress=dot:giga "https://bitcoincore.org/bin/bitcoin-core-${VERSION}/SHA256SUMS" && \
  wget --progress=dot:giga "https://bitcoincore.org/bin/bitcoin-core-${VERSION}/SHA256SUMS.asc"
RUN VERSION="$(sed 's/-.*//' /version)" && \
  gpg --verify SHA256SUMS.asc SHA256SUMS && \
  grep "bitcoin-${VERSION}-x86_64-linux-gnu.tar.gz" SHA256SUMS | sha256sum -c && \
  mkdir bitcoin && \
  tar xzvf "bitcoin-${VERSION}-x86_64-linux-gnu.tar.gz" --strip-components=1 -C bitcoin

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
