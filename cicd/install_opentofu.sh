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

DEFAULT_TOFU_VERSION="1.1.0"  # Latest stable version of OpenTofu

# Set environment variables with defaults if not already set
TOFU_VERSION="${TOFU_VERSION:-$DEFAULT_TOFU_VERSION}"  # Default to latest stable if not set
ARCH="${ARCH:-$DEFAULT_ARCH}"  # Use the detected architecture if ARCH is not set
OS="${OS:-linux}"                  # Default to linux if OS is not set

# Check if the correct version of OpenTofu is already installed
if command -v opentofu &> /dev/null; then
  INSTALLED_TOFU_VERSION=$(opentofu version | awk '{print $2}')
  if [ "$INSTALLED_TOFU_VERSION" == "$TOFU_VERSION" ]; then
    echo "OpenTofu version $TOFU_VERSION is already installed. Skipping installation."
    exit 0
  else
    echo "Removing incorrect OpenTofu version $INSTALLED_TOFU_VERSION..."
    rm -f /usr/local/bin/opentofu
  fi
fi

# Example commands
echo "Installing OpenTofu version $TOFU_VERSION for $OS/$ARCH..."
curl -fsSL "https://releases.opentofu.org/${TOFU_VERSION}/opentofu_${TOFU_VERSION}_${OS}_${ARCH}.zip" -o opentofu.zip
unzip opentofu.zip
mv opentofu /usr/local/bin/
rm opentofu.zip

# After installation, verify the installed version of OpenTofu
INSTALLED_TOFU_VERSION=$(opentofu version | awk '{print $2}')
if [ "$INSTALLED_TOFU_VERSION" == "$TOFU_VERSION" ]; then
  echo "OpenTofu version $TOFU_VERSION has been successfully installed."
else
  echo "Error: Installed OpenTofu version $INSTALLED_TOFU_VERSION does not match the expected version $TOFU_VERSION."
  exit 1
fi

