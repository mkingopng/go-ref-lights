# Dockerfile for the Go application
# Build stage
FROM golang:1.17-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main .

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the executable
COPY --from=builder /app/main .

# Copy necessary files
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/config.yaml .

# Expose the port
EXPOSE 8080

# Run the application
CMD ["./main"]
