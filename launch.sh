#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"

echo "Pulling latest changes..."
git pull

echo "Building..."
make mo build

echo "Launching Dark Station..."
exec ./darkstation "$@"
