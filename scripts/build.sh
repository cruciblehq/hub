#!/bin/bash
set -e

# Create build directory if it doesn't exist
mkdir -p build

# Build for multiple platforms and export as OCI tarball
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --output type=oci,dest=build/image.tar \
  .
