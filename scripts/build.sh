#!/bin/bash
set -e

# Create dist directory if it doesn't exist
mkdir -p dist

# Build for multiple platforms and export as OCI tarball
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --output type=oci,dest=dist/image.tar \
  .
