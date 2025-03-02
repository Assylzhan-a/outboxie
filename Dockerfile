FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the applications
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/example ./cmd/example
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/subscriber ./cmd/subscriber

# Create a minimal runtime image
FROM alpine:3.18

WORKDIR /app

# Install ca-certificates for HTTPS connections
RUN apk --no-cache add ca-certificates

# Copy the binaries from the builder stage
COPY --from=builder /app/bin/example /app/bin/example
COPY --from=builder /app/bin/subscriber /app/bin/subscriber

# Set executable permissions
RUN chmod +x /app/bin/example /app/bin/subscriber

# Create a non-root user to run the application
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser
