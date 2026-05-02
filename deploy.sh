#!/bin/bash
# Deploys the chaincode to an already-running Fabric network.
# Run this after the network is up (step 3 in the README).
# Usage: ./deploy.sh
set -e

PROJECT_ROOT="$(cd "$(dirname "$0")" && pwd)"
CHAINCODE_DIR="$PROJECT_ROOT/chaincode/shipment"

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
  echo "ERROR: fabric-samples not found. Set FABRIC_SAMPLES_PATH."; exit 1
fi

TN="$FABRIC_SAMPLES_PATH/test-network"
export PATH="$FABRIC_SAMPLES_PATH/bin:$PATH"
export FABRIC_CFG_PATH="$FABRIC_SAMPLES_PATH/config"
export CORE_PEER_TLS_ENABLED=true

ORDERER_CA="$TN/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"
ORG1_CA="$TN/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt"
ORG2_CA="$TN/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt"
ORG1_MSP="$TN/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp"
ORG2_MSP="$TN/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp"

echo "[1/5] Building chaincode Docker image..."
docker rmi shipment-chaincode:latest 2>/dev/null || true
docker build -t shipment-chaincode:latest "$CHAINCODE_DIR" --quiet
echo "    Built: shipment-chaincode:latest"

echo "[2/5] Creating CCaaS package and installing on both peers..."
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT
echo '{"path":"","type":"ccaas","label":"shipment_1.0"}' > "$TMPDIR/metadata.json"
echo '{"address":"shipment-chaincode:7052","dial_timeout":"10s","tls_required":false}' > "$TMPDIR/connection.json"
cd "$TMPDIR"
tar cfz code.tar.gz connection.json
tar cfz shipment-ccaas.tar.gz metadata.json code.tar.gz

export CORE_PEER_LOCALMSPID=Org1MSP CORE_PEER_MSPCONFIGPATH="$ORG1_MSP" CORE_PEER_TLS_ROOTCERT_FILE="$ORG1_CA" CORE_PEER_ADDRESS=localhost:7051
peer lifecycle chaincode install shipment-ccaas.tar.gz

export CORE_PEER_LOCALMSPID=Org2MSP CORE_PEER_MSPCONFIGPATH="$ORG2_MSP" CORE_PEER_TLS_ROOTCERT_FILE="$ORG2_CA" CORE_PEER_ADDRESS=localhost:9051
peer lifecycle chaincode install shipment-ccaas.tar.gz

echo "[3/5] Getting package ID and starting chaincode container..."
export CORE_PEER_LOCALMSPID=Org1MSP CORE_PEER_MSPCONFIGPATH="$ORG1_MSP" CORE_PEER_TLS_ROOTCERT_FILE="$ORG1_CA" CORE_PEER_ADDRESS=localhost:7051
PKGID=$(peer lifecycle chaincode queryinstalled --output json | python3 -c "import json,sys; data=json.load(sys.stdin); [print(cc['package_id']) for cc in data.get('installed_chaincodes',[]) if cc['label']=='shipment_1.0']")

if [ -z "$PKGID" ]; then echo "ERROR: could not get package ID"; exit 1; fi
echo "    Package ID: $PKGID"

docker rm -f shipment-chaincode 2>/dev/null || true
docker run -d \
  --name shipment-chaincode \
  --network fabric_test \
  -e CORE_CHAINCODE_SERVER_ADDRESS=0.0.0.0:7052 \
  -e "CORE_CHAINCODE_ID=$PKGID" \
  shipment-chaincode:latest
sleep 3
echo "    Chaincode container running"

echo "[4/5] Approving and committing chaincode..."
export CORE_PEER_LOCALMSPID=Org1MSP CORE_PEER_MSPCONFIGPATH="$ORG1_MSP" CORE_PEER_TLS_ROOTCERT_FILE="$ORG1_CA" CORE_PEER_ADDRESS=localhost:7051
peer lifecycle chaincode approveformyorg \
  -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com \
  --tls --cafile "$ORDERER_CA" \
  --channelID shipchannel --name shipment --version 1.0 \
  --package-id "$PKGID" --sequence 1

export CORE_PEER_LOCALMSPID=Org2MSP CORE_PEER_MSPCONFIGPATH="$ORG2_MSP" CORE_PEER_TLS_ROOTCERT_FILE="$ORG2_CA" CORE_PEER_ADDRESS=localhost:9051
peer lifecycle chaincode approveformyorg \
  -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com \
  --tls --cafile "$ORDERER_CA" \
  --channelID shipchannel --name shipment --version 1.0 \
  --package-id "$PKGID" --sequence 1

export CORE_PEER_LOCALMSPID=Org1MSP CORE_PEER_MSPCONFIGPATH="$ORG1_MSP" CORE_PEER_TLS_ROOTCERT_FILE="$ORG1_CA" CORE_PEER_ADDRESS=localhost:7051
peer lifecycle chaincode commit \
  -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com \
  --tls --cafile "$ORDERER_CA" \
  --channelID shipchannel --name shipment --version 1.0 --sequence 1 \
  --peerAddresses localhost:7051 --tlsRootCertFiles "$ORG1_CA" \
  --peerAddresses localhost:9051 --tlsRootCertFiles "$ORG2_CA"

echo "[5/5] Initializing ledger..."
sleep 3
peer chaincode invoke \
  -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com \
  --tls --cafile "$ORDERER_CA" \
  -C shipchannel -n shipment \
  --peerAddresses localhost:7051 --tlsRootCertFiles "$ORG1_CA" \
  --peerAddresses localhost:9051 --tlsRootCertFiles "$ORG2_CA" \
  -c '{"function":"InitLedger","Args":[]}'

echo ""
echo "Chaincode deployed and ledger initialized."
echo "Next: run ./run-api.sh to start the REST API."
