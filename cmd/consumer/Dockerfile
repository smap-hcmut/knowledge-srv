# syntax=docker/dockerfile:1

# =============================================================================
# STAGE 1: BUILDER
# Simpler builder image for local builds (no Buildx-specific platform flags)
# =============================================================================
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Docker Buildx automatic ARG variables for cross-compilation
ARG TARGETOS
ARG TARGETARCH

# Install git and tzdata (tzdata will be copied to runtime image)
RUN apk add --no-cache git tzdata

# Download dependencies (no BuildKit mounts for simplicity)
COPY go.mod go.sum ./
RUN go mod download

# Copy Source Code
COPY . .

# Build Binary with optimizations (no BuildKit mounts)
RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
    -ldflags="-s -w" \
    -o consumer ./cmd/consumer

# =============================================================================
# STAGE 2: RUNTIME (Distroless Static - Ultra lightweight & Secure)
# - No shell, no package manager → Minimal attack surface
# - Image size ~2MB base (vs Alpine ~20MB)
# - Contains only: binary + ca-certs + timezone + user
# =============================================================================
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app

# Copy timezone data (distroless static doesn't include it)
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=Asia/Ho_Chi_Minh

# Copy the compiled binary
COPY --from=builder --chown=nonroot:nonroot /app/consumer .

# Run as non-root (UID 65532)
USER nonroot:nonroot

ENTRYPOINT ["./consumer"]
