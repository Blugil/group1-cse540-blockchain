#!/bin/bash
# Starts the REST API from /tmp (avoids iCloud Drive timeout issues).
# Run this after ./deploy.sh completes.
set -e

PROJECT_ROOT="$(cd "$(dirname "$0")" && pwd)"
NODE18="/Users/anushreebhure/.nvm/versions/node/v18.20.8/bin/node"
NPM18="/Users/anushreebhure/.nvm/versions/node/v18.20.8/bin/npm"

FABRIC_SAMPLES_PATH="${FABRIC_SAMPLES_PATH:-}"
for candidate in \
  "$HOME/fabric-install/fabric-samples" \
  "$HOME/fabric-samples"; do
  if [ -f "$candidate/test-network/network.sh" ]; then
    FABRIC_SAMPLES_PATH="$candidate"; break
  fi
done

if [ -z "$FABRIC_SAMPLES_PATH" ]; then
  echo "ERROR: fabric-samples not found. Set FABRIC_SAMPLES_PATH."; exit 1
fi

API_DIR="/tmp/shipapi"

echo "[1/3] Copying source files to /tmp/shipapi (avoids iCloud Drive)..."
mkdir -p "$API_DIR"
cp -r "$PROJECT_ROOT/client/src" "$API_DIR/"
cp -r "$PROJECT_ROOT/client/public" "$API_DIR/" 2>/dev/null || true
cp "$PROJECT_ROOT/client/package.json" "$API_DIR/"

if [ ! -d "$API_DIR/node_modules" ]; then
  echo "    Installing npm packages (first run only, ~60s)..."
  cd "$API_DIR"
  FABRIC_SAMPLES_PATH="$FABRIC_SAMPLES_PATH" "$NPM18" install --prefer-offline --no-audit --no-fund
fi

echo "[2/3] Importing wallet identities..."
FABRIC_SAMPLES_PATH="$FABRIC_SAMPLES_PATH" "$NODE18" "$API_DIR/src/importCryptoIdentity.js"

echo "[3/3] Starting API on http://localhost:3000 ..."
echo "      (Press Ctrl+C to stop)"
echo ""
pkill -f "node.*app.js" 2>/dev/null || true
sleep 1
cd "$API_DIR"
FABRIC_SAMPLES_PATH="$FABRIC_SAMPLES_PATH" "$NODE18" src/app.js
