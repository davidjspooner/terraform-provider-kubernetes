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

DEFAULT_KUBECTL_VERSION="1.32.3"  # Latest stable version of kubectl

# Set environment variables with defaults if not already set
KUBECTL_VERSION="${KUBECTL_VERSION:-$DEFAULT_KUBECTL_VERSION}"  # Default to latest stable if not set
ARCH="${ARCH:-$DEFAULT_ARCH}"  # Use the detected architecture if ARCH is not set
OS="${OS:-linux}"                  # Default to linux if OS is not set

# Check if the correct version of kubectl is already installed
if command -v kubectl &> /dev/null; then
  INSTALLED_KUBECTL_VERSION=$(kubectl version --client --output=json | jq -r '.clientVersion.gitVersion' | sed 's/v//')
  if [ "$INSTALLED_KUBECTL_VERSION" == "$KUBECTL_VERSION" ]; then
    echo "kubectl version $KUBECTL_VERSION is already installed. Skipping installation."
    exit 0
  else
    echo "Removing incorrect kubectl version $INSTALLED_KUBECTL_VERSION..."
    rm -f /usr/local/bin/kubectl
  fi
fi


# Example commands

echo "Installing kubectl version $KUBECTL_VERSION for $OS/$ARCH..."
curl -fsSL "https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/${OS}/${ARCH}/kubectl" -o kubectl
chmod +x kubectl
mv kubectl /usr/local/bin/

# After installation, verify the installed version of kubectl
INSTALLED_KUBECTL_VERSION=$(kubectl version --client --output=json | jq -r '.clientVersion.gitVersion' | sed 's/v//')
if [ "$INSTALLED_KUBECTL_VERSION" == "$KUBECTL_VERSION" ]; then
  echo "kubectl version $KUBECTL_VERSION has been successfully installed."
else
  echo "Error: Installed kubectl version $INSTALLED_KUBECTL_VERSION does not match the expected version $KUBECTL_VERSION."
  exit 1
fi

