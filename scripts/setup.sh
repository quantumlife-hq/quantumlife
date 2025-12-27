#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo ""
echo -e "${BLUE}⚡ QuantumLife Setup${NC}"
echo "======================="
echo ""

# Check Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker not found. Please install Docker first.${NC}"
    echo "   Visit: https://docs.docker.com/get-docker/"
    exit 1
fi
echo -e "${GREEN}✓ Docker found${NC}"

# Check Docker Compose
if ! docker compose version &> /dev/null; then
    echo -e "${RED}❌ Docker Compose not found.${NC}"
    echo "   Docker Compose V2 is required (docker compose, not docker-compose)"
    exit 1
fi
echo -e "${GREEN}✓ Docker Compose found${NC}"

# Create .env if not exists
if [ ! -f .env ]; then
    echo ""
    echo -e "${YELLOW}Creating .env file from template...${NC}"
    cp .env.example .env
    
    echo ""
    echo -e "${YELLOW}📝 Before starting, you need to configure:${NC}"
    echo ""
    echo "  1. Google OAuth credentials (REQUIRED for Gmail/Calendar)"
    echo "     Get from: https://console.cloud.google.com/apis/credentials"
    echo ""
    echo "  2. Azure OpenAI credentials (OPTIONAL - for complex AI tasks)"
    echo "     Leave empty to use only local Ollama"
    echo ""
    echo "  3. Plaid credentials (OPTIONAL - for Finance integration)"
    echo "     Get from: https://dashboard.plaid.com/"
    echo ""
    echo -e "Edit ${BLUE}.env${NC} file with your credentials, then run:"
    echo -e "  ${GREEN}./scripts/setup.sh${NC}"
    echo ""
    exit 0
else
    echo -e "${GREEN}✓ .env file exists${NC}"
fi

# Check if Google OAuth is configured
if grep -q "GOOGLE_CLIENT_ID=your-client-id" .env 2>/dev/null; then
    echo ""
    echo -e "${YELLOW}⚠️  Google OAuth not configured!${NC}"
    echo "   Edit .env and add your Google OAuth credentials"
    echo "   Gmail and Calendar integration won't work without this."
    echo ""
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Start services
echo ""
echo -e "${BLUE}Starting services...${NC}"
docker compose up -d --build

# Wait for services
echo ""
echo -e "${BLUE}Waiting for services to be ready...${NC}"
sleep 5

# Check if Ollama is running
echo -e "${BLUE}Checking Ollama status...${NC}"
for i in {1..30}; do
    if docker compose exec -T ollama ollama list &>/dev/null; then
        echo -e "${GREEN}✓ Ollama is running${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${YELLOW}⚠️  Ollama not responding yet, continuing...${NC}"
    fi
    sleep 1
done

# Pull embedding model if not present
echo ""
echo -e "${BLUE}Checking for embedding model...${NC}"
if ! docker compose exec -T ollama ollama list 2>/dev/null | grep -q "nomic-embed-text"; then
    echo -e "${YELLOW}Pulling embedding model (nomic-embed-text)...${NC}"
    echo "This may take a few minutes on first run."
    docker compose exec -T ollama ollama pull nomic-embed-text
    echo -e "${GREEN}✓ Embedding model ready${NC}"
else
    echo -e "${GREEN}✓ Embedding model already installed${NC}"
fi

# Pull LLM model if not present
echo ""
echo -e "${BLUE}Checking for LLM model...${NC}"
OLLAMA_MODEL="${OLLAMA_MODEL:-llama3.2}"
if ! docker compose exec -T ollama ollama list 2>/dev/null | grep -q "$OLLAMA_MODEL"; then
    echo -e "${YELLOW}Pulling LLM model ($OLLAMA_MODEL)...${NC}"
    echo "This may take several minutes on first run."
    docker compose exec -T ollama ollama pull "$OLLAMA_MODEL"
    echo -e "${GREEN}✓ LLM model ready${NC}"
else
    echo -e "${GREEN}✓ LLM model already installed${NC}"
fi

# Wait for QuantumLife to be healthy
echo ""
echo -e "${BLUE}Waiting for QuantumLife to be ready...${NC}"
for i in {1..60}; do
    if curl -s http://localhost:8080/api/v1/health &>/dev/null; then
        echo -e "${GREEN}✓ QuantumLife is ready!${NC}"
        break
    fi
    if [ $i -eq 60 ]; then
        echo -e "${YELLOW}⚠️  QuantumLife taking longer than expected...${NC}"
        echo "   Check logs with: docker compose logs quantumlife"
    fi
    sleep 1
done

# Final status
echo ""
echo "==========================================="
echo -e "${GREEN}✅ QuantumLife is running!${NC}"
echo "==========================================="
echo ""
echo -e "🌐 Open ${BLUE}http://localhost:8080${NC} in your browser"
echo ""
echo "Useful commands:"
echo "  View logs:        docker compose logs -f"
echo "  Stop:             docker compose down"
echo "  Restart:          docker compose restart"
echo "  CLI access:       docker compose exec quantumlife /app/ql"
echo "  Backup data:      ./scripts/backup.sh"
echo ""
