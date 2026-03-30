# Blockchain-Based Shipment Tracking System

**CSE 540 – Spring B 2026 | Group 1**  
Dominick Agnello · Ritish Abrol · Vatsal Patel · Shashikant Nanda · Anushree Bhure

---

## Project Description

This project applies blockchain technology to the shipping logistics industry using **Hyperledger Fabric** and **smart contracts** to track, monitor, and immutably record product movement data across the full custody chain.

Each time a shipment changes hands — from manufacturer, to transporter, to warehouse, to retailer, to recipient — a digitally signed, timestamped transaction is appended to the ledger. This creates a tamper-proof chain of custody that all authorized parties can audit in real time.

## Dependencies & Setup Instructions

### Prerequisites

| Tool | Version |
|------|---------|
| [Docker](https://www.docker.com/) | 20.x+ |
| [Docker Compose](https://docs.docker.com/compose/) | 2.x+ |
| [Go](https://go.dev/) | 1.21+ |
| [Node.js](https://nodejs.org/) | 18.x+ (for SDK/client) |
| [Hyperledger Fabric Binaries](https://hyperledger-fabric.readthedocs.io/en/latest/install.html) | 2.5.x |
| [jq](https://stedolan.github.io/jq/) | any recent version |

### Install Hyperledger Fabric

```bash
curl -sSL https://bit.ly/2ysbOFE | bash -s -- 2.5.0 1.5.7
export PATH=$PATH:$PWD/fabric-samples/bin
```

### Clone This Repository

```bash
git clone (our git repo link)
cd blockchain-shipment-tracker
```

### Start the Network (Local Dev)

```bash
cd network/
./network.sh up createChannel -c shipchannel -ca
```

### Deploy Chaincode

```bash
./network.sh deployCC -c shipchannel -ccn shipment -ccp ../chaincode/ -ccl go
```
