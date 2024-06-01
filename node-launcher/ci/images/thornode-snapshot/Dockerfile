########################################################################################
# Build
########################################################################################

FROM golang:1.21 AS build

RUN mkdir /app
COPY go.* /app/
COPY *.go /app/
WORKDIR /app

RUN CGO_ENABLED=0 GOOS=linux go build ./...


########################################################################################
# Deploy
########################################################################################

FROM alpine:3.15 AS deploy

# run as non-root user
RUN addgroup -S app && adduser -S app -G app
USER app

WORKDIR /app
COPY --from=build /app/thornode-snapshot .

ENTRYPOINT ["/app/thornode-snapshot"]

# vi: ft=dockerfile
