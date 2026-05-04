# Blockchain-Based Shipment Tracking System

**CSE 540 – Engineering Blockchain Applications | Spring B 2026 | Group 1**  
Dominick Agnello · Ritish Abrol · Vatsal Patel · Shashikant Nanda · Anushree Bhure

---

## Project Description

This project applies blockchain technology to the shipping logistics industry using **Hyperledger Fabric** and **smart contracts** (chaincode) to track, monitor, and immutably record product movement data across the full custody chain.

Each time a shipment changes hands — from **manufacturer**, to **transporter**, to **warehouse**, to **retailer**, to **recipient** — a digitally signed, timestamped transaction is appended to the ledger. This creates a tamper-proof chain of custody that all authorized parties can audit in real time.

### Key Features

- **Immutable chain of custody** — every handoff is recorded on-chain with digital signatures and timestamps
- **Role-based access control** — only authorized participants can interact with a shipment
- **Off-chain data hashing** — sensitive metadata (weight, volume, contents) is hashed on-chain for integrity verification
- **Event-driven architecture** — chaincode events enable real-time notifications to SDK listeners
- **RESTful API** — Node.js/Express client application for easy integration
- **CouchDB rich queries** — supports complex queries against the world state

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     Client Application (REST API)               │
│                   Node.js + Express + Fabric SDK                │
└──────────────────────────┬──────────────────────────────────────┘
                           │  gRPC / Fabric Gateway
┌──────────────────────────▼──────────────────────────────────────┐
│                    Hyperledger Fabric Network                    │
│                                                                 │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────────┐   │
│  │ Manufacturer │  │  Transporter  │  │  Orderer (Raft)     │   │
│  │   Peer (Org1)│  │  Peer (Org2) │  │  orderer.shipment   │   │
│  │  + CouchDB   │  │  + CouchDB   │  │       .com          │   │
│  └──────┬───────┘  └──────┬───────┘  └─────────────────────┘   │
│         │                 │                                     │
│  ┌──────▼─────────────────▼──────┐                              │
│  │   Shipment Chaincode (Go)     │                              │
│  │   - CreateShipment            │                              │
│  │   - TransferCustody           │                              │
│  │   - UpdateShipmentStatus      │                              │
│  │   - VerifyShipment            │                              │
│  │   - GetShipmentHistory        │                              │
│  │   - AuthorizeParticipant      │                              │
│  │   - RevokeParticipant         │                              │
│  │   - GetAllShipments           │                              │
│  └───────────────────────────────┘                              │
└─────────────────────────────────────────────────────────────────┘
                           │
              ┌────────────▼────────────┐
              │   Off-Chain Storage     │
              │   (IPFS / Encrypted DB) │
              │   Hash stored on-chain  │
              └─────────────────────────┘
```

### Stakeholder Roles

| Role | MSP ID | Description |
|------|--------|-------------|
| Manufacturer | `ManufacturerMSP` | Creates shipments and initiates the chain of custody |
| Transporter | `TransporterMSP` | Moves shipments between locations |
| Warehouse | `WarehouseMSP` | Stores shipments in warehouse facilities |
| Retailer | `RetailerMSP` | Receives shipments for retail distribution |
| Recipient | `RecipientMSP` | Final delivery recipient |

---

## Repository Structure

```
group1-cse540-blockchain/
├── chaincode/
│   └── shipment/
│       ├── main.go          # Smart contract implementation (Go)
│       └── go.mod            # Go module dependencies
├── client/
│   ├── package.json          # Node.js client dependencies
│   └── src/
│       ├── app.js            # Express REST API server
│       ├── fabricConfig.js   # Fabric connection profile resolver
│       ├── ipfsClient.js     # IPFS upload and SHA-256 utilities
│       └── importCryptoIdentity.js  # Wallet identity importer
├── network/
│   ├── configtx/
│   │   └── configtx.yaml     # Channel configuration
│   ├── docker/
│   │   ├── docker-compose-test-net.yaml  # Peers, orderer, CouchDB
│   │   └── docker-compose-ca.yaml        # Certificate Authorities
│   ├── organizations/
│   │   └── cryptogen/        # Crypto material config
│   │       ├── crypto-config-manufacturer.yaml
│   │       ├── crypto-config-transporter.yaml
│   │       └── crypto-config-orderer.yaml
│   └── scripts/
│       ├── createChannel.sh  # Channel creation
│       ├── deployCC.sh       # Chaincode deployment
│       ├── envVar.sh         # Environment variable helpers
│       └── testChaincode.sh  # CLI-based chaincode tests
├── .gitignore
└── README.md
```

---

## Dependencies & Setup Instructions

### Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| [Docker](https://www.docker.com/) | 20.x+ | Container runtime |
| [Docker Compose](https://docs.docker.com/compose/) | 2.x+ | Multi-container orchestration |
| [Go](https://go.dev/) | 1.21+ | Chaincode compilation |
| [Node.js](https://nodejs.org/) | 18.x+ | Client application |
| [Hyperledger Fabric Binaries](https://hyperledger-fabric.readthedocs.io/en/latest/install.html) | 2.5.x | Peer, orderer, configtxgen, cryptogen |
| [jq](https://stedolan.github.io/jq/) | any | JSON processing in scripts |

### Step 1: Install Hyperledger Fabric Binaries

```bash
curl -sSL https://bit.ly/2ysbOFE | bash -s -- 2.5.0 1.5.7
export PATH=$PATH:$PWD/fabric-samples/bin
export FABRIC_CFG_PATH=$PWD/fabric-samples/config
export FABRIC_SAMPLES_PATH=$PWD/fabric-samples
```

### Step 2: Clone This Repository

```bash
git clone https://github.com/Blugil/group1-cse540-blockchain.git
cd group1-cse540-blockchain
```
### Step 3: Vendor the chaincode

```bash
cd group1-cse540-blockchain/chaincode/shipment
go mod tidy
go mod vendor
```

### Step 4: Start Everything

```bash
# From the project root — starts the network, deploys chaincode, and launches the API
./start.sh
```

This single script:
1. Starts the Fabric test-network (2 orgs, CouchDB)
2. Builds and deploys the chaincode via CCaaS (no Docker-in-Docker required)
3. Imports crypto identities into the SDK wallet
4. Starts the REST API + dashboard on http://localhost:3000

> **Node.js version:** `fabric-network` 2.2.x requires Node 18. If your default is Node 20+, the script auto-switches via nvm. You can also set it manually: `nvm use 18`.

### Step 4: Open the Dashboard

Navigate to **http://localhost:3000** in your browser to use the UI, or interact via REST API directly (see endpoints below).

### Tear Down

```bash
./stop.sh
```

---

## Smart Contract API Reference

### `InitLedger()`
Initializes the ledger with a sample shipment (`SHIP-001`) for demonstration.

### `CreateShipment(shipmentID, origin, destination, participantsJSON, offChainData)`
Registers a new shipment on the ledger. The caller becomes the initial holder. Off-chain data is SHA-256 hashed and stored on-chain.

### `GetShipment(shipmentID)` → `Shipment`
Returns the current state of a shipment. Access restricted to authorized participants.

### `UpdateShipmentStatus(shipmentID, status, location, notes)`
Updates the shipment status. Only the current holder can update. Valid statuses: `Created`, `InTransit`, `InWarehouse`, `WithRetailer`, `Delivered`.

### `TransferCustody(shipmentID, newHolder)`
Transfers the shipment to a new holder. Only the current holder can initiate. The new holder must be an authorized participant.

### `VerifyShipment(shipmentID, offChainData)` → `bool`
Verifies data integrity by comparing the SHA-256 hash of provided data against the on-chain hash.

### `GetShipmentHistory(shipmentID)` → `[]ShipmentEvent`
Returns the full event history for a shipment (creation, transfers, status updates, etc.).

### `AuthorizeParticipant(shipmentID, participant)`
Adds a new participant to the shipment's authorized list. Only the current holder can authorize.

### `RevokeParticipant(shipmentID, participant)`
Removes a participant from the authorized list. The current holder cannot revoke themselves.

### `GetAllShipments()` → `[]Shipment`
Returns all shipments stored on the ledger.

---

## REST API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/health` | Health check |
| `POST` | `/api/shipments` | Create a new shipment |
| `GET` | `/api/shipments` | Get all shipments |
| `GET` | `/api/shipments/:id` | Get a specific shipment |
| `PUT` | `/api/shipments/:id/status` | Update shipment status |
| `PUT` | `/api/shipments/:id/transfer` | Transfer custody |
| `GET` | `/api/shipments/:id/verify?offChainData=...` | Verify data integrity |
| `GET` | `/api/shipments/:id/history` | Get event history |
| `POST` | `/api/shipments/:id/participants` | Authorize a participant |
| `DELETE` | `/api/shipments/:id/participants/:participant` | Revoke a participant |

---

## Shipment Lifecycle Flow

```
  Manufacturer          Transporter          Warehouse           Retailer            Recipient
      │                     │                    │                   │                    │
      │ CreateShipment()    │                    │                   │                    │
      ├─────────────────────┤                    │                   │                    │
      │                     │                    │                   │                    │
      │ TransferCustody()   │                    │                   │                    │
      │────────────────────>│                    │                   │                    │
      │                     │ TransferCustody()  │                   │                    │
      │                     │───────────────────>│                   │                    │
      │                     │                    │ TransferCustody() │                    │
      │                     │                    │──────────────────>│                    │
      │                     │                    │                   │ TransferCustody()  │
      │                     │                    │                   │───────────────────>│
      │                     │                    │                   │                    │
      │                     │                    │                   │ UpdateStatus()     │
      │                     │                    │                   │ ("Delivered")      │
      │                     │                    │                   │                    │
      ▼                     ▼                    ▼                   ▼                    ▼
  ┌────────────────────────────────────────────────────────────────────────────────────────┐
  │                         Immutable Blockchain Ledger                                   │
  │                    (All events timestamped & digitally signed)                         │
  └────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## References

1. D. Shakhbulatov, J. Medina, Z. Dong and R. Rojas-Cessa, "How Blockchain Enhances Supply Chain Management: A Survey," in *IEEE Open Journal of the Computer Society*, vol. 1, pp. 230-249, 2020.
2. S. Oğuz, G. Alkan, B. Yilmaz and C. Kocabaş, "The Use of Blockchain Technology in Logistics and Supply Chain Management (SCM): A Systematic Review," in *IEEE Access*, vol. 12, pp. 166211-166224, 2024.
3. P. Gonczol, P. Katsikouli, L. Herskind and N. Dragoni, "Blockchain Implementations and Use Cases for Supply Chains—A Survey," in *IEEE Access*, vol. 8, pp. 11856-11871, 2020.
4. Hyperledger Fabric Documentation, https://hyperledger-fabric.readthedocs.io/

---

## License

This project is developed as part of CSE 540 coursework at Arizona State University.
