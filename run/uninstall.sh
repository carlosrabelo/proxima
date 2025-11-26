#!/bin/bash

# Proxima uninstallation script
# Removes binary from $HOME/.local/bin

set -e

# Variables
BINARY_NAME="proxima"
BIN_DIR="$HOME/.local/bin"
TARGET_PATH="$BIN_DIR/$BINARY_NAME"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Uninstalling Proxima...${NC}"

# Check if binary exists
if [ ! -f "$TARGET_PATH" ]; then
    echo -e "${RED}Warning: Proxima not found at $TARGET_PATH${NC}"
    echo -e "${YELLOW}It seems Proxima is not installed or was installed elsewhere.${NC}"
    exit 0
fi

# Remove binary (without confirmation)
echo -e "${YELLOW}Removing binary...${NC}"
rm -f "$TARGET_PATH"

echo -e "${GREEN}Proxima uninstalled successfully!${NC}"
echo -e "${GREEN}Binary removed from: $TARGET_PATH${NC}"

# Check if directory is empty and can be removed
if [ -d "$BIN_DIR" ] && [ -z "$(ls -A "$BIN_DIR" 2>/dev/null)" ]; then
    echo -e "${YELLOW}Directory $BIN_DIR is empty. You can remove it manually if desired.${NC}"
fi