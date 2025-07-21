#!/bin/bash

# Build script for BidZy application
# This script builds multi-platform Docker images

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
IMAGE_NAME="bidzy"
DOCKER_USERNAME=${DOCKER_USERNAME:-"likhith2005"}
VERSION=${VERSION:-"latest"}

echo -e "${GREEN}🚀 Building BidZy Docker Image${NC}"
echo -e "${YELLOW}Image: ${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}${NC}"

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running. Please start Docker and try again.${NC}"
    exit 1
fi

# Check if buildx is available
if ! docker buildx version > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker buildx is not available. Please install Docker buildx.${NC}"
    exit 1
fi

# Create and use a new builder instance for multi-platform builds
echo -e "${YELLOW}📦 Setting up multi-platform builder...${NC}"
docker buildx create --name bidzy-builder --use 2>/dev/null || docker buildx use bidzy-builder

# Build for multiple platforms
echo -e "${YELLOW}🔨 Building for multiple platforms...${NC}"
docker buildx build \
    --platform linux/amd64,linux/arm64 \
    --tag "${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}" \
    --tag "${DOCKER_USERNAME}/${IMAGE_NAME}:latest" \
    --push \
    .

echo -e "${GREEN}✅ Build completed successfully!${NC}"
echo -e "${GREEN}📦 Image: ${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}${NC}"
echo -e "${GREEN}📦 Image: ${DOCKER_USERNAME}/${IMAGE_NAME}:latest${NC}"

# Show image info
echo -e "${YELLOW}📋 Image information:${NC}"
docker buildx imagetools inspect "${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}" 