#!/bin/bash

# Proxima installation script
# Installs binary to $HOME/.local/bin

set -e

# Variables
BINARY_NAME="proxima"
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="$HOME/.local/bin"
BINARY_PATH="$SOURCE_DIR/bin/$BINARY_NAME"
TARGET_PATH="$BIN_DIR/$BINARY_NAME"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Installing Proxima...${NC}"

# Check if binary exists
if [ ! -f "$BINARY_PATH" ]; then
    echo -e "${RED}Error: Binary not found at $BINARY_PATH${NC}"
    echo -e "${YELLOW}Run 'make build' first to compile the project.${NC}"
    exit 1
fi

# Create installation directory if it doesn't exist
if [ ! -d "$BIN_DIR" ]; then
    echo -e "${YELLOW}Creating directory $BIN_DIR...${NC}"
    mkdir -p "$BIN_DIR"
fi

# Copy binary (overwrite without asking)
echo -e "${YELLOW}Copying binary to $TARGET_PATH...${NC}"
cp "$BINARY_PATH" "$TARGET_PATH"

# Make executable
chmod +x "$TARGET_PATH"

# Check if $HOME/.local/bin is in PATH
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    echo -e "${YELLOW}Warning: $BIN_DIR is not in your PATH${NC}"
    echo -e "${YELLOW}Add the following line to your ~/.bashrc or ~/.zshrc:${NC}"
    echo -e "${YELLOW}export PATH=\"\$PATH:$BIN_DIR\"${NC}"
    echo -e "${YELLOW}And run: source ~/.bashrc (or source ~/.zshrc)${NC}"
fi

echo -e "${GREEN}Installation completed successfully!${NC}"
echo -e "${GREEN}Proxima installed at: $TARGET_PATH${NC}"
echo -e "${GREEN}Run 'proxima --help' to verify installation.${NC}"