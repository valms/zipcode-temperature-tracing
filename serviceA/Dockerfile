# Use Go 1.22 Alpine as base image for building
FROM golang:1.22-alpine AS builder

# Set working directory to /app
WORKDIR /app

# Copy all files from current directory to /app
COPY . .

# Download Go module dependencies
RUN go mod download

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/main.go

# Use Alpine as base image for final container
FROM alpine:latest

# Set working directory to /root
WORKDIR /root/

# Copy the built binary from builder stage
COPY --from=builder /app/main .

# Expose port 8080
EXPOSE 8080

# Set the command to run the application
CMD ["./main"]
