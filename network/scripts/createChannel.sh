#!/bin/bash
#
# Create Channel Script
# Creates the application channel and joins both peers to it.

CHANNEL_NAME=${1:-"shipchannel"}
DELAY=${2:-3}
MAX_RETRY=${3:-5}
VERBOSE=${4:-false}

FABRIC_CFG_PATH=${PWD}/configtx

# import utility functions
. scripts/envVar.sh

# ============================================================
# Create the channel genesis block
# ============================================================
createChannelGenesisBlock() {
  echo "Generating channel genesis block '${CHANNEL_NAME}.block'..."
  set -x
  configtxgen -profile TwoOrgsOrdererGenesis -outputBlock ./channel-artifacts/${CHANNEL_NAME}.block -channelID $CHANNEL_NAME
  res=$?
  { set +x; } 2>/dev/null
  if [ $res -ne 0 ]; then
    echo "Failed to generate channel genesis block"
    exit 1
  fi
}

# ============================================================
# Create channel on orderer using osnadmin
# ============================================================
createChannel() {
  echo "Creating channel ${CHANNEL_NAME}..."

  local rc=1
  local COUNTER=1
  while [ $rc -ne 0 -a $COUNTER -lt $MAX_RETRY ]; do
    sleep $DELAY
    set -x
    osnadmin channel join --channelID $CHANNEL_NAME \
      --config-block ./channel-artifacts/${CHANNEL_NAME}.block \
      -o localhost:7053 \
      --ca-file "$ORDERER_CA" \
      --client-cert "$ORDERER_ADMIN_TLS_SIGN_CERT" \
      --client-key "$ORDERER_ADMIN_TLS_PRIVATE_KEY" >&log.txt
    res=$?
    { set +x; } 2>/dev/null
    rc=$res
    COUNTER=$(expr $COUNTER + 1)
  done

  cat log.txt

  if [ $rc -ne 0 ]; then
    echo "Failed to create channel after $MAX_RETRY attempts"
    exit 1
  fi
}

# ============================================================
# Join a peer to the channel
# ============================================================
joinChannel() {
  ORG=$1
  setGlobals $ORG

  local rc=1
  local COUNTER=1
  while [ $rc -ne 0 -a $COUNTER -lt $MAX_RETRY ]; do
    sleep $DELAY
    set -x
    peer channel join -b ./channel-artifacts/${CHANNEL_NAME}.block >&log.txt
    res=$?
    { set +x; } 2>/dev/null
    rc=$res
    COUNTER=$(expr $COUNTER + 1)
  done

  cat log.txt

  if [ $rc -ne 0 ]; then
    echo "Peer ${ORG} failed to join channel after $MAX_RETRY attempts"
    exit 1
  fi
}

# ============================================================
# Set anchor peers
# ============================================================
setAnchorPeer() {
  ORG=$1
  echo "Setting anchor peer for ${ORG}..."
  docker exec cli ./scripts/setAnchorPeer.sh $ORG $CHANNEL_NAME
}

# ============================================================
# Main execution
# ============================================================

# Create channel artifacts directory
mkdir -p channel-artifacts

# Generate genesis block
createChannelGenesisBlock

# Create channel
ORDERER_CA="${PWD}/organizations/ordererOrganizations/shipment.com/tlsca/tlsca.shipment.com-cert.pem"
ORDERER_ADMIN_TLS_SIGN_CERT="${PWD}/organizations/ordererOrganizations/shipment.com/orderers/orderer.shipment.com/tls/server.crt"
ORDERER_ADMIN_TLS_PRIVATE_KEY="${PWD}/organizations/ordererOrganizations/shipment.com/orderers/orderer.shipment.com/tls/server.key"

createChannel

# Join ManufacturerOrg peer to channel
echo "Joining ManufacturerOrg peer to channel..."
joinChannel 1

# Join TransporterOrg peer to channel
echo "Joining TransporterOrg peer to channel..."
joinChannel 2

echo "========== Channel '${CHANNEL_NAME}' created and peers joined =========="
