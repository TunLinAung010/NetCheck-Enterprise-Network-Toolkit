#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:-1.0.0}"
OUTPUT_DIR="${OUTPUT_DIR:-dist}"

PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

echo "Building NetCheck v${VERSION}"

mkdir -p "${OUTPUT_DIR}"

for PLATFORM in "${PLATFORMS[@]}"; do
    OS="${PLATFORM%%/*}"
    ARCH="${PLATFORM##*/}"
    
    EXT=""
    if [ "${OS}" = "windows" ]; then
        EXT=".exe"
    fi
    
    OUTPUT="${OUTPUT_DIR}/netcheck-${OS}-${ARCH}${EXT}"
    
    echo "Building for ${OS}/${ARCH}..."
    
    GOOS="${OS}" GOARCH="${ARCH}" CGO_ENABLED=0 \
        go build -ldflags="-s -w -X main.version=${VERSION}" \
        -o "${OUTPUT}" ./cmd/netcheck
    
    sha256sum "${OUTPUT}" > "${OUTPUT}.sha256"
done

echo "Build complete. Artifacts in ${OUTPUT_DIR}/"
ls -la "${OUTPUT_DIR}/"
