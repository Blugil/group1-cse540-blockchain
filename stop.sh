#!/bin/bash
# Tears down the Fabric network and stops the API server.

FABRIC_SAMPLES_PATH="${FABRIC_SAMPLES_PATH:-}"

if [ -z "$FABRIC_SAMPLES_PATH" ]; then
  for candidate in \
    "$HOME/fabric-install/fabric-samples" \
    "$HOME/fabric-samples" \
    "$HOME/go/src/github.com/hyperledger/fabric-samples" \
    "/usr/local/fabric-samples" \
    "/opt/fabric-samples"; do
    if [ -f "$candidate/test-network/network.sh" ]; then
      FABRIC_SAMPLES_PATH="$candidate"
      break
    fi
  done
fi

# Stop API server if running
if lsof -ti :3000 >/dev/null 2>&1; then
  echo "Stopping API server on port 3000..."
  kill "$(lsof -ti :3000)" 2>/dev/null || true
fi

# Stop IPFS node
if docker ps -q -f name=ipfs-node | grep -q .; then
  echo "Stopping IPFS node..."
  docker rm -f ipfs-node 2>/dev/null || true
fi

# Tear down Fabric network
if [ -n "$FABRIC_SAMPLES_PATH" ] && [ -f "$FABRIC_SAMPLES_PATH/test-network/network.sh" ]; then
  echo "Tearing down Fabric network..."
  cd "$FABRIC_SAMPLES_PATH/test-network"
  ./network.sh down
fi

echo "Done. To restart: ./start.sh"
