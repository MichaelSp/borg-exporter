# Stage 1: Build the Go binary
FROM golang:1.26-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY cmd/ cmd/
COPY pkg/ pkg/

# Build the Go binary
ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target="/root/.cache/go-build" go build -o borg-exporter ./cmd/main.go

# Stage 2: Create the final image
FROM ghcr.io/borgmatic-collective/borgmatic:2.1.7

# Borgmatic's Alpine 3.22 base only carries PostgreSQL 16 and 17 clients.
# Use 3.23 packages so backup can match PostgreSQL server majors 16 through 18.
RUN sed -i 's/v3\.22/v3.23/g' /etc/apk/repositories \
    && apk upgrade --no-cache \
    && apk add --no-cache postgresql16-client postgresql17-client postgresql18-client

# Copy the Go binary from the builder stage
COPY --from=builder /app/borg-exporter /usr/local/bin/borg-exporter
COPY --chmod=0755 scripts/pg_dumpall-for-server /usr/local/bin/pg_dumpall-for-server

ENTRYPOINT ["/usr/local/bin/borg-exporter"]
