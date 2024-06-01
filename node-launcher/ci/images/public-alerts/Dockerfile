########################################################################################
# Build
########################################################################################

FROM golang:1.22 as builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
# Get dependancies - will also be cached if we won't change mod/sum
RUN go mod download

# Copy the source code
COPY . .

# Build the binary.
RUN CGO_ENABLED=0 GOOS=linux go build -o alert ./cmd/alert


########################################################################################
# Deploy
########################################################################################

FROM alpine:3.15 

# run as non-root user
RUN addgroup -S app && adduser -S app -G app
# Set a working directory that the non-root user can access
WORKDIR /home/app

# Copy the binary from the builder stage to the accessible directory
COPY --from=builder /app/alert .
# TODO: switch to volume in provider 
RUN mkdir data 
# Switch to non-root user
USER app

# Set the binary as the entrypoint of the container
ENTRYPOINT ["./alert"]
