# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS builder

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
FROM ghcr.io/borgmatic-collective/borgmatic:1.9.12

# Copy the Go binary from the builder stage
COPY --from=builder /app/borg-exporter /usr/local/bin/borg-exporter

ENTRYPOINT ["/usr/local/bin/borg-exporter"]