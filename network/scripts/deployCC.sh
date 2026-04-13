
CHANNEL_NAME=${1:-"shipchannel"}
CC_NAME=${2:-"shipment"}
CC_SRC_PATH=${3:-"../chaincode/shipment"}
CC_LANGUAGE=${4:-"go"}
CC_VERSION=${5:-"1.0"}
CC_SEQUENCE=${6:-1}
CC_INIT_FCN=${7:-"InitLedger"}
CC_END_POLICY=${8:-""}
CC_COLL_CONFIG=${9:-""}
DELAY=${10:-3}
MAX_RETRY=${11:-5}
VERBOSE=${12:-false}

FABRIC_CFG_PATH=${PWD}/configtx

. scripts/envVar.sh

println() { echo -e "$1"; }

packageChaincode() {
  println "Packaging chaincode '${CC_NAME}'..."

  set -x
  peer lifecycle chaincode package ${CC_NAME}.tar.gz \
    --path ${CC_SRC_PATH} \
    --lang ${CC_LANGUAGE} \
    --label ${CC_NAME}_${CC_VERSION} >&log.txt
  res=$?
  { set +x; } 2>/dev/null
  cat log.txt

  if [ $res -ne 0 ]; then
    println "ERROR: Failed to package chaincode"
    exit 1
  fi

  println "Chaincode packaged: ${CC_NAME}.tar.gz"
}

installChaincode() {
  ORG=$1
  setGlobals $ORG

  println "Installing chaincode on org${ORG} peer..."

  set -x
  peer lifecycle chaincode install ${CC_NAME}.tar.gz >&log.txt
  res=$?
  { set +x; } 2>/dev/null
  cat log.txt

  if [ $res -ne 0 ]; then
    println "ERROR: Chaincode installation on org${ORG} peer failed"
    exit 1
  fi

  println "Chaincode installed on org${ORG} peer"
}

queryInstalled() {
  ORG=$1
  setGlobals $ORG

  set -x
  peer lifecycle chaincode queryinstalled >&log.txt
  res=$?
  { set +x; } 2>/dev/null

  PACKAGE_ID=$(sed -n "/${CC_NAME}_${CC_VERSION}/{s/^Package ID: //; s/, Label:.*$//; p;}" log.txt)

  if [ -z "$PACKAGE_ID" ]; then
    println "ERROR: Failed to query installed chaincode"
    exit 1
  fi

  println "Package ID: ${PACKAGE_ID}"
}

approveForMyOrg() {
  ORG=$1
  setGlobals $ORG

  println "Approving chaincode for org${ORG}..."

  set -x
  peer lifecycle chaincode approveformyorg \
    -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.shipment.com \
    --tls \
    --cafile "${PWD}/organizations/ordererOrganizations/shipment.com/orderers/orderer.shipment.com/msp/tlscacerts/tlsca.shipment.com-cert.pem" \
    --channelID $CHANNEL_NAME \
    --name ${CC_NAME} \
    --version ${CC_VERSION} \
    --package-id ${PACKAGE_ID} \
    --sequence ${CC_SEQUENCE} \
    ${INIT_REQUIRED} \
    ${CC_END_POLICY:+--signature-policy "$CC_END_POLICY"} \
    ${CC_COLL_CONFIG:+--collections-config "$CC_COLL_CONFIG"} >&log.txt
  res=$?
  { set +x; } 2>/dev/null
  cat log.txt

  if [ $res -ne 0 ]; then
    println "ERROR: Chaincode approval failed for org${ORG}"
    exit 1
  fi

  println "Chaincode approved for org${ORG}"
}

checkCommitReadiness() {
  ORG=$1
  setGlobals $ORG

  println "Checking commit readiness on org${ORG}..."

  local rc=1
  local COUNTER=1
  while [ $rc -ne 0 -a $COUNTER -lt $MAX_RETRY ]; do
    sleep $DELAY
    set -x
    peer lifecycle chaincode checkcommitreadiness \
      --channelID $CHANNEL_NAME \
      --name ${CC_NAME} \
      --version ${CC_VERSION} \
      --sequence ${CC_SEQUENCE} \
      ${INIT_REQUIRED} \
      ${CC_END_POLICY:+--signature-policy "$CC_END_POLICY"} \
      ${CC_COLL_CONFIG:+--collections-config "$CC_COLL_CONFIG"} \
      --output json >&log.txt
    res=$?
    { set +x; } 2>/dev/null
    cat log.txt
    rc=0
    COUNTER=$(expr $COUNTER + 1)
  done
}

commitChaincodeDefinition() {
  println "Committing chaincode definition to channel..."

  PEER_CONN_PARMS=""

  setGlobals 1
  PEER_CONN_PARMS="${PEER_CONN_PARMS} --peerAddresses localhost:7051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/manufacturer.shipment.com/peers/peer0.manufacturer.shipment.com/tls/ca.crt"

  setGlobals 2
  PEER_CONN_PARMS="${PEER_CONN_PARMS} --peerAddresses localhost:9051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/transporter.shipment.com/peers/peer0.transporter.shipment.com/tls/ca.crt"

  setGlobals 1

  set -x
  peer lifecycle chaincode commit \
    -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.shipment.com \
    --tls \
    --cafile "${PWD}/organizations/ordererOrganizations/shipment.com/orderers/orderer.shipment.com/msp/tlscacerts/tlsca.shipment.com-cert.pem" \
    --channelID $CHANNEL_NAME \
    --name ${CC_NAME} \
    --version ${CC_VERSION} \
    --sequence ${CC_SEQUENCE} \
    ${INIT_REQUIRED} \
    ${CC_END_POLICY:+--signature-policy "$CC_END_POLICY"} \
    ${CC_COLL_CONFIG:+--collections-config "$CC_COLL_CONFIG"} \
    ${PEER_CONN_PARMS} >&log.txt
  res=$?
  { set +x; } 2>/dev/null
  cat log.txt

  if [ $res -ne 0 ]; then
    println "ERROR: Chaincode commit failed"
    exit 1
  fi

  println "Chaincode committed to channel"
}

queryCommitted() {
  ORG=$1
  setGlobals $ORG

  local rc=1
  local COUNTER=1
  while [ $rc -ne 0 -a $COUNTER -lt $MAX_RETRY ]; do
    sleep $DELAY
    set -x
    peer lifecycle chaincode querycommitted \
      --channelID $CHANNEL_NAME \
      --name ${CC_NAME} >&log.txt
    res=$?
    { set +x; } 2>/dev/null
    cat log.txt
    if [ $res -eq 0 ]; then
      rc=0
    fi
    COUNTER=$(expr $COUNTER + 1)
  done
}

chaincodeInvokeInit() {
  println "Invoking chaincode init function '${CC_INIT_FCN}'..."

  PEER_CONN_PARMS=""
  PEER_CONN_PARMS="${PEER_CONN_PARMS} --peerAddresses localhost:7051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/manufacturer.shipment.com/peers/peer0.manufacturer.shipment.com/tls/ca.crt"
  PEER_CONN_PARMS="${PEER_CONN_PARMS} --peerAddresses localhost:9051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/transporter.shipment.com/peers/peer0.transporter.shipment.com/tls/ca.crt"

  setGlobals 1

  set -x
  peer chaincode invoke \
    -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.shipment.com \
    --tls \
    --cafile "${PWD}/organizations/ordererOrganizations/shipment.com/orderers/orderer.shipment.com/msp/tlscacerts/tlsca.shipment.com-cert.pem" \
    -C $CHANNEL_NAME \
    -n ${CC_NAME} \
    ${PEER_CONN_PARMS} \
    -c "{\"function\":\"${CC_INIT_FCN}\",\"Args\":[]}" >&log.txt
  res=$?
  { set +x; } 2>/dev/null
  cat log.txt

  if [ $res -ne 0 ]; then
    println "ERROR: Chaincode invoke init failed"
    exit 1
  fi

  println "Chaincode initialization complete"
}


if [ -n "$CC_INIT_FCN" ] && [ "$CC_INIT_FCN" != "NA" ]; then
  INIT_REQUIRED="--init-required"
fi

println "========== Deploying Chaincode =========="
println "Channel:    ${CHANNEL_NAME}"
println "Chaincode:  ${CC_NAME}"
println "Source:     ${CC_SRC_PATH}"
println "Language:   ${CC_LANGUAGE}"
println "Version:    ${CC_VERSION}"
println "Sequence:   ${CC_SEQUENCE}"
println "Init Func:  ${CC_INIT_FCN}"
println "========================================"

packageChaincode

installChaincode 1
installChaincode 2

queryInstalled 1

approveForMyOrg 1
approveForMyOrg 2

checkCommitReadiness 1
checkCommitReadiness 2

commitChaincodeDefinition

queryCommitted 1
queryCommitted 2

if [ -n "$CC_INIT_FCN" ] && [ "$CC_INIT_FCN" != "NA" ]; then
  chaincodeInvokeInit
fi

println "========== Chaincode Deployment Complete =========="
