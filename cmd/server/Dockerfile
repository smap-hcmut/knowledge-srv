# syntax=docker/dockerfile:1

# =============================================================================
# STAGE 1: BUILDER
# --platform=$BUILDPLATFORM: Go compiler runs natively (no QEMU emulation)
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

WORKDIR /app

ARG TARGETOS
ARG TARGETARCH

# Install swag alongside apk deps — cached before source COPY
RUN apk add --no-cache git tzdata && \
    go install github.com/swaggo/swag/cmd/swag@v1.8.12

COPY go.mod go.sum ./
COPY vendor/ vendor/

COPY . .

# Generate Swagger docs (swag already installed above)
RUN swag init -g cmd/server/main.go --parseVendor

# Build Binary (vendor mode) with go-build cache mount
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
    -mod=vendor \
    -trimpath \
    -ldflags="-s -w" \
    -o knowledge-srv ./cmd/server

# =============================================================================
# STAGE 2: RUNTIME (Distroless Static)
# =============================================================================
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=Asia/Ho_Chi_Minh

COPY --from=builder --chown=nonroot:nonroot /app/knowledge-srv .

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["./knowledge-srv"]
