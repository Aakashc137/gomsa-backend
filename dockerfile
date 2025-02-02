# gomsa-backend/Dockerfile

# Stage 1: Build the binary
FROM golang:1.23-alpine AS builder
WORKDIR /app

# Copy go.mod and go.sum to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the backend binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -o backend .

# Stage 2: Create a minimal image
FROM alpine:latest
WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/backend .

# Expose the port that your backend listens on (for example, 50051 for gRPC)
EXPOSE 50051

# Run the binary
ENTRYPOINT ["./backend"]