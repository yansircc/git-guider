#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "==> Building frontend..."
cd "$ROOT/web"
npm install --silent
npx vite build

echo "==> Building Go binary..."
cd "$ROOT"
go build -o git-guider ./cmd/server/

echo "==> Done: ./git-guider"
ls -lh git-guider
