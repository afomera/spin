#!/bin/bash
set -e

# Get the directory of this script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Create the destination directory if it doesn't exist
mkdir -p /usr/local/bin

# Extract the binary from the app bundle using ditto to preserve code signing
ditto "$DIR/Spin.app/Contents/MacOS/spin" /usr/local/bin/spin

echo "Spin CLI installed successfully! Run 'spin' to get started."
