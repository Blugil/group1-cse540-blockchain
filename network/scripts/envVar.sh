setGlobals() {
  local USING_ORG=$1

  echo "Using organization ${USING_ORG}"

  if [ $USING_ORG -eq 1 ]; then
    export CORE_PEER_LOCALMSPID="ManufacturerMSP"
    export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/manufacturer.shipment.com/peers/peer0.manufacturer.shipment.com/tls/ca.crt
    export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/manufacturer.shipment.com/users/Admin@manufacturer.shipment.com/msp
    export CORE_PEER_ADDRESS=localhost:7051
  elif [ $USING_ORG -eq 2 ]; then
    export CORE_PEER_LOCALMSPID="TransporterMSP"
    export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/transporter.shipment.com/peers/peer0.transporter.shipment.com/tls/ca.crt
    export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/transporter.shipment.com/users/Admin@transporter.shipment.com/msp
    export CORE_PEER_ADDRESS=localhost:9051
  else
    echo "Unknown organization: ${USING_ORG}"
    exit 1
  fi

  if [ "$VERBOSE" = true ]; then
    env | grep CORE
  fi
}

setGlobalsByOrgName() {
  local ORG_NAME=$1

  case $ORG_NAME in
    ManufacturerMSP | manufacturer)
      setGlobals 1
      ;;
    TransporterMSP | transporter)
      setGlobals 2
      ;;
    *)
      echo "Unknown organization name: ${ORG_NAME}"
      exit 1
      ;;
  esac
}

verifyResult() {
  if [ $1 -ne 0 ]; then
    echo "$2"
    exit 1
  fi
}

export -f setGlobals
export -f setGlobalsByOrgName
export -f verifyResult
