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

# Use non-standard port to avoid conflicts with other local containers
export FRONTEND_PORT="${FRONTEND_PORT:-8088}"
TEST_URL="http://localhost:${FRONTEND_PORT}"

echo -e "${BLUE}🐳 Running Docker integration tests (frontend on port ${FRONTEND_PORT})...${NC}"

# Clear any stale buildx cache that might cause "parent snapshot does not exist" errors
echo -e "${BLUE}🧹 Clearing stale build cache...${NC}"
docker builder prune -f --filter "until=24h" 2>/dev/null || true

# Build containers
echo -e "${BLUE}📦 Building Docker images...${NC}"
if ! docker compose build --no-cache --quiet; then
    echo -e "${YELLOW}⚠️  Build failed, retrying with full cache clear...${NC}"
    docker builder prune -af 2>/dev/null || true
    if ! docker compose build --no-cache --quiet; then
        echo -e "${RED}❌ Docker build failed${NC}"
        exit 1
    fi
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
    if docker compose exec -T backend wget -qO- http://localhost:5000/api/system/health > /dev/null 2>&1; then
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
HEALTH=$(docker compose exec -T backend wget -qO- http://localhost:5000/api/system/health)
echo "Backend response: $HEALTH"

if ! echo "$HEALTH" | grep -q '"status"[[:space:]]*:[[:space:]]*"healthy"'; then
    echo -e "${RED}❌ Backend health response invalid${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Backend health verified${NC}"

# Test 3: Frontend serves static content
echo -e "${BLUE}🌐 Testing frontend static content...${NC}"
if ! curl -f -s ${TEST_URL}/ > /dev/null; then
    echo -e "${RED}❌ Frontend not serving content${NC}"
    docker compose logs frontend
    exit 1
fi
echo -e "${GREEN}✅ Frontend serving content${NC}"

# Test 4: Frontend proxy to backend
echo -e "${BLUE}🔗 Testing frontend-backend proxy integration...${NC}"
PROXY_RESPONSE=$(curl -s ${TEST_URL}/api/system/health)
echo "Proxy response: $PROXY_RESPONSE"

if ! echo "$PROXY_RESPONSE" | grep -q '"status"[[:space:]]*:[[:space:]]*"healthy"'; then
    echo -e "${RED}❌ Frontend proxy not working${NC}"
    docker compose logs frontend
    exit 1
fi
echo -e "${GREEN}✅ Frontend proxy working${NC}"

# Test 5: Create test data via API and verify write path
echo -e "${BLUE}📊 Creating test data via API...${NC}"
API_URL="${TEST_URL}/api"

# Create a test portfolio
PORTFOLIO_RESPONSE=$(curl -s -X POST "${API_URL}/portfolio" \
    -H "Content-Type: application/json" \
    -d '{"name": "Test Portfolio", "description": "Integration test portfolio"}')
echo "Create portfolio response: $PORTFOLIO_RESPONSE"

PORTFOLIO_ID=$(echo "$PORTFOLIO_RESPONSE" | grep -o '"id"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')

if [ -z "$PORTFOLIO_ID" ]; then
    echo -e "${RED}❌ Failed to create test portfolio${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Portfolio created (ID: $PORTFOLIO_ID)${NC}"

# Create test funds
create_fund() {
    local name="$1" isin="$2" symbol="$3" currency="$4" exchange="$5" div_type="$6" inv_type="$7"
    local response
    response=$(curl -s -X POST "${API_URL}/fund" \
        -H "Content-Type: application/json" \
        -d "{\"name\": \"${name}\", \"isin\": \"${isin}\", \"symbol\": \"${symbol}\", \"currency\": \"${currency}\", \"exchange\": \"${exchange}\", \"dividendType\": \"${div_type}\", \"investmentType\": \"${inv_type}\"}")
    echo "$response"
}

FUND1_RESPONSE=$(create_fund "Vanguard Total Stock Market ETF" "US9229087690" "VTI" "USD" "NYSE" "CASH" "FUND")
FUND1_ID=$(echo "$FUND1_RESPONSE" | grep -o '"id"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')

FUND2_RESPONSE=$(create_fund "Amundi Prime All Country World UCITS ETF Acc" "LU2089238203" "WEBG" "EUR" "AMS" "NONE" "FUND")
FUND2_ID=$(echo "$FUND2_RESPONSE" | grep -o '"id"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')

FUND3_RESPONSE=$(create_fund "Apple Inc." "US0378331005" "AAPL" "USD" "NASDAQ" "CASH" "STOCK")
FUND3_ID=$(echo "$FUND3_RESPONSE" | grep -o '"id"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')

if [ -z "$FUND1_ID" ] || [ -z "$FUND2_ID" ] || [ -z "$FUND3_ID" ]; then
    echo -e "${RED}❌ Failed to create one or more test funds${NC}"
    echo "Fund1: $FUND1_RESPONSE"
    echo "Fund2: $FUND2_RESPONSE"
    echo "Fund3: $FUND3_RESPONSE"
    exit 1
fi
echo -e "${GREEN}✅ Test funds created${NC}"

# Assign funds to portfolio
for FUND_ID in "$FUND1_ID" "$FUND2_ID" "$FUND3_ID"; do
    ASSIGN_RESPONSE=$(curl -s -X POST "${API_URL}/portfolio/${PORTFOLIO_ID}/fund" \
        -H "Content-Type: application/json" \
        -d "{\"portfolioId\": \"${PORTFOLIO_ID}\", \"fundId\": \"${FUND_ID}\"}")
done
echo -e "${GREEN}✅ Funds assigned to portfolio${NC}"

# Test 6: Verify data can be read back
echo -e "${BLUE}🔍 Verifying test data via read endpoints...${NC}"
FUNDS_RESPONSE=$(curl -s ${API_URL}/fund)

if ! echo "$FUNDS_RESPONSE" | grep -q "Vanguard Total Stock Market ETF"; then
    echo -e "${RED}❌ Expected fund 'Vanguard Total Stock Market ETF' not found${NC}"
    exit 1
fi

if ! echo "$FUNDS_RESPONSE" | grep -q "Amundi Prime All Country World UCITS ETF Acc"; then
    echo -e "${RED}❌ Expected fund 'Amundi Prime All Country World UCITS ETF Acc' not found${NC}"
    exit 1
fi

if ! echo "$FUNDS_RESPONSE" | grep -q "Apple Inc."; then
    echo -e "${RED}❌ Expected fund 'Apple Inc.' not found${NC}"
    exit 1
fi
echo -e "${GREEN}✅ All test funds verified via read endpoint${NC}"

# Test 7: Verify portfolios
echo -e "${BLUE}💼 Verifying portfolios...${NC}"
PORTFOLIOS_RESPONSE=$(curl -s ${API_URL}/portfolio)
PORTFOLIO_COUNT=$(echo "$PORTFOLIOS_RESPONSE" | grep -o '"id"[[:space:]]*:' | wc -l | tr -d '[:space:]')

if [ "$PORTFOLIO_COUNT" -lt 1 ]; then
    echo -e "${RED}❌ No portfolios found${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Portfolios verified (found $PORTFOLIO_COUNT portfolios)${NC}"

# Test 8 (Optional): Run E2E tests against Docker stack
if [ "${RUN_E2E_TESTS}" = "true" ]; then
    echo -e "${BLUE}🎭 Running Playwright E2E tests against Docker stack...${NC}"

    # Check if Playwright is available
    if ! command -v pnpm &> /dev/null; then
        echo -e "${YELLOW}⚠️  pnpm not available - skipping E2E tests${NC}"
    else
        # Install Playwright browsers if needed (will skip if already installed)
        echo -e "${BLUE}📦 Ensuring Playwright browsers are installed...${NC}"
        cd frontend && pnpm exec playwright install chromium --with-deps > /dev/null 2>&1

        # Run E2E tests against Docker stack
        if PLAYWRIGHT_BASE_URL="${TEST_URL}" pnpm run test:e2e; then
            echo -e "${GREEN}✅ E2E tests passed${NC}"
        else
            echo -e "${RED}❌ E2E tests failed${NC}"
            cd ..
            exit 1
        fi
        cd ..
    fi
else
    echo -e "${YELLOW}ℹ️  Skipping E2E tests (set RUN_E2E_TESTS=true to run them)${NC}"
fi

# All tests passed
echo -e "${GREEN}✅ All Docker integration tests passed!${NC}"
exit 0
