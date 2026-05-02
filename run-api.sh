#!/bin/bash
# Starts the REST API from /tmp (avoids iCloud Drive timeout issues on macOS).
# Run this after ./deploy.sh completes.
set -e

PROJECT_ROOT="$(cd "$(dirname "$0")" && pwd)"

# ── Resolve Node 18 ───────────────────────────────────────────
# fabric-network 2.2.x requires Node 18; newer versions hang on startup.
NODE_BIN="$(command -v node)"
NODE_MAJOR=$(node -e "process.stdout.write(String(process.versions.node.split('.')[0]))" 2>/dev/null || echo "0")
if [ "$NODE_MAJOR" -ge 20 ]; then
  NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
  if [ -s "$NVM_DIR/nvm.sh" ]; then
    . "$NVM_DIR/nvm.sh"
    nvm install 18 --no-progress >/dev/null 2>&1 || true
    nvm use 18 >/dev/null 2>&1 || true
    NODE_BIN="$(command -v node)"
  else
    echo "WARNING: Node $(node --version) detected. fabric-network 2.2.x requires Node 18."
    echo "         Install Node 18 via nvm: nvm install 18 && nvm use 18"
  fi
fi
NPM_BIN="$(command -v npm)"

# ── Locate fabric-samples ─────────────────────────────────────
FABRIC_SAMPLES_PATH="${FABRIC_SAMPLES_PATH:-}"
for candidate in \
  "$HOME/fabric-install/fabric-samples" \
  "$HOME/fabric-samples" \
  "$HOME/go/src/github.com/hyperledger/fabric-samples" \
  "/usr/local/fabric-samples" \
  "/opt/fabric-samples"; do
  if [ -f "$candidate/test-network/network.sh" ]; then
    FABRIC_SAMPLES_PATH="$candidate"; break
  fi
done

if [ -z "$FABRIC_SAMPLES_PATH" ]; then
  echo "ERROR: fabric-samples not found. Set FABRIC_SAMPLES_PATH and retry."; exit 1
fi

API_DIR="/tmp/shipapi"

echo "[1/3] Copying source files to /tmp/shipapi..."
mkdir -p "$API_DIR"
cp -r "$PROJECT_ROOT/client/src" "$API_DIR/"
cp -r "$PROJECT_ROOT/client/public" "$API_DIR/" 2>/dev/null || true
cp "$PROJECT_ROOT/client/package.json" "$API_DIR/"

if [ ! -d "$API_DIR/node_modules" ]; then
  echo "    Installing npm packages (first run only, ~60s)..."
  cd "$API_DIR"
  FABRIC_SAMPLES_PATH="$FABRIC_SAMPLES_PATH" "$NPM_BIN" install --prefer-offline --no-audit --no-fund
fi

echo "[2/3] Importing wallet identities..."
FABRIC_SAMPLES_PATH="$FABRIC_SAMPLES_PATH" "$NODE_BIN" "$API_DIR/src/importCryptoIdentity.js"

echo "[3/3] Starting API on http://localhost:3000 ..."
echo "      (Press Ctrl+C to stop)"
echo ""
pkill -f "node.*app.js" 2>/dev/null || true
sleep 1
cd "$API_DIR"
FABRIC_SAMPLES_PATH="$FABRIC_SAMPLES_PATH" "$NODE_BIN" src/app.js
