# Test Chaincode via Peer CLI
# ============================
# This script demonstrates the full shipment lifecycle by invoking
# chaincode functions directly via the Fabric peer CLI.
# Prerequisites: Network must be running and chaincode must be deployed.
# CSE 540 – Spring B 2026 | Group 1

set -e

# Source environment variables
. scripts/envVar.sh

CHANNEL_NAME="shipchannel"
CC_NAME="shipment"

ORDERER_CA="${PWD}/organizations/ordererOrganizations/shipment.com/orderers/orderer.shipment.com/msp/tlscacerts/tlsca.shipment.com-cert.pem"

PEER_MANUFACTURER="--peerAddresses localhost:7051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/manufacturer.shipment.com/peers/peer0.manufacturer.shipment.com/tls/ca.crt"
PEER_TRANSPORTER="--peerAddresses localhost:9051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/transporter.shipment.com/peers/peer0.transporter.shipment.com/tls/ca.crt"

ORDERER_ARGS="-o localhost:7050 --ordererTLSHostnameOverride orderer.shipment.com --tls --cafile ${ORDERER_CA}"

C_RESET='\033[0m'
C_GREEN='\033[0;32m'
C_BLUE='\033[0;34m'
C_YELLOW='\033[1;33m'

println() { echo -e "$1"; }
header() { println "\n${C_BLUE}============================================================${C_RESET}"; println "${C_GREEN}  $1${C_RESET}"; println "${C_BLUE}============================================================${C_RESET}"; }

# ============================================================
# Test 1: Query the sample shipment from InitLedger
# ============================================================
header "Test 1: Query Sample Shipment (SHIP-001)"
setGlobals 1
peer chaincode query -C $CHANNEL_NAME -n $CC_NAME -c '{"function":"GetShipment","Args":["SHIP-001"]}' | python3 -m json.tool
sleep 2

# ============================================================
# Test 2: Create a new shipment as Manufacturer
# ============================================================
header "Test 2: Create Shipment (SHIP-CLI-001)"
setGlobals 1
peer chaincode invoke $ORDERER_ARGS -C $CHANNEL_NAME -n $CC_NAME $PEER_MANUFACTURER $PEER_TRANSPORTER \
  -c '{"function":"CreateShipment","Args":["SHIP-CLI-001","Factory-Phoenix-AZ","Store-Los-Angeles-CA","[\"ManufacturerMSP\",\"TransporterMSP\",\"WarehouseMSP\",\"RetailerMSP\",\"RecipientMSP\"]","weight:30kg,volume:1.5cbm,count:200"]}'
sleep 3

# ============================================================
# Test 3: Query the new shipment
# ============================================================
header "Test 3: Query New Shipment (SHIP-CLI-001)"
setGlobals 1
peer chaincode query -C $CHANNEL_NAME -n $CC_NAME -c '{"function":"GetShipment","Args":["SHIP-CLI-001"]}' | python3 -m json.tool
sleep 2

# ============================================================
# Test 4: Update shipment status to InTransit
# ============================================================
header "Test 4: Update Status to InTransit"
setGlobals 1
peer chaincode invoke $ORDERER_ARGS -C $CHANNEL_NAME -n $CC_NAME $PEER_MANUFACTURER $PEER_TRANSPORTER \
  -c '{"function":"UpdateShipmentStatus","Args":["SHIP-CLI-001","InTransit","Loading-Dock-Phoenix","Shipment loaded onto truck"]}'
sleep 3

# ============================================================
# Test 5: Transfer custody from Manufacturer to Transporter
# ============================================================
header "Test 5: Transfer Custody to TransporterMSP"
setGlobals 1
peer chaincode invoke $ORDERER_ARGS -C $CHANNEL_NAME -n $CC_NAME $PEER_MANUFACTURER $PEER_TRANSPORTER \
  -c '{"function":"TransferCustody","Args":["SHIP-CLI-001","TransporterMSP"]}'
sleep 3

# ============================================================
# Test 6: Query shipment after transfer (as Transporter)
# ============================================================
header "Test 6: Query Shipment After Transfer (as TransporterMSP)"
setGlobals 2
peer chaincode query -C $CHANNEL_NAME -n $CC_NAME -c '{"function":"GetShipment","Args":["SHIP-CLI-001"]}' | python3 -m json.tool
sleep 2

# ============================================================
# Test 7: Verify shipment data integrity
# ============================================================
header "Test 7: Verify Shipment Data Integrity"
setGlobals 1
peer chaincode query -C $CHANNEL_NAME -n $CC_NAME -c '{"function":"VerifyShipment","Args":["SHIP-CLI-001","weight:30kg,volume:1.5cbm,count:200"]}'
sleep 2

# ============================================================
# Test 8: Get shipment event history
# ============================================================
header "Test 8: Get Shipment History"
setGlobals 1
peer chaincode query -C $CHANNEL_NAME -n $CC_NAME -c '{"function":"GetShipmentHistory","Args":["SHIP-CLI-001"]}' | python3 -m json.tool
sleep 2

# ============================================================
# Test 9: Authorize a new participant
# ============================================================
header "Test 9: Authorize New Participant (NewPartnerMSP)"
setGlobals 2
peer chaincode invoke $ORDERER_ARGS -C $CHANNEL_NAME -n $CC_NAME $PEER_MANUFACTURER $PEER_TRANSPORTER \
  -c '{"function":"AuthorizeParticipant","Args":["SHIP-CLI-001","NewPartnerMSP"]}'
sleep 3

# ============================================================
# Test 10: Revoke a participant
# ============================================================
header "Test 10: Revoke Participant (NewPartnerMSP)"
setGlobals 2
peer chaincode invoke $ORDERER_ARGS -C $CHANNEL_NAME -n $CC_NAME $PEER_MANUFACTURER $PEER_TRANSPORTER \
  -c '{"function":"RevokeParticipant","Args":["SHIP-CLI-001","NewPartnerMSP"]}'
sleep 3

# ============================================================
# Test 11: Get all shipments
# ============================================================
header "Test 11: Get All Shipments"
setGlobals 1
peer chaincode query -C $CHANNEL_NAME -n $CC_NAME -c '{"function":"GetAllShipments","Args":[]}' | python3 -m json.tool

# ============================================================
# Test 12: Final shipment state after full workflow
# ============================================================
header "Test 12: Final State of SHIP-CLI-001"
setGlobals 1
peer chaincode query -C $CHANNEL_NAME -n $CC_NAME -c '{"function":"GetShipment","Args":["SHIP-CLI-001"]}' | python3 -m json.tool

println "\n${C_GREEN}========== All Tests Passed! ==========${C_RESET}\n"
