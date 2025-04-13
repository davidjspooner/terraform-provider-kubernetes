#!/bin/bash
set -e # echo statements
set -x # exit on error

# Check if the script is running as root
if [ "$EUID" -ne 0 ]; then
  echo "This script must be run as root. Exiting."
  exit 1
fi

# Add DEBIAN_FRONTEND=noninteractive to suppress prompts during package installation
export DEBIAN_FRONTEND=noninteractive

# Update and upgrade apt packages
echo "Updating and upgrading apt packages..."
apt update && apt -y upgrade

# Install required utilities
echo "Installing curl, tar, and unzip..."
apt -y install curl tar unzip

echo "Utilities installation complete."