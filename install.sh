#!/bin/bash
set -e

# mcp-codewizard installer/updater
# Usage: curl -fsSL https://raw.githubusercontent.com/spetr/mcp-codewizard/main/install.sh | bash

REPO="spetr/mcp-codewizard"
BINARY_NAME="mcp-codewizard"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)     echo "linux";;
        Darwin*)    echo "darwin";;
        MINGW*|MSYS*|CYGWIN*) echo "windows";;
        *)          error "Unsupported operating system: $(uname -s)";;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "amd64";;
        aarch64|arm64)  echo "arm64";;
        *)              error "Unsupported architecture: $(uname -m)";;
    esac
}

# Get latest version from GitHub
get_latest_version() {
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name":' | \
        sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and install
install() {
    echo ""
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║              mcp-codewizard Installer                        ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""

    info "Detecting system..."
    local os=$(detect_os)
    local arch=$(detect_arch)
    echo "  → Operating System: ${os}"
    echo "  → Architecture: ${arch}"

    info "Fetching latest version from GitHub..."
    local version="${VERSION:-$(get_latest_version)}"

    if [ -z "$version" ]; then
        error "Failed to determine version. Set VERSION env var or check network."
    fi
    echo "  → Version: ${version}"
    echo ""

    # Check if already installed
    if command -v "$BINARY_NAME" &> /dev/null; then
        local current=$("$BINARY_NAME" version 2>/dev/null | head -1 | awk '{print $2}' || echo "unknown")
        info "Current installation detected: ${current}"
    fi

    # Determine file extension
    local ext="tar.gz"
    if [ "$os" = "windows" ]; then
        ext="zip"
    fi

    local filename="${BINARY_NAME}-${version}-${os}-${arch}.${ext}"
    local url="https://github.com/${REPO}/releases/download/${version}/${filename}"

    # Create temp directory
    info "Creating temporary directory..."
    local tmpdir=$(mktemp -d)
    trap "rm -rf $tmpdir" EXIT
    echo "  → Temp: ${tmpdir}"

    info "Downloading from GitHub..."
    echo "  → URL: ${url}"
    curl -fsSL --progress-bar "$url" -o "${tmpdir}/${filename}" || error "Download failed"
    echo "  → Downloaded: $(du -h "${tmpdir}/${filename}" | cut -f1)"

    info "Extracting archive..."
    cd "$tmpdir"
    if [ "$ext" = "tar.gz" ]; then
        tar -xzf "$filename"
    else
        unzip -q "$filename"
    fi
    echo "  → Extracted successfully"

    # Install binary
    local binary="${BINARY_NAME}"
    if [ "$os" = "windows" ]; then
        binary="${BINARY_NAME}.exe"
    fi

    if [ ! -f "$binary" ]; then
        error "Binary not found in archive"
    fi

    info "Installing to ${INSTALL_DIR}..."
    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        mv "$binary" "${INSTALL_DIR}/${binary}"
        chmod +x "${INSTALL_DIR}/${binary}"
        echo "  → Installed (user permissions)"
    else
        echo "  → Requesting sudo access..."
        sudo mv "$binary" "${INSTALL_DIR}/${binary}"
        sudo chmod +x "${INSTALL_DIR}/${binary}"
        echo "  → Installed (with sudo)"
    fi

    echo ""
    # Verify installation
    if command -v "$BINARY_NAME" &> /dev/null; then
        info "Verifying installation..."
        local installed_version=$("$BINARY_NAME" version 2>/dev/null || echo "unknown")
        echo ""
        echo "╔══════════════════════════════════════════════════════════════╗"
        echo "║  ✓ Installation complete!                                    ║"
        echo "╚══════════════════════════════════════════════════════════════╝"
        echo ""
        "$BINARY_NAME" version
        echo ""
        info "Get started with: ${BINARY_NAME} init"
    else
        warn "Binary installed to ${INSTALL_DIR}/${binary}"
        warn "Make sure ${INSTALL_DIR} is in your PATH"
        echo ""
        echo "Add to PATH with:"
        echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
    fi
    echo ""
}

# Check for updates
check_update() {
    echo ""
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║              mcp-codewizard Updater                          ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""

    info "Checking current installation..."
    local current=""
    if command -v "$BINARY_NAME" &> /dev/null; then
        current=$("$BINARY_NAME" version 2>/dev/null | head -1 | awk '{print $2}' || echo "")
        echo "  → Installed version: ${current:-unknown}"
    else
        echo "  → Not installed"
    fi

    info "Fetching latest version from GitHub..."
    local latest=$(get_latest_version)
    echo "  → Latest version: ${latest}"
    echo ""

    if [ -z "$current" ]; then
        info "mcp-codewizard is not installed. Installing..."
        install
    elif [ "$current" = "$latest" ] || [ "v$current" = "$latest" ] || [ "$current" = "${latest#v}" ]; then
        echo "╔══════════════════════════════════════════════════════════════╗"
        echo "║  ✓ Already at latest version!                               ║"
        echo "╚══════════════════════════════════════════════════════════════╝"
        echo ""
        "$BINARY_NAME" version
        echo ""
    else
        info "Update available: ${current} → ${latest}"
        echo ""
        install
    fi
}

# Main
main() {
    case "${1:-install}" in
        install)
            install
            ;;
        update|upgrade)
            check_update
            ;;
        --help|-h)
            echo "mcp-codewizard installer"
            echo ""
            echo "Usage: $0 [command]"
            echo ""
            echo "Commands:"
            echo "  install   Install mcp-codewizard (default)"
            echo "  update    Check for updates and install if available"
            echo ""
            echo "Environment variables:"
            echo "  VERSION     Specific version to install (e.g., v0.1.1)"
            echo "  INSTALL_DIR Installation directory (default: /usr/local/bin)"
            ;;
        *)
            error "Unknown command: $1"
            ;;
    esac
}

main "$@"
