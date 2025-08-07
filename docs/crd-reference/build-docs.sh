#!/bin/bash

# Build script for CRD reference documentation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Building Nephoran Intent Operator CRD Documentation${NC}"

# Check if Python is installed
if ! command -v python3 &> /dev/null; then
    echo -e "${RED}Python 3 is required but not installed${NC}"
    exit 1
fi

# Create virtual environment if it doesn't exist
if [ ! -d "venv" ]; then
    echo -e "${YELLOW}Creating Python virtual environment...${NC}"
    python3 -m venv venv
fi

# Activate virtual environment
echo -e "${YELLOW}Activating virtual environment...${NC}"
source venv/bin/activate || . venv/Scripts/activate 2>/dev/null || {
    echo -e "${RED}Failed to activate virtual environment${NC}"
    exit 1
}

# Install dependencies
echo -e "${YELLOW}Installing dependencies...${NC}"
pip install --upgrade pip
pip install -r requirements.txt

# Generate CRD documentation from Go types
echo -e "${YELLOW}Generating CRD documentation from Go types...${NC}"
if command -v controller-gen &> /dev/null; then
    cd ../..
    controller-gen crd:maxDescLen=0 paths="./api/v1/..." output:crd:artifacts:config=docs/crd-reference/generated/crds
    cd docs/crd-reference
else
    echo -e "${YELLOW}controller-gen not found, skipping CRD generation${NC}"
fi

# Build the documentation
echo -e "${YELLOW}Building documentation site...${NC}"
mkdocs build

# Serve locally for preview (optional)
if [ "$1" = "--serve" ]; then
    echo -e "${GREEN}Starting local server at http://localhost:8000${NC}"
    mkdocs serve
elif [ "$1" = "--deploy" ]; then
    echo -e "${GREEN}Deploying to GitHub Pages...${NC}"
    mkdocs gh-deploy --force
else
    echo -e "${GREEN}Documentation built successfully!${NC}"
    echo -e "${GREEN}Output: site/${NC}"
    echo ""
    echo "To preview locally, run: $0 --serve"
    echo "To deploy to GitHub Pages, run: $0 --deploy"
fi