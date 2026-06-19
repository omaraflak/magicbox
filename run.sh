#!/bin/bash
set -e

# Default environment variables
export MAGICBOX_ROOT=${MAGICBOX_ROOT:-$(pwd)/data}
export MAGICBOX_PORT=${MAGICBOX_PORT:-8081}

echo "1. Rebuilding Magicbox core binary..."
go build -o magicbox cmd/server/main.go

echo "2. Bootstrapping data folders..."
mkdir -p "${MAGICBOX_ROOT}/core/web"

echo "3. Synchronizing web frontend assets..."
if [ -d "web/dist" ]; then
    cp -r web/dist/* "${MAGICBOX_ROOT}/core/web/"
else
    echo "Warning: web/dist does not exist. Please run a Docker build first."
fi

echo "4. Launching Magicbox server (Root: ${MAGICBOX_ROOT}, Port: ${MAGICBOX_PORT})..."
./magicbox
