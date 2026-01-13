# =============================================================================
# TxHammer Dockerfile
# StableNet Stress Testing Tool
# =============================================================================
#
# Build:
#   docker build -t txhammer .
#
# Build with version info:
#   docker build --build-arg VERSION=v1.0.0 -t txhammer:v1.0.0 .
#
# Corporate network (SSL inspection) workarounds:
#   Option 1: Use vendor directory (recommended)
#     go mod vendor
#     docker build -t txhammer .
#
#   Option 2: Build outside Docker
#     CGO_ENABLED=0 go build -o txhammer ./cmd
#     docker build -f Dockerfile.scratch -t txhammer .
#
# =============================================================================

# Build stage
FROM golang:1.24-bookworm AS builder

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Copy vendor directory if exists (for corporate network builds)
COPY vendor* ./vendor/

# Copy source code and vendor
COPY . .

# Build arguments for version info
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
ARG TARGETARCH=amd64

# Determine build mode: use vendor if available, otherwise download
# Set GOPROXY for fallback and disable network if using vendor
RUN if [ -d "vendor" ] && [ -f "vendor/modules.txt" ]; then \
        echo "Building with vendored dependencies..."; \
        CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
            -mod=vendor \
            -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${BUILD_DATE}" \
            -o txhammer ./cmd; \
    else \
        echo "Building with downloaded dependencies..."; \
        GOPROXY=https://proxy.golang.org,direct go mod download && \
        CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
            -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${BUILD_DATE}" \
            -o txhammer ./cmd; \
    fi

# =============================================================================
# Final stage - minimal runtime image
# =============================================================================
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user for security
RUN adduser -D -g '' txhammer
USER txhammer

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/txhammer /app/txhammer

# Create reports directory
RUN mkdir -p /app/reports

# Expose reports volume
VOLUME ["/app/reports"]

# Set entrypoint
ENTRYPOINT ["/app/txhammer"]

# Default command (show help)
CMD ["--help"]

# =============================================================================
# Labels
# =============================================================================
LABEL org.opencontainers.image.title="TxHammer"
LABEL org.opencontainers.image.description="StableNet Stress Testing Tool"
LABEL org.opencontainers.image.source="https://github.com/0xmhha/txhammer"
