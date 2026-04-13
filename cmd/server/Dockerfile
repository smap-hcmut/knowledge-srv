# syntax=docker/dockerfile:1

# =============================================================================
# STAGE 1: BUILDER
# =============================================================================
FROM golang:1.25-alpine AS builder

WORKDIR /app

ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache git tzdata

COPY go.mod go.sum ./
COPY vendor/ vendor/

COPY . .

# Install Swag and Generate Swagger Documentation
RUN go install github.com/swaggo/swag/cmd/swag@v1.8.12
RUN swag init -g cmd/server/main.go --parseVendor

# Build Binary (vendor mode)
RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
    -mod=vendor \
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
