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
    local os=$(detect_os)
    local arch=$(detect_arch)
    local version="${VERSION:-$(get_latest_version)}"

    if [ -z "$version" ]; then
        error "Failed to determine version. Set VERSION env var or check network."
    fi

    info "Installing ${BINARY_NAME} ${version} for ${os}/${arch}..."

    # Determine file extension
    local ext="tar.gz"
    if [ "$os" = "windows" ]; then
        ext="zip"
    fi

    local filename="${BINARY_NAME}-${version}-${os}-${arch}.${ext}"
    local url="https://github.com/${REPO}/releases/download/${version}/${filename}"

    # Create temp directory
    local tmpdir=$(mktemp -d)
    trap "rm -rf $tmpdir" EXIT

    info "Downloading ${url}..."
    curl -fsSL "$url" -o "${tmpdir}/${filename}" || error "Download failed"

    # Extract
    info "Extracting..."
    cd "$tmpdir"
    if [ "$ext" = "tar.gz" ]; then
        tar -xzf "$filename"
    else
        unzip -q "$filename"
    fi

    # Install binary
    local binary="${BINARY_NAME}"
    if [ "$os" = "windows" ]; then
        binary="${BINARY_NAME}.exe"
    fi

    if [ ! -f "$binary" ]; then
        error "Binary not found in archive"
    fi

    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        mv "$binary" "${INSTALL_DIR}/${binary}"
        chmod +x "${INSTALL_DIR}/${binary}"
    else
        info "Requesting sudo for installation to ${INSTALL_DIR}..."
        sudo mv "$binary" "${INSTALL_DIR}/${binary}"
        sudo chmod +x "${INSTALL_DIR}/${binary}"
    fi

    # Verify installation
    if command -v "$BINARY_NAME" &> /dev/null; then
        local installed_version=$("$BINARY_NAME" version 2>/dev/null || echo "unknown")
        info "Successfully installed ${BINARY_NAME} ${installed_version} to ${INSTALL_DIR}"
    else
        warn "Installed to ${INSTALL_DIR}/${binary}"
        warn "Make sure ${INSTALL_DIR} is in your PATH"
    fi
}

# Check for updates
check_update() {
    local current=""
    if command -v "$BINARY_NAME" &> /dev/null; then
        current=$("$BINARY_NAME" version 2>/dev/null | head -1 || echo "")
    fi

    local latest=$(get_latest_version)

    if [ -z "$current" ]; then
        info "mcp-codewizard is not installed. Installing..."
        install
    elif [ "$current" = "$latest" ] || [ "v$current" = "$latest" ]; then
        info "Already at latest version: ${latest}"
    else
        info "Update available: ${current} -> ${latest}"
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
