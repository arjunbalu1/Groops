# Build stage
FROM golang:1.24 as builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the code
COPY . .

# Build the Go app (adjust path if your main is not in ./cmd/server)
RUN go build -o app ./cmd/server

# Final stage
FROM debian:bookworm-slim

# Install CA certificates and timezone data
RUN apt-get update && apt-get install -y ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the built binary from the builder
COPY --from=builder /app/app .

# Copy the assets directory
COPY --from=builder /app/assets /app/assets

# Set environment variables
ENV GIN_MODE=release
ENV TZ=UTC

# Expose the port your app runs on
EXPOSE 8080

# Run the binary
CMD ["./app"] 