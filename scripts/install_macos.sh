#!/bin/bash
set -e

# Get the directory of this script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Create the destination directory if it doesn't exist
mkdir -p /usr/local/bin

# Create a symlink to the binary inside the app bundle
ln -sf "$DIR/Spin.app/Contents/MacOS/spin" /usr/local/bin/spin

echo "Spin CLI installed successfully! Run 'spin' to get started."
