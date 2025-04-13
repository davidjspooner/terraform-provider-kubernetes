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

# Set environment variables with defaults if not already set
DEFAULT_TF_VERSION="1.5.7"  # Last stable version of Terraform before the fork
TF_VERSION="${TF_VERSION:-$DEFAULT_TF_VERSION}"  # Default to latest stable if not set
ARCH="${ARCH:-$DEFAULT_ARCH}"  # Use the detected architecture if ARCH is not set
OS="${OS:-linux}"                  # Default to linux if OS is not set

# Check if the correct version of Terraform is already installed
if command -v terraform &> /dev/null; then
  INSTALLED_TF_VERSION=$(terraform version -json | jq -r '.terraform_version')
  if [ "$INSTALLED_TF_VERSION" == "$TF_VERSION" ]; then
    echo "Terraform version $TF_VERSION is already installed. Skipping installation."
    exit 0
  else
    echo "Removing incorrect Terraform version $INSTALLED_TF_VERSION..."
    rm -f /usr/local/bin/terraform
  fi
fi

# Example commands
echo "Installing Terraform version $TF_VERSION for $OS/$ARCH..."
curl -fsSL "https://releases.hashicorp.com/terraform/${TF_VERSION}/terraform_${TF_VERSION}_${OS}_${ARCH}.zip" -o terraform.zip
unzip terraform.zip
mv terraform /usr/local/bin/
rm terraform.zip

# After installation, verify the installed version of Terraform
INSTALLED_TF_VERSION=$(terraform version -json | jq -r '.terraform_version')
if [ "$INSTALLED_TF_VERSION" == "$TF_VERSION" ]; then
  echo "Terraform version $TF_VERSION has been successfully installed."
else
  echo "Error: Installed Terraform version $INSTALLED_TF_VERSION does not match the expected version $TF_VERSION."
  exit 1
fi

