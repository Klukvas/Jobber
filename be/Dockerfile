# Multi-stage build for Go backend
# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o jobber-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o jobber-seed ./cmd/seed

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS requests (needed for S3) and chromium for PDF generation
RUN apk --no-cache add ca-certificates tzdata chromium
ENV CHROME_BIN=/usr/bin/chromium-browser

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Copy binaries from builder
COPY --from=builder /app/jobber-api .
COPY --from=builder /app/jobber-seed .
COPY --from=builder /app/migrations ./migrations
# plans.yaml is the source of truth for plan limits (hard-coded defaults are only a fallback)
COPY --from=builder /app/config ./config

# Change ownership
RUN chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

# Commit the image was built from. Sentry falls back to it when SENTRY_RELEASE is
# empty, so events always name the code that is actually running (rollbacks too).
# Declared last so a new commit does not invalidate the cached layers above.
ARG BUILD_REVISION=
ENV BUILD_REVISION=$BUILD_REVISION

CMD ["./jobber-api"]
