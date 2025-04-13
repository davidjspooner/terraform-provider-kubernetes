#!/bin/bash
set -e # echo statements
set -x # exit on error

# Check if the script is running as root
if [ "$EUID" -ne 0 ]; then
  echo "This script must be run as root. Exiting."
  exit 1
fi

DEFAULT_ARCH=$(uname -m)
if [[ "$DEFAULT_ARCH" == "aarch64" ]]; then
  DEFAULT_ARCH="arm64"
elif [[ "$DEFAULT_ARCH" == "x86_64" ]]; then
  DEFAULT_ARCH="amd64"
else
  echo "Unsupported architecture: $DEFAULT_ARCH"
  exit 1
fi

DEFAULT_GO_VERSION="1.24.2"  # Latest stable version of Go

# Set environment variables with defaults if not already set
GO_VERSION="${GO_VERSION:-$DEFAULT_GO_VERSION}"  # Default to latest stable if not set
ARCH="${ARCH:-$DEFAULT_ARCH}"  # Use the detected architecture if ARCH is not set
OS="${OS:-linux}"                  # Default to linux if OS is not set

# Check if the correct version of Go is already installed
if command -v go &> /dev/null; then
  INSTALLED_GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
  if [ "$INSTALLED_GO_VERSION" == "$GO_VERSION" ]; then
    echo "Go version $GO_VERSION is already installed. Skipping installation."
    exit 0
  else
    echo "Removing incorrect Go version $INSTALLED_GO_VERSION..."
    rm -rf /usr/local/go
  fi
fi

# Update the download URL to use the correct domain for Go downloads
echo "Installing Go version $GO_VERSION for $OS/$ARCH..."

curl -fsSL "https://go.dev/dl/go${GO_VERSION}.${OS}-${ARCH}.tar.gz" -o go.tar.gz
tar -C /usr/local -xzf go.tar.gz
rm go.tar.gz

# Ensure /usr/local/bin/go symlink exists only if not already present
if [ ! -f /usr/local/bin/go ]; then
  ln -sf /usr/local/go/bin/go /usr/local/bin/go
fi

# After installation, verify the installed version of Go
INSTALLED_GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
if [ "$INSTALLED_GO_VERSION" == "$GO_VERSION" ]; then
  echo "Go version $GO_VERSION has been successfully installed."
else
  echo "Error: Installed Go version $INSTALLED_GO_VERSION does not match the expected version $GO_VERSION."
  exit 1
fi
