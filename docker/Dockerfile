# Stage 1: Build the Go application
FROM golang:1.23-alpine AS builder

# Set necessary environment variables
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the entire project (only what's needed)
COPY . .

# Tidy up modules (ensure no unused dependencies)
RUN go mod tidy

# Build the application
RUN go build -o /app/main .

# Stage 2: Create the final lightweight image
FROM alpine:latest

WORKDIR /app

# Install certificates (if your app makes HTTPS requests)
RUN apk --no-cache add ca-certificates

# Copy the executable from the builder stage
COPY --from=builder /app/main .

# Copy necessary files
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/config.yaml .

# Ensure config directory exists
RUN mkdir -p ./config

# Copy JSON config files
COPY --from=builder /app/config/meets.json ./config/meets.json
COPY --from=builder /app/config/meet_creds.json ./config/meet_creds.json

# Expose the application port
EXPOSE 8080

# Set the entrypoint
CMD ["./main"]
