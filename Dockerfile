# stage 1: Build the Go application
FROM golang:1.23-alpine AS builder

# set necessary environment variables
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app

# copy go.mod and go.sum files
COPY ./go.mod ./go.sum ./

# download dependencies
RUN go mod download

# copy the entire project (only what's needed)
COPY . .

# tidy up modules (ensure no unused dependencies)
RUN go mod tidy

# build the application
RUN go build -o /app/main .

# stage 2: Create the final lightweight image
FROM alpine:latest

WORKDIR /app

# install certificates (if your app makes HTTPS requests)
RUN apk --no-cache add ca-certificates curl

# copy the executable from the builder stage
COPY --from=builder /app/main .

# copy necessary files
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/config.yaml .

# ensure config directory exists
RUN mkdir -p ./config

# copy JSON config files
COPY --from=builder /app/config/meets.json ./config/meets.json
COPY --from=builder /app/config/meet_creds.json ./config/meet_creds.json

# expose the application port
EXPOSE 8080

# set the entrypoint
CMD ["./main"]
