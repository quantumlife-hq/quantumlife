#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

# Backup directory
BACKUP_DIR="./backups/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

echo ""
echo -e "${BLUE}📦 QuantumLife Backup${NC}"
echo "========================"
echo ""
echo "Backup location: $BACKUP_DIR"
echo ""

# Check if services are running
if ! docker compose ps | grep -q "quantumlife"; then
    echo -e "${RED}❌ QuantumLife is not running.${NC}"
    echo "   Start with: docker compose up -d"
    exit 1
fi

# Backup SQLite database
echo -e "${BLUE}Backing up database...${NC}"
docker compose exec -T quantumlife cat /data/quantumlife.db > "$BACKUP_DIR/quantumlife.db" 2>/dev/null || {
    echo -e "${RED}Failed to backup database${NC}"
    exit 1
}
echo -e "${GREEN}✓ Database backed up${NC}"

# Get database size
DB_SIZE=$(ls -lh "$BACKUP_DIR/quantumlife.db" | awk '{print $5}')
echo "  Size: $DB_SIZE"

# Backup Qdrant vectors
echo ""
echo -e "${BLUE}Backing up vector database...${NC}"
docker compose exec -T qdrant tar -czf - /qdrant/storage 2>/dev/null > "$BACKUP_DIR/qdrant.tar.gz" || {
    echo -e "${RED}Failed to backup Qdrant${NC}"
    exit 1
}
echo -e "${GREEN}✓ Vector database backed up${NC}"

# Get Qdrant size
QDRANT_SIZE=$(ls -lh "$BACKUP_DIR/qdrant.tar.gz" | awk '{print $5}')
echo "  Size: $QDRANT_SIZE"

# Create metadata
echo ""
echo -e "${BLUE}Creating backup metadata...${NC}"
cat > "$BACKUP_DIR/metadata.json" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "version": "$(docker compose exec -T quantumlife /app/ql version 2>/dev/null || echo 'unknown')",
    "files": {
        "database": "quantumlife.db",
        "vectors": "qdrant.tar.gz"
    }
}
EOF
echo -e "${GREEN}✓ Metadata created${NC}"

# Calculate total size
TOTAL_SIZE=$(du -sh "$BACKUP_DIR" | awk '{print $1}')

echo ""
echo "==========================================="
echo -e "${GREEN}✅ Backup complete!${NC}"
echo "==========================================="
echo ""
echo "Location: $BACKUP_DIR"
echo "Total size: $TOTAL_SIZE"
echo ""
echo "To restore, see: scripts/restore.sh"
echo ""

# List backups
echo "Recent backups:"
ls -lt ./backups/ 2>/dev/null | head -5 || echo "  (no previous backups)"
