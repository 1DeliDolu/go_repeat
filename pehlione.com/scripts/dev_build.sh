#!/usr/bin/env bash
set -euo pipefail

# Development build script: templ generate + tailwind build + go build
# This ensures that templ changes trigger Tailwind CSS rebuild

echo "📦 Generating templ..."
templ generate ./...

echo "🎨 Building Tailwind CSS..."
npm run build:css

# Determine output binary name based on OS
GOOS="$(go env GOOS)"
OUT="./tmp/pehlione-web"
if [[ "$GOOS" == "windows" ]]; then
  OUT="./tmp/pehlione-web.exe"
fi

echo "🔨 Building Go binary..."
go build -o "$OUT" ./cmd/web

echo "✅ Build complete"
