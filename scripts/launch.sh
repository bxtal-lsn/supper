#!/bin/bash
# Launcher script for SOPS-TUI

# Define colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to check if a command exists
command_exists() {
    command -v "$1" &>/dev/null
}

# Check dependencies
echo -e "${BLUE}Checking dependencies...${NC}"

# Check for sops
if ! command_exists sops; then
    echo -e "${RED}Error: SOPS is not installed.${NC}"
    echo -e "${YELLOW}Please install SOPS from https://github.com/getsops/sops${NC}"
    echo -e "On macOS: ${GREEN}brew install sops${NC}"
    echo -e "On Linux: ${GREEN}sudo apt-get install sops${NC} or download from GitHub"
    exit 1
fi

# Check for age
if ! command_exists age; then
    echo -e "${RED}Error: age is not installed.${NC}"
    echo -e "${YELLOW}Please install age from https://github.com/FiloSottile/age${NC}"
    echo -e "On macOS: ${GREEN}brew install age${NC}"
    echo -e "On Linux: ${GREEN}sudo apt-get install age${NC} or download from GitHub"
    exit 1
fi

# Check for age-keygen
if ! command_exists age-keygen; then
    echo -e "${RED}Error: age-keygen is not installed.${NC}"
    echo -e "${YELLOW}It should be installed with age.${NC}"
    exit 1
fi

# Get the directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
BINARY_PATH="${SCRIPT_DIR}/../build/sops-tui"

# Check if the binary exists
if [ ! -f "$BINARY_PATH" ]; then
    echo -e "${YELLOW}Binary not found. Building...${NC}"
    cd "${SCRIPT_DIR}/.." || exit 1
    make build
    
    if [ ! -f "$BINARY_PATH" ]; then
        echo -e "${RED}Error: Failed to build the binary.${NC}"
        exit 1
    fi
fi

# Launch the application
echo -e "${GREEN}Starting SOPS-TUI...${NC}"
"$BINARY_PATH"

# Check exit code
EXIT_CODE=$?
if [ $EXIT_CODE -ne 0 ]; then
    echo -e "${RED}SOPS-TUI exited with code $EXIT_CODE${NC}"
    exit $EXIT_CODE
fi
