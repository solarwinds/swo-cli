#!/bin/bash
# setup.sh for swo-cli devcontainer
set -e

echo "Setting up swo-cli development environment..."

# Ensure Go directories exist and have proper permissions
echo "   Setting up Go directory permissions..."
sudo mkdir -p "$GOCACHE" "$GOMODCACHE" "$GOPATH/bin"
sudo chown -R vscode:vscode "$GOCACHE" "$GOMODCACHE" "$GOPATH"

# Install any additional Go tools if needed
go install -a std

echo "Setup complete!"