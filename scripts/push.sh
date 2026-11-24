#!/bin/bash
set -e

# Configuration
IMAGE_NAME="${IMAGE_NAME:-cruciblehq/hub}"
TAG="${TAG:-latest}"

echo "Building and pushing ${IMAGE_NAME}:${TAG}..."

# Build and push multi-platform image directly to Docker Hub
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag "${IMAGE_NAME}:${TAG}" \
  --push \
  .

echo "Successfully pushed ${IMAGE_NAME}:${TAG}"
