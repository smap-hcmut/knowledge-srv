#!/bin/bash

# SMAP Knowledge Service - Build and Push Script
# Usage: ./build-api.sh [build-push|login|help]

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
REGISTRY="${HARBOR_REGISTRY:-registry.tantai.dev}"
PROJECT="smap"
SERVICE="knowledge-srv"
DOCKERFILE="cmd/server/Dockerfile"
PLATFORM="${PLATFORM:-linux/amd64}"

# Harbor credentials (set HARBOR_USERNAME and HARBOR_PASSWORD in ~/.zshrc)
HARBOR_USER="${HARBOR_USERNAME:?HARBOR_USERNAME is not set. Export it in ~/.zshrc}"
HARBOR_PASS="${HARBOR_PASSWORD:?HARBOR_PASSWORD is not set. Export it in ~/.zshrc}"

# Helper functions
info() { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Ensure we run from project root
cd "$(dirname "$0")/.."

# Generate image tag with timestamp
generate_tag() {
    date +"%y%m%d-%H%M%S"
}

# Get full image name
get_image_name() {
    local tag="${1:-$(generate_tag)}"
    echo "${REGISTRY}/${PROJECT}/${SERVICE}:${tag}"
}

# Login to Harbor registry
login() {
    info "Logging into Harbor registry: $REGISTRY"

    echo "$HARBOR_PASS" | docker login "$REGISTRY" -u "$HARBOR_USER" --password-stdin

    if [ $? -eq 0 ]; then
        success "Logged in successfully"
    else
        error "Login failed"
        exit 1
    fi
}

# Build and push image using classic docker build + push (no buildx, no OCI index)
build_and_push() {
    # Basic checks
    if ! command -v docker &> /dev/null; then
        error "Docker is not installed"
        exit 1
    fi

    if [ ! -f "$DOCKERFILE" ]; then
        error "Dockerfile not found: $DOCKERFILE"
        exit 1
    fi

    if [ ! -f "go.mod" ]; then
        error "go.mod not found - not in project root?"
        exit 1
    fi

    # Ensure logged in
    if ! docker info 2>/dev/null | grep -q "$REGISTRY"; then
        warning "Not logged in to $REGISTRY, attempting login..."
        login
    fi

    local tag
    tag=$(generate_tag)
    local image_name
    image_name=$(get_image_name "$tag")
    local latest_name
    latest_name=$(get_image_name latest)

    info "Registry:   $REGISTRY"
    info "Image:      $image_name"
    info "Platform:   $PLATFORM"
    info "Dockerfile: $DOCKERFILE"
    echo ""

    info "Starting docker build (BuildKit enabled)..."
    DOCKER_BUILDKIT=1 docker build \
        --tag "$image_name" \
        --tag "$latest_name" \
        --file "$DOCKERFILE" \
        --progress=plain \
        --build-arg TARGETOS=linux \
        --build-arg TARGETARCH=${PLATFORM#linux/} \
        .

    if [ $? -ne 0 ]; then
        error "Docker build failed"
        exit 1
    fi

    info "Pushing image: $image_name"
    docker push "$image_name"
    if [ $? -ne 0 ]; then
        error "Failed to push image: $image_name"
        exit 1
    fi

    info "Pushing image: $latest_name"
    docker push "$latest_name"
    if [ $? -ne 0 ]; then
        error "Failed to push image: $latest_name"
        exit 1
    fi

    success "Image built and pushed successfully!"
    echo ""
    info "Tagged images:"
    echo "  - $image_name"
    echo "  - $latest_name"
    echo ""
    info "To deploy, update the image in manifests/api-deployment.yaml:"
    echo "  image: $image_name"
    echo ""
    info "Then apply:"
    echo "  kubectl apply -f manifests/"
}

# Show help
show_help() {
    cat << EOF
${GREEN}SMAP Knowledge API - Build and Push Script${NC}

Usage: $0 [command]

Commands:
    build-push    Build and push Docker image (default)
    login         Login to Zot registry
    help          Show this help

Examples:
    $0                    # Build and push
    $0 build-push         # Build and push
    $0 login              # Login to Zot registry

Configuration:
    Registry:   $REGISTRY
    Project:    $PROJECT
    Service:    $SERVICE
    Platform:   $PLATFORM
    Dockerfile: $DOCKERFILE

Image Format:
    ${REGISTRY}/${PROJECT}/${SERVICE}:YYMMDD-HHMMSS
    ${REGISTRY}/${PROJECT}/${SERVICE}:latest

Environment Variables:
    REGISTRY            Harbor registry URL (default: registry.tantai.dev)
    HARBOR_USERNAME     Registry username
    HARBOR_PASSWORD     Registry password
    PLATFORM            Build platform (default: linux/amd64)

Build Process:
    1. Builder stage   - Download deps, generate Swagger, compile Go binary
    2. Runtime stage   - Distroless static image (~2MB) with binary only

Features:
    - BuildKit enabled for optimized builds
    - Multi-stage build for minimal image size
    - Distroless runtime (no shell, minimal attack surface)
    - Non-root user (UID 65532)
    - Cross-platform compilation support (linux/amd64, linux/arm64, etc.)
    - Optimized for Harbor registry

EOF
}

# Main
case "${1:-build-push}" in
    build-push)
        build_and_push
        ;;
    login)
        login
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        error "Unknown command: $1"
        echo ""
        show_help
        exit 1
        ;;
esac
