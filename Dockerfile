# Stage 1: Build the Go binary
FROM golang:1.23-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY cmd/main.go /app/cmd/main.go
COPY pkg /app/pkg

# Build the Go binary
RUN go build -o borg-exporter ./cmd/main.go

# Stage 2: Create the final image
FROM alpine:latest

# Copy the Go binary from the builder stage
COPY --from=builder /borg-exporter /usr/local/bin/borg-exporter
COPY --from=ghcr.io/borgmatic-collective/borgmatic:1.9.4 /usr/local/bin/borgmatic /bin/borgmatic
COPY --from=ghcr.io/borgmatic-collective/borgmatic:1.9.4 /usr/local/bin/borg /bin/borg
