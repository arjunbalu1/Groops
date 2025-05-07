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

WORKDIR /app

# Copy the built binary from the builder
COPY --from=builder /app/app .

# Expose the port your app runs on
EXPOSE 8080

# Run the binary
CMD ["./app"] 