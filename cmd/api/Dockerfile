# syntax=docker/dockerfile:1

# =============================================================================
# STAGE 1: BUILDER
# Simpler builder image for local builds (no Buildx-specific platform flags)
# =============================================================================
FROM golang:1.25-alpine AS builder

WORKDIR /app

# 1. Declare Docker Buildx automatic ARG variables
# When you build with --platform linux/amd64, Docker will automatically set "linux" for TARGETOS and "amd64" for TARGETARCH
ARG TARGETOS
ARG TARGETARCH

# Install git and tzdata (will be copied to runtime image)
RUN apk add --no-cache git tzdata

# 2. Copy dependencies (vendor mode)
COPY go.mod go.sum ./
COPY vendor/ vendor/

# 3. Copy Source Code
COPY . .

# 4. Install Swag and Generate Swagger Documentation
RUN go install github.com/swaggo/swag/cmd/swag@v1.8.12
RUN swag init -g cmd/api/main.go --parseVendor

# 5. Build Binary (vendor mode)
RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
    -mod=vendor \
    -ldflags="-s -w" \
    -o api ./cmd/api

# =============================================================================
# STAGE 2: RUNTIME (Distroless Static - Ultra lightweight & Secure)
# - No shell, no package manager → Minimal attack surface
# - Image size ~2MB (vs Alpine ~20MB)
# - Contains only: binary + ca-certs + timezone + user
# =============================================================================
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app

# Copy timezone data (because distroless static doesn't have it by default)
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=Asia/Ho_Chi_Minh

# Copy the compiled binary for the correct architecture
COPY --from=builder --chown=nonroot:nonroot /app/api .

# Run as non-root (UID 65532)
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["./api"]