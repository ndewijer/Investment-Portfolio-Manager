#!/bin/bash
# Docker Integration Test Script
# Tests Docker container builds, startup, and integration

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo -e "${YELLOW}⚠️  Docker not available - skipping Docker integration tests${NC}"
    echo "   Install Docker to run these tests locally"
    exit 0
fi

# Cleanup function
cleanup() {
    echo -e "${BLUE}🧹 Cleaning up containers...${NC}"
    docker compose down -v --remove-orphans 2>/dev/null || true
}

# Set trap to cleanup on exit
trap cleanup EXIT

echo -e "${BLUE}🐳 Running Docker integration tests...${NC}"

# Build containers
echo -e "${BLUE}📦 Building Docker images...${NC}"
if ! docker compose build --no-cache --quiet; then
    echo -e "${RED}❌ Docker build failed${NC}"
    exit 1
fi

# Start containers
echo -e "${BLUE}🚀 Starting containers...${NC}"
if ! docker compose up -d; then
    echo -e "${RED}❌ Failed to start containers${NC}"
    exit 1
fi

# Give containers time to initialize
sleep 3

# Test 1: Backend health (from inside container)
echo -e "${BLUE}⏳ Waiting for backend to be ready (testing from inside container)...${NC}"
for i in {1..60}; do
    if docker compose exec -T backend python -c "import urllib.request; urllib.request.urlopen('http://localhost:5000/api/system/health')" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Backend is healthy${NC}"
        break
    fi

    if [ $i -eq 60 ]; then
        echo -e "${RED}❌ Backend health check timeout${NC}"
        echo -e "${YELLOW}Backend logs:${NC}"
        docker compose logs backend
        echo -e "${YELLOW}Frontend logs:${NC}"
        docker compose logs frontend
        exit 1
    fi

    sleep 1
done

# Test 2: Verify backend response
echo -e "${BLUE}🔍 Verifying backend response...${NC}"
HEALTH=$(docker compose exec -T backend python -c "import urllib.request; print(urllib.request.urlopen('http://localhost:5000/api/system/health').read().decode())")
echo "Backend response: $HEALTH"

if ! echo "$HEALTH" | grep -q '"status"[[:space:]]*:[[:space:]]*"healthy"'; then
    echo -e "${RED}❌ Backend health response invalid${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Backend health verified${NC}"

# Test 3: Frontend serves static content
echo -e "${BLUE}🌐 Testing frontend static content...${NC}"
if ! curl -f -s http://localhost/ > /dev/null; then
    echo -e "${RED}❌ Frontend not serving content${NC}"
    docker compose logs frontend
    exit 1
fi
echo -e "${GREEN}✅ Frontend serving content${NC}"

# Test 4: Frontend proxy to backend
echo -e "${BLUE}🔗 Testing frontend-backend proxy integration...${NC}"
PROXY_RESPONSE=$(curl -s http://localhost/api/system/health)
echo "Proxy response: $PROXY_RESPONSE"

if ! echo "$PROXY_RESPONSE" | grep -q '"status"[[:space:]]*:[[:space:]]*"healthy"'; then
    echo -e "${RED}❌ Frontend proxy not working${NC}"
    docker compose logs frontend
    exit 1
fi
echo -e "${GREEN}✅ Frontend proxy working${NC}"

# All tests passed
echo -e "${GREEN}✅ All Docker integration tests passed!${NC}"
exit 0
