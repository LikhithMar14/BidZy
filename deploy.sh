#!/bin/bash

# Deployment script for BidZy application on EC2
# This script sets up the application on an EC2 instance

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Deploying BidZy to EC2${NC}"

# Check if .env file exists
if [ ! -f .env ]; then
    echo -e "${RED}❌ .env file not found. Please create one based on env.example${NC}"
    exit 1
fi

# Load environment variables
source .env

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running. Please start Docker and try again.${NC}"
    exit 1
fi

# Check if docker-compose is available
if ! docker-compose version > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker Compose is not available. Please install Docker Compose.${NC}"
    exit 1
fi

# Pull the latest image
echo -e "${YELLOW}📦 Pulling latest image...${NC}"
docker-compose -f docker-compose.production.yml pull

# Stop existing containers
echo -e "${YELLOW}🛑 Stopping existing containers...${NC}"
docker-compose -f docker-compose.production.yml down

# Start the application
echo -e "${YELLOW}🚀 Starting application...${NC}"
docker-compose -f docker-compose.production.yml up -d

# Wait for services to be healthy
echo -e "${YELLOW}⏳ Waiting for services to be healthy...${NC}"
sleep 30

# Check service status
echo -e "${YELLOW}📋 Checking service status...${NC}"
docker-compose -f docker-compose.production.yml ps

# Check application health
echo -e "${YELLOW}🏥 Checking application health...${NC}"
if curl -f http://localhost:8080/health > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Application is healthy!${NC}"
else
    echo -e "${RED}❌ Application health check failed${NC}"
    echo -e "${YELLOW}📋 Checking logs...${NC}"
    docker-compose -f docker-compose.production.yml logs app
    exit 1
fi

echo -e "${GREEN}🎉 Deployment completed successfully!${NC}"
echo -e "${GREEN}🌐 Application is running on http://localhost:8080${NC}"
echo -e "${GREEN}🏥 Health check: http://localhost:8080/health${NC}" 