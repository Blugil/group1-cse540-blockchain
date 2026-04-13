# Shipment Tracking Network Control Script
# CSE 540 – Spring B 2026 | Group 1

set -e

export PATH=${PWD}/../bin:$PATH
export FABRIC_CFG_PATH=${PWD}/configtx
export VERBOSE=false

CHANNEL_NAME="shipchannel"
CC_NAME="shipment"
CC_SRC_PATH="../chaincode/shipment"
CC_LANGUAGE="go"
CC_VERSION="1.0"
CC_SEQUENCE=1
CC_INIT_FCN="InitLedger"
CC_END_POLICY=""
CC_COLL_CONFIG=""
DELAY=3
MAX_RETRY=5
COMPOSE_FILE_BASE="docker/docker-compose-test-net.yaml"
COMPOSE_FILE_CA="docker/docker-compose-ca.yaml"

C_RESET='\033[0m'
C_RED='\033[0;31m'
C_GREEN='\033[0;32m'
C_BLUE='\033[0;34m'
C_YELLOW='\033[1;33m'

println() { echo -e "$1"; }
infoln() { println "${C_BLUE}[INFO]${C_RESET} $1"; }
successln() { println "${C_GREEN}[SUCCESS]${C_RESET} $1"; }
warnln() { println "${C_YELLOW}[WARN]${C_RESET} $1"; }
errorln() { println "${C_RED}[ERROR]${C_RESET} $1"; }
fatalln() { errorln "$1"; exit 1; }

verifyPrereqs() {
  infoln "Verifying prerequisites..."

  if ! command -v docker &> /dev/null; then
    fatalln "Docker is not installed. Please install Docker first."
  fi

  if ! command -v docker compose &> /dev/null && ! command -v docker-compose &> /dev/null; then
    fatalln "Docker Compose is not installed."
  fi

  local FABRIC_TOOLS=("peer" "osnadmin" "configtxgen" "cryptogen")
  for tool in "${FABRIC_TOOLS[@]}"; do
    if ! command -v "$tool" &> /dev/null; then
      warnln "$tool not found in PATH. Attempting to use from fabric-samples/bin..."
    fi
  done

  successln "Prerequisites verified."
}

createOrgs() {
  infoln "Generating crypto material..."

  if [ -d "organizations/peerOrganizations" ]; then
    rm -rf organizations/peerOrganizations
    rm -rf organizations/ordererOrganizations
  fi

  cryptogen generate --config=./organizations/cryptogen/crypto-config-manufacturer.yaml --output="organizations"
  if [ $? -ne 0 ]; then
    fatalln "Failed to generate crypto material for ManufacturerOrg"
  fi

  cryptogen generate --config=./organizations/cryptogen/crypto-config-transporter.yaml --output="organizations"
  if [ $? -ne 0 ]; then
    fatalln "Failed to generate crypto material for TransporterOrg"
  fi

  cryptogen generate --config=./organizations/cryptogen/crypto-config-orderer.yaml --output="organizations"
  if [ $? -ne 0 ]; then
    fatalln "Failed to generate crypto material for Orderer"
  fi

  successln "Crypto material generated."
}

createChannel() {
  infoln "Creating channel '${CHANNEL_NAME}'..."

  scripts/createChannel.sh $CHANNEL_NAME $DELAY $MAX_RETRY $VERBOSE
  if [ $? -ne 0 ]; then
    fatalln "Create channel failed"
  fi

  successln "Channel '${CHANNEL_NAME}' created successfully."
}

deployCC() {
  infoln "Deploying chaincode '${CC_NAME}' to channel '${CHANNEL_NAME}'..."

  scripts/deployCC.sh $CHANNEL_NAME $CC_NAME $CC_SRC_PATH $CC_LANGUAGE $CC_VERSION $CC_SEQUENCE $CC_INIT_FCN $CC_END_POLICY $CC_COLL_CONFIG $DELAY $MAX_RETRY $VERBOSE

  if [ $? -ne 0 ]; then
    fatalln "Deploying chaincode failed"
  fi

  successln "Chaincode '${CC_NAME}' deployed successfully."
}

networkUp() {
  infoln "Starting Shipment Tracking Network..."

  if [ ! -d "organizations/peerOrganizations" ]; then
    createOrgs
  fi

  COMPOSE_FILES="-f ${COMPOSE_FILE_BASE}"

  if [ "$USE_CA" = true ]; then
    COMPOSE_FILES="${COMPOSE_FILES} -f ${COMPOSE_FILE_CA}"
  fi

  docker compose ${COMPOSE_FILES} up -d 2>&1

  sleep 5

  docker ps -a
  if [ $? -ne 0 ]; then
    fatalln "Unable to start network"
  fi

  successln "Network started successfully."
}

networkDown() {
  infoln "Stopping Shipment Tracking Network..."

  COMPOSE_FILES="-f ${COMPOSE_FILE_BASE} -f ${COMPOSE_FILE_CA}"
  docker compose ${COMPOSE_FILES} down --volumes --remove-orphans 2>&1 || true

  if [ -d "organizations/peerOrganizations" ]; then
    rm -rf organizations/peerOrganizations
    rm -rf organizations/ordererOrganizations
  fi

  rm -rf channel-artifacts

  rm -rf *.tar.gz

  successln "Network stopped and cleaned up."
}

USE_CA=false
MODE=""
SUBCOMMAND=""

while [[ $# -ge 1 ]]; do
  key="$1"
  case $key in
    up)
      MODE="up"
      shift
      ;;
    down)
      MODE="down"
      shift
      ;;
    createChannel)
      SUBCOMMAND="createChannel"
      shift
      ;;
    deployCC)
      SUBCOMMAND="deployCC"
      shift
      ;;
    -c)
      CHANNEL_NAME="$2"
      shift 2
      ;;
    -ca)
      USE_CA=true
      shift
      ;;
    -ccn)
      CC_NAME="$2"
      shift 2
      ;;
    -ccp)
      CC_SRC_PATH="$2"
      shift 2
      ;;
    -ccl)
      CC_LANGUAGE="$2"
      shift 2
      ;;
    -ccv)
      CC_VERSION="$2"
      shift 2
      ;;
    -ccs)
      CC_SEQUENCE="$2"
      shift 2
      ;;
    -cci)
      CC_INIT_FCN="$2"
      shift 2
      ;;
    *)
      errorln "Unknown flag: $key"
      exit 1
      ;;
  esac
done

println ""
println "  ____  _   _ ___ ____  __  __ _____ _   _ _____"
println " / ___|| | | |_ _|  _ \\|  \\/  | ____| \\ | |_   _|"
println " \\___ \\| |_| || || |_) | |\\/| |  _| |  \\| | | |  "
println "  ___) |  _  || ||  __/| |  | | |___| |\\  | | |  "
println " |____/|_| |_|___|_|   |_|  |_|_____|_| \\_| |_|  "
println ""
println " Blockchain-Based Shipment Tracking System"
println " CSE 540 – Spring B 2026 | Group 1"
println ""

if [ "$MODE" = "up" ]; then
  verifyPrereqs
  networkUp
  if [ "$SUBCOMMAND" = "createChannel" ]; then
    createChannel
  fi
elif [ "$MODE" = "down" ]; then
  networkDown
elif [ "$SUBCOMMAND" = "createChannel" ]; then
  createChannel
elif [ "$SUBCOMMAND" = "deployCC" ]; then
  deployCC
else
  println "Usage:"
  println "  ./network.sh up                              - Start the network"
  println "  ./network.sh up createChannel -c shipchannel - Start and create channel"
  println "  ./network.sh up createChannel -ca            - Start with Certificate Authorities"
  println "  ./network.sh createChannel -c shipchannel    - Create channel on running network"
  println "  ./network.sh deployCC -c shipchannel -ccn shipment -ccp ../chaincode/shipment -ccl go"
  println "  ./network.sh down                            - Tear down the network"
fi
