# Build Stage
FROM golang:1.22-alpine AS builder

# Install system dependencies required for make and tool installations
RUN apk add --no-cache make curl git

# Install linter and security checker directly into the builder layer
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.56.2
RUN go install github.com/securego/gosec/v2/cmd/gosec@latest

WORKDIR /app

# Copy dependency files first to utilize Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project code (including Makefile)
COPY . .

# Run the full test, lint, and security checklist locally inside the container
# This guarantees 'docker-compose up' will ABORT if tests or security fails.
RUN make check

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o heimdall-server ./cmd/server/main.go

# Production Stage
FROM alpine:latest

WORKDIR /app

# The mock_provider directory is required by the binary at runtime (CWD/mock_provider)
COPY --from=builder /app/mock_provider ./mock_provider
COPY --from=builder /app/heimdall-server .

# Expose the standard port
EXPOSE 8080

# Run the compiled binary
CMD ["./heimdall-server"]
