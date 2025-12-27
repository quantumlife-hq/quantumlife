#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo ""
echo -e "${BLUE}📦 QuantumLife Restore${NC}"
echo "========================="
echo ""

# Check for backup path argument
if [ -z "$1" ]; then
    echo "Usage: ./scripts/restore.sh <backup_directory>"
    echo ""
    echo "Available backups:"
    ls -lt ./backups/ 2>/dev/null | head -10 || echo "  (no backups found)"
    exit 1
fi

BACKUP_DIR="$1"

# Verify backup exists
if [ ! -d "$BACKUP_DIR" ]; then
    echo -e "${RED}❌ Backup directory not found: $BACKUP_DIR${NC}"
    exit 1
fi

if [ ! -f "$BACKUP_DIR/quantumlife.db" ]; then
    echo -e "${RED}❌ Database backup not found in $BACKUP_DIR${NC}"
    exit 1
fi

echo -e "${YELLOW}⚠️  WARNING: This will replace all current data!${NC}"
echo ""
read -p "Are you sure you want to restore from $BACKUP_DIR? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Restore cancelled."
    exit 0
fi

# Stop services
echo ""
echo -e "${BLUE}Stopping services...${NC}"
docker compose stop quantumlife

# Restore database
echo ""
echo -e "${BLUE}Restoring database...${NC}"
docker compose cp "$BACKUP_DIR/quantumlife.db" quantumlife:/data/quantumlife.db
echo -e "${GREEN}✓ Database restored${NC}"

# Restore Qdrant if backup exists
if [ -f "$BACKUP_DIR/qdrant.tar.gz" ]; then
    echo ""
    echo -e "${BLUE}Restoring vector database...${NC}"
    docker compose stop qdrant
    cat "$BACKUP_DIR/qdrant.tar.gz" | docker compose exec -T qdrant tar -xzf - -C /
    echo -e "${GREEN}✓ Vector database restored${NC}"
fi

# Restart services
echo ""
echo -e "${BLUE}Restarting services...${NC}"
docker compose up -d

echo ""
echo "==========================================="
echo -e "${GREEN}✅ Restore complete!${NC}"
echo "==========================================="
echo ""
echo "Services are restarting..."
echo "Check status with: docker compose ps"
echo ""
