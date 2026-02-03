#!/bin/sh
set -e

# PHP-FPM Optimizer Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/php-fpm/optimizer/main/install.sh | sh

REPO="php-fpm/optimizer"
BINARY_NAME="php-fpm-optimizer"
INSTALL_DIR="/usr/local/bin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

info() {
    printf "${CYAN}[INFO]${NC} %s\n" "$1"
}

success() {
    printf "${GREEN}[OK]${NC} %s\n" "$1"
}

warn() {
    printf "${YELLOW}[WARN]${NC} %s\n" "$1"
}

error() {
    printf "${RED}[ERROR]${NC} %s\n" "$1"
    exit 1
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        linux) OS="linux" ;;
        darwin) OS="darwin" ;;
        *) error "Unsupported operating system: $OS" ;;
    esac

    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac

    PLATFORM="${OS}-${ARCH}"
    info "Detected platform: $PLATFORM"
}

# Get latest release version
get_latest_version() {
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$VERSION" ]; then
        warn "Could not fetch latest version, using 'latest'"
        VERSION="latest"
    else
        info "Latest version: $VERSION"
    fi
}

# Download and install
install() {
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}-${PLATFORM}"
    
    info "Downloading from: $DOWNLOAD_URL"
    
    TMP_FILE=$(mktemp)
    
    if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE"; then
        rm -f "$TMP_FILE"
        error "Failed to download binary"
    fi
    
    chmod +x "$TMP_FILE"
    
    # Try to install to /usr/local/bin, fall back to ~/bin
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
        success "Installed to ${INSTALL_DIR}/${BINARY_NAME}"
    elif command -v sudo >/dev/null 2>&1; then
        info "Requesting sudo access to install to ${INSTALL_DIR}"
        sudo mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
        success "Installed to ${INSTALL_DIR}/${BINARY_NAME}"
    else
        # Fallback to ~/bin
        INSTALL_DIR="$HOME/bin"
        mkdir -p "$INSTALL_DIR"
        mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
        success "Installed to ${INSTALL_DIR}/${BINARY_NAME}"
        warn "Add ${INSTALL_DIR} to your PATH if not already present"
    fi
}

# Verify installation
verify() {
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        success "Installation verified!"
        echo ""
        info "Run '$BINARY_NAME --help' to get started"
    else
        warn "Binary installed but not in PATH. You may need to restart your shell."
        info "Or run directly: ${INSTALL_DIR}/${BINARY_NAME}"
    fi
}

main() {
    echo ""
    printf "${CYAN}PHP-FPM Optimizer Installer${NC}\n"
    echo "─────────────────────────────"
    echo ""
    
    detect_platform
    get_latest_version
    install
    verify
    
    echo ""
}

main
