#!/bin/bash
set -e # Exit on error
set -x # Echo commands

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

DEFAULT_KIND_VERSION="0.27.0"  # Updated to the latest stable version of KIND

# Set environment variables with defaults if not already set
KIND_VERSION="${KIND_VERSION:-$DEFAULT_KIND_VERSION}"  # Default to latest stable if not set
ARCH="${ARCH:-$DEFAULT_ARCH}"  # Use the detected architecture if ARCH is not set
OS="${OS:-linux}"                  # Default to linux if OS is not set

# Check if the correct version of kind is already installed
if command -v kind &> /dev/null; then
  INSTALLED_KIND_VERSION=$(kind version | awk '{print $3}' | sed 's/v//')
  if [ "$INSTALLED_KIND_VERSION" == "$KIND_VERSION" ]; then
    echo "kind version $KIND_VERSION is already installed. Skipping installation."
    exit 0
  else
    echo "Removing incorrect kind version $INSTALLED_KIND_VERSION..."
    rm -f /usr/local/bin/kind
  fi
fi

# Download and install kind
KIND_URL="https://github.com/kubernetes-sigs/kind/releases/download/v${KIND_VERSION}/kind-${OS}-${ARCH}"
echo "Downloading kind from $KIND_URL..."
curl -fsSL "$KIND_URL" -o kind
if [ $? -ne 0 ]; then
  echo "Failed to download kind. Exiting."
  exit 1
fi

chmod +x kind
mv kind /usr/local/bin/

# After installation, verify the installed version of kind
INSTALLED_KIND_VERSION=$(kind version | awk '{print $2}' | sed 's/v//')
if [ "$INSTALLED_KIND_VERSION" == "$KIND_VERSION" ]; then
  echo "kind version $KIND_VERSION has been successfully installed."
else
  echo "Error: Installed kind version $INSTALLED_KIND_VERSION does not match the expected version $KIND_VERSION."
  exit 1
fi

