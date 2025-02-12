#!/bin/bash
set -e

# Get the directory of this script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Extract the binary from the app bundle
cp "$DIR/Spin.app/Contents/MacOS/spin" /usr/local/bin/spin

# Make it executable
chmod +x /usr/local/bin/spin

echo "Spin CLI installed successfully! Run 'spin' to get started."
