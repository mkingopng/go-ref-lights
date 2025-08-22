#!/usr/bin/env bash
set -e

echo "Running gofmt..."
gofmt -l -w .

echo "Running goimports..."
goimports -l -w .

echo "Running go vet..."
go vet ./...

echo "Running golangci-lint..."
golangci-lint run --timeout 2m

echo "Running go tests with coverage..."
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

echo "All checks passed!"
