#!/bin/bash
# Starts the full shipment tracking demo:
#   1. Spins up Hyperledger Fabric test-network (2 orgs, CouchDB)
#   2. Builds the chaincode Docker image on the host
#   3. Deploys chaincode via CCaaS (avoids Docker-in-Docker issues on macOS)
#   4. Imports crypto identities into the wallet
#   5. Starts the REST API + dashboard UI on http://localhost:3000

set -e

PROJECT_ROOT="$(cd "$(dirname "$0")" && pwd)"
CLIENT_DIR="$PROJECT_ROOT/client"
CHAINCODE_DIR="$PROJECT_ROOT/chaincode/shipment"

# ── Resolve a Node 18-compatible binary ──────────────────────
# fabric-network 2.2.x uses native gRPC that hangs on Node 20+
NODE_BIN="$(command -v node)"
NODE_MAJOR=$(node -e "process.stdout.write(String(process.versions.node.split('.')[0]))" 2>/dev/null || echo "0")
if [ "$NODE_MAJOR" -ge 20 ]; then
  # Try nvm
  NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
  if [ -s "$NVM_DIR/nvm.sh" ]; then
    # shellcheck source=/dev/null
    . "$NVM_DIR/nvm.sh"
    nvm install 18 --no-progress >/dev/null 2>&1 || true
    nvm use 18 >/dev/null 2>&1 || true
    NODE_BIN="$(command -v node)"
  else
    echo "WARNING: Node $(node --version) detected. fabric-network 2.2.x requires Node 18."
    echo "         Install Node 18 via nvm: nvm install 18 && nvm use 18"
    echo "         Continuing anyway — API may hang on startup."
  fi
fi
export NODE_BIN

# ── Locate fabric-samples ────────────────────────────────────
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

if [ -z "$FABRIC_SAMPLES_PATH" ]; then
  echo ""
  echo "ERROR: fabric-samples not found."
  echo "Install Hyperledger Fabric, then set:"
  echo "  export FABRIC_SAMPLES_PATH=/path/to/fabric-samples"
  echo "  ./start.sh"
  echo ""
  echo "Install guide: https://hyperledger-fabric.readthedocs.io/en/latest/install.html"
  exit 1
fi

TEST_NETWORK="$FABRIC_SAMPLES_PATH/test-network"
export PATH="$FABRIC_SAMPLES_PATH/bin:$PATH"
export FABRIC_CFG_PATH="$FABRIC_SAMPLES_PATH/config"
export FABRIC_SAMPLES_PATH

# TLS cert shortcuts used throughout
ORDERER_CA="$TEST_NETWORK/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"
ORG1_PEER_CA="$TEST_NETWORK/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt"
ORG2_PEER_CA="$TEST_NETWORK/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt"
ORG1_MSP="$TEST_NETWORK/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp"
ORG2_MSP="$TEST_NETWORK/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp"

echo ""
echo "======================================================"
echo "  Blockchain Shipment Tracking — Group 1, CSE 540"
echo "======================================================"
echo "  fabric-samples : $FABRIC_SAMPLES_PATH"
echo "  Project root   : $PROJECT_ROOT"
echo "======================================================"
echo ""

# ── Prerequisites ────────────────────────────────────────────
echo "[1/7] Checking prerequisites..."
command -v docker >/dev/null 2>&1  || { echo "ERROR: Docker not found. Install Docker Desktop."; exit 1; }
command -v node   >/dev/null 2>&1  || { echo "ERROR: Node.js not found. Install Node.js 18+."; exit 1; }
command -v peer   >/dev/null 2>&1  || { echo "ERROR: Fabric peer binary not in PATH. Check FABRIC_SAMPLES_PATH."; exit 1; }
docker info >/dev/null 2>&1        || { echo "ERROR: Docker daemon is not running. Start Docker Desktop first."; exit 1; }
echo "    OK"

# ── Start IPFS node ───────────────────────────────────────────
echo "[2/7] Starting IPFS node..."
docker rm -f ipfs-node 2>/dev/null || true
docker-compose -f "$PROJECT_ROOT/network/docker/docker-compose-ipfs.yaml" up -d
# Wait for IPFS daemon to be ready
IPFS_READY=false
for i in $(seq 1 15); do
  if curl -s http://localhost:5001/api/v0/version >/dev/null 2>&1; then
    IPFS_READY=true
    break
  fi
  sleep 2
done
if [ "$IPFS_READY" = "false" ]; then
  echo "    WARNING: IPFS node did not become ready in time. Document upload will not work."
else
  echo "    IPFS ready at http://localhost:5001 (gateway: http://localhost:8080)"
fi

# ── Start Fabric network ─────────────────────────────────────
echo "[3/7] Starting Fabric test-network..."
cd "$TEST_NETWORK"
./network.sh down 2>/dev/null || true
./network.sh up createChannel -c shipchannel -s couchdb
echo "    Network up"

# ── Build chaincode Docker image on host ─────────────────────
echo "[4/7] Building chaincode Docker image..."
docker rmi shipment-chaincode:latest 2>/dev/null || true
docker build -t shipment-chaincode:latest "$CHAINCODE_DIR" --quiet
echo "    Image built: shipment-chaincode:latest"

# ── Create CCaaS package ─────────────────────────────────────
echo "[5/7] Deploying chaincode via CCaaS..."

TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

# metadata tells the peer this is an external chaincode
echo '{"path":"","type":"ccaas","label":"shipment_1.0"}' > "$TMPDIR/metadata.json"
# connection tells the peer where the chaincode process is listening
echo '{"address":"shipment-chaincode:7052","dial_timeout":"10s","tls_required":false}' > "$TMPDIR/connection.json"

cd "$TMPDIR"
tar cfz code.tar.gz connection.json
tar cfz shipment-ccaas.tar.gz metadata.json code.tar.gz

# ── Install on both peers ─────────────────────────────────────
export CORE_PEER_TLS_ENABLED=true

# Org1
export CORE_PEER_LOCALMSPID=Org1MSP
export CORE_PEER_MSPCONFIGPATH="$ORG1_MSP"
export CORE_PEER_TLS_ROOTCERT_FILE="$ORG1_PEER_CA"
export CORE_PEER_ADDRESS=localhost:7051
peer lifecycle chaincode install shipment-ccaas.tar.gz

# Org2
export CORE_PEER_LOCALMSPID=Org2MSP
export CORE_PEER_MSPCONFIGPATH="$ORG2_MSP"
export CORE_PEER_TLS_ROOTCERT_FILE="$ORG2_PEER_CA"
export CORE_PEER_ADDRESS=localhost:9051
peer lifecycle chaincode install shipment-ccaas.tar.gz

# ── Get package ID ────────────────────────────────────────────
export CORE_PEER_LOCALMSPID=Org1MSP
export CORE_PEER_MSPCONFIGPATH="$ORG1_MSP"
export CORE_PEER_TLS_ROOTCERT_FILE="$ORG1_PEER_CA"
export CORE_PEER_ADDRESS=localhost:7051
PACKAGE_ID=$(peer lifecycle chaincode queryinstalled --output json | python3 -c "
import json,sys
data=json.load(sys.stdin)
for cc in data.get('installed_chaincodes',[]):
    if cc['label']=='shipment_1.0':
        print(cc['package_id'])
        break
")

if [ -z "$PACKAGE_ID" ]; then
  echo "ERROR: Could not determine chaincode package ID"
  exit 1
fi
echo "    Package ID: $PACKAGE_ID"

# ── Start chaincode container ─────────────────────────────────
docker rm -f shipment-chaincode 2>/dev/null || true
docker run -d \
  --name shipment-chaincode \
  --network fabric_test \
  -e CORE_CHAINCODE_SERVER_ADDRESS=0.0.0.0:7052 \
  -e CORE_CHAINCODE_ID="$PACKAGE_ID" \
  shipment-chaincode:latest
echo "    Chaincode container started"
sleep 3

# ── Approve for both orgs ─────────────────────────────────────
# Org1
export CORE_PEER_LOCALMSPID=Org1MSP
export CORE_PEER_MSPCONFIGPATH="$ORG1_MSP"
export CORE_PEER_TLS_ROOTCERT_FILE="$ORG1_PEER_CA"
export CORE_PEER_ADDRESS=localhost:7051
peer lifecycle chaincode approveformyorg \
  -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com \
  --tls --cafile "$ORDERER_CA" \
  --channelID shipchannel --name shipment --version 1.0 \
  --package-id "$PACKAGE_ID" --sequence 1

# Org2
export CORE_PEER_LOCALMSPID=Org2MSP
export CORE_PEER_MSPCONFIGPATH="$ORG2_MSP"
export CORE_PEER_TLS_ROOTCERT_FILE="$ORG2_PEER_CA"
export CORE_PEER_ADDRESS=localhost:9051
peer lifecycle chaincode approveformyorg \
  -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com \
  --tls --cafile "$ORDERER_CA" \
  --channelID shipchannel --name shipment --version 1.0 \
  --package-id "$PACKAGE_ID" --sequence 1

# ── Commit chaincode definition ───────────────────────────────
export CORE_PEER_LOCALMSPID=Org1MSP
export CORE_PEER_MSPCONFIGPATH="$ORG1_MSP"
export CORE_PEER_TLS_ROOTCERT_FILE="$ORG1_PEER_CA"
export CORE_PEER_ADDRESS=localhost:7051
peer lifecycle chaincode commit \
  -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com \
  --tls --cafile "$ORDERER_CA" \
  --channelID shipchannel --name shipment --version 1.0 --sequence 1 \
  --peerAddresses localhost:7051 --tlsRootCertFiles "$ORG1_PEER_CA" \
  --peerAddresses localhost:9051 --tlsRootCertFiles "$ORG2_PEER_CA"
echo "    Chaincode committed"

# ── Invoke InitLedger ─────────────────────────────────────────
sleep 3
peer chaincode invoke \
  -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com \
  --tls --cafile "$ORDERER_CA" \
  -C shipchannel -n shipment \
  --peerAddresses localhost:7051 --tlsRootCertFiles "$ORG1_PEER_CA" \
  --peerAddresses localhost:9051 --tlsRootCertFiles "$ORG2_PEER_CA" \
  -c '{"function":"InitLedger","Args":[]}'
echo "    Ledger initialized"

# ── Set up wallet ─────────────────────────────────────────────
echo "[6/7] Installing dependencies and setting up wallet..."
cd "$CLIENT_DIR"
npm install --silent
"$NODE_BIN" src/importCryptoIdentity.js

# ── Start REST API ────────────────────────────────────────────
echo "[7/7] Starting REST API and dashboard..."
echo ""
echo "======================================================"
echo "  Dashboard ready at: http://localhost:3000"
echo "  Press Ctrl+C to stop the API."
echo "  To stop everything: ./stop.sh"
echo "======================================================"
echo ""
FABRIC_SAMPLES_PATH="$FABRIC_SAMPLES_PATH" "$NODE_BIN" src/app.js
