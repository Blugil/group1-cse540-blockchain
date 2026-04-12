# 📦 Blockchain-Based Shipment Tracking System
### **CSE 540 – Engineering Blockchain Applications | Spring B 2026 | Group 1**
**Dominick Agnello · Ritish Abrol · Vatsal Patel · Shashikant Nanda · Anushree Bhure**

---

## Table of Contents

1. [Introduction — The Story](#1-introduction--the-story)
2. [Problem Statement — What Goes Wrong Today](#2-problem-statement--what-goes-wrong-today)
3. [Why Blockchain? — A Simple Explanation](#3-why-blockchain--a-simple-explanation)
4. [How Our System Works — Step by Step](#4-how-our-system-works--step-by-step)
5. [System Architecture — The Big Picture](#5-system-architecture--the-big-picture)
6. [Technologies Used](#6-technologies-used)
7. [Smart Contract Design — The Rules of the Game](#7-smart-contract-design--the-rules-of-the-game)
8. [Workflow Walkthrough — A Day in the Life of a Shipment](#8-workflow-walkthrough--a-day-in-the-life-of-a-shipment)
9. [Testing & Verification](#9-testing--verification)
10. [Challenges & Limitations](#10-challenges--limitations)
11. [Conclusion](#11-conclusion)
12. [References](#12-references)

---

## 1. Introduction — The Story

Imagine you order a brand-new laptop online. You get a tracking number. You check it every day. But one morning, the status says "Delivered" — except you never received it. You call the retailer, who blames the transporter. The transporter blames the warehouse. Nobody knows where the laptop actually is, because every company uses its own private system, and nobody trusts the other's records.

**This is the problem we set out to solve.**

Our project is a **Blockchain-Based Shipment Tracking System** — a system where every handoff of a package, from the factory floor to your doorstep, is recorded on a shared, tamper-proof digital ledger that no single party can alter or erase. If the transporter says they handed the package to the warehouse, there is a permanent, timestamped, digitally-signed record that proves (or disproves) it.

We built this using **Hyperledger Fabric**, a permissioned blockchain framework designed for business applications — not cryptocurrency, but real-world enterprise problems like supply chain management.

---

## 2. Problem Statement — What Goes Wrong Today

Modern supply chains involve **many organizations** — manufacturers, transporters, warehouses, retailers, and end customers. Each of these keeps its own records in its own database. This creates several serious problems:

```
┌────────────────────────────────────────────────────────────────────────────┐
│                     THE TRUST PROBLEM IN SUPPLY CHAINS                    │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│   Manufacturer ──→ Transporter ──→ Warehouse ──→ Retailer ──→ Customer   │
│     (System A)      (System B)     (System C)    (System D)               │
│                                                                           │
│   ❌ Each has their OWN database                                          │
│   ❌ No single shared "source of truth"                                   │
│   ❌ Records can be altered or deleted                                     │
│   ❌ Disputes are hard to resolve — he-said / she-said                    │
│   ❌ Counterfeit goods can slip through undetected                        │
│   ❌ Delays and losses are hard to trace                                  │
│                                                                           │
└────────────────────────────────────────────────────────────────────────────┘
```

### Real-World Impact

| Problem | Example |
|---------|---------|
| **Fraud & counterfeiting** | Fake pharmaceuticals enter the supply chain — patients receive counterfeit medicines |
| **Theft & loss** | A shipment disappears between the transporter and the warehouse — nobody can prove who had it last |
| **Disputes** | Manufacturer says they shipped 100 units; warehouse says they received 90. Who is right? |
| **Lack of traceability** | A food recall happens — the company cannot identify which batches went where |
| **Delays** | Paperwork gets lost, customs forms are duplicated, nobody has the full picture |

**The root cause is simple: there is no shared, trustworthy, tamper-proof record that all parties can rely on.**

---

## 3. Why Blockchain? — A Simple Explanation

### 🧱 What IS a Blockchain? (For Absolute Beginners)

Think of a blockchain as a **shared notebook** that many people can write in, but nobody can erase or change what was already written. Every time someone writes something new, everyone else gets a copy of that page. If someone tries to change an old page, everyone else's copies would not match — and the tampering would be detected immediately.

```
  ┌─────────┐     ┌─────────┐     ┌─────────┐     ┌─────────┐
  │ Block 1 │────▶│ Block 2 │────▶│ Block 3 │────▶│ Block 4 │
  │         │     │         │     │         │     │         │
  │ Shipment│     │ Custody │     │ Status  │     │ Custody │
  │ Created │     │ Transfer│     │ Update  │     │ Transfer│
  │         │     │ Mfg→Trn │     │ InTranst│     │ Trn→Whs │
  │ Hash: a1│     │ Hash: b2│     │ Hash: c3│     │ Hash: d4│
  │ Prev: 00│     │ Prev: a1│     │ Prev: b2│     │ Prev: c3│
  └─────────┘     └─────────┘     └─────────┘     └─────────┘

  ← Each block points to the previous one. Change one, and the
    entire chain after it breaks. That's what makes it tamper-proof. →
```

### 📝 Key Terms Explained Simply

| Term | Simple Explanation |
|------|-------------------|
| **Blockchain** | A shared digital notebook that nobody can erase. Every new entry is linked to the previous one. |
| **Block** | One page in the notebook. It contains one or more transactions (entries). |
| **Transaction** | A single recorded action — like "Manufacturer transferred the shipment to Transporter." |
| **Ledger** | The complete history of all transactions — the full notebook. |
| **Smart Contract** | A set of rules written in code that run automatically. Think of it like a vending machine: put in the right input, get the right output — no human needed. |
| **Node / Peer** | A computer that holds a copy of the ledger and runs the smart contracts. |
| **Consensus** | The process by which all peers agree that a transaction is valid before it's added to the ledger. |
| **MSP (Membership Service Provider)** | The "ID card" system. It verifies who you are before you're allowed to do anything on the network. |
| **Chaincode** | Hyperledger Fabric's word for "smart contract." Written in Go, Java, or JavaScript. |
| **CouchDB** | A database that stores the current state of all data on the blockchain for fast queries. |

### 🤔 Why Not Just Use a Regular Database?

| Feature | Regular Database | Blockchain |
|---------|-----------------|------------|
| Who controls it? | One company | Shared among all parties |
| Can records be changed? | Yes (by admin) | No — once written, it's permanent |
| Do all parties see the same data? | No | Yes — everyone has the same copy |
| Is there an audit trail? | Maybe, if the company chooses to keep one | Always — every change is recorded with a timestamp and digital signature |
| Trust required? | You must trust the database owner | Trust is built into the system — no single party controls it |

### 🔒 Why Hyperledger Fabric Specifically?

We chose Hyperledger Fabric — not Bitcoin or Ethereum — because:

```
┌───────────────────────────────────────────────────────────────────────┐
│                    HYPERLEDGER FABRIC vs. PUBLIC BLOCKCHAINS          │
├───────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ✅ Permissioned — only invited organizations can join                │
│     (A public blockchain like Bitcoin lets anyone participate)         │
│                                                                       │
│  ✅ No cryptocurrency / No gas fees                                   │
│     (Ethereum requires ETH to run smart contracts — we don't)         │
│                                                                       │
│  ✅ Private data — companies can share selective information           │
│     (On Bitcoin, everything is visible to everyone)                    │
│                                                                       │
│  ✅ High performance — thousands of transactions per second            │
│     (Bitcoin: ~7 TPS, Ethereum: ~30 TPS, Fabric: ~3000+ TPS)         │
│                                                                       │
│  ✅ Identity-based — every participant has a verified identity (MSP)   │
│     (On Bitcoin, participants are anonymous)                           │
│                                                                       │
│  ✅ Enterprise-grade — designed by the Linux Foundation for business   │
│                                                                       │
└───────────────────────────────────────────────────────────────────────┘
```

---

## 4. How Our System Works — Step by Step

Here is the journey of a shipment through our system, told as a story:

### Step 1: A Shipment is Born 🏭

A manufacturer in Phoenix, AZ creates a new shipment on the blockchain. They specify:
- A unique shipment ID (e.g., `SHIP-042`)
- Where it's coming from (`Factory-Phoenix-AZ`)
- Where it's going (`Warehouse-Dallas-TX`)
- Who is allowed to touch it (Manufacturer, Transporter, Warehouse, Retailer, Recipient)
- Any off-chain data (weight, volume, contents count) — this gets **hashed** (converted into a fingerprint) and stored on-chain

```
  Manufacturer calls: CreateShipment("SHIP-042", "Phoenix", "Dallas", 
                       ["ManufacturerMSP", "TransporterMSP", ...], 
                       "weight:25kg,volume:1cbm")

  → Blockchain records: Shipment CREATED ✅
  → SHA-256 hash of metadata stored on-chain
  → Event emitted: "CREATE" — all listeners notified
```

### Step 2: Handing Off to the Transporter 🚛

When the truck arrives, the manufacturer "transfers custody" of the shipment to the transporter. This is like signing a receipt — except the receipt is permanent, timestamped, and cannot be forged.

```
  Manufacturer calls: TransferCustody("SHIP-042", "TransporterMSP")

  → Blockchain records: CUSTODY_TRANSFER from ManufacturerMSP to TransporterMSP ✅
  → Status changes to "InTransit"
  → Digital signature proves the manufacturer authorized this transfer
```

### Step 3: Journey Through the Supply Chain 📦➡️📦➡️📦

The same process repeats at every stop:

```
  Transporter → Warehouse:     TransferCustody("SHIP-042", "WarehouseMSP")
  Warehouse → Retailer:        TransferCustody("SHIP-042", "RetailerMSP")
  Retailer → Recipient:        TransferCustody("SHIP-042", "RecipientMSP")
```

At each handoff, the blockchain records WHO transferred, TO WHOM, and WHEN — with cryptographic proof.

### Step 4: Status Updates Along the Way 📊

At any point, the current holder can update the status:

```
  Transporter calls: UpdateShipmentStatus("SHIP-042", "InTransit", "Highway I-10", "On schedule")
  Warehouse calls:   UpdateShipmentStatus("SHIP-042", "InWarehouse", "Dallas Facility", "Received intact")
  Recipient calls:   UpdateShipmentStatus("SHIP-042", "Delivered", "Home Address", "Package received")
```

### Step 5: Verification — Is the Data Authentic? 🔍

Anyone with access can verify the shipment's data hasn't been tampered with:

```
  Anyone calls: VerifyShipment("SHIP-042", "weight:25kg,volume:1cbm")

  → System recomputes the SHA-256 hash of the provided data
  → Compares it with the hash stored on-chain at creation time
  → If they match → ✅ Data is authentic
  → If they don't → ❌ Data has been tampered with!
```

### Step 6: Full History — The Complete Story 📜

At any time, anyone authorized can pull the entire history:

```
  GetShipmentHistory("SHIP-042") →

  ┌──────────────────────────────────────────────────────────────┐
  │ Event 1: CREATE          | ManufacturerMSP | Phoenix    | T1│
  │ Event 2: CUSTODY_TRANSFER| ManufacturerMSP | -          | T2│
  │ Event 3: STATUS_UPDATE   | TransporterMSP  | Highway    | T3│
  │ Event 4: CUSTODY_TRANSFER| TransporterMSP  | -          | T4│
  │ Event 5: STATUS_UPDATE   | WarehouseMSP    | Dallas     | T5│
  │ Event 6: CUSTODY_TRANSFER| WarehouseMSP    | -          | T6│
  │ Event 7: CUSTODY_TRANSFER| RetailerMSP     | -          | T7│
  │ Event 8: STATUS_UPDATE   | RecipientMSP    | Home       | T8│
  └──────────────────────────────────────────────────────────────┘
  
  Every event is immutable, timestamped, and digitally signed.
```

---

## 5. System Architecture — The Big Picture

Our system has **three layers**, like a cake:

```
┌═══════════════════════════════════════════════════════════════════════════┐
║                          LAYER 1: CLIENT APPLICATION                     ║
║                                                                          ║
║    ┌──────────────────────────────────────────────────────────────────┐   ║
║    │              Node.js + Express REST API Server                   │   ║
║    │                                                                  │   ║
║    │   POST /api/shipments ─────── Create a new shipment              │   ║
║    │   GET  /api/shipments/:id ─── Get shipment details               │   ║
║    │   PUT  /api/shipments/:id/status ── Update status                │   ║
║    │   PUT  /api/shipments/:id/transfer ── Transfer custody           │   ║
║    │   GET  /api/shipments/:id/verify ── Verify data integrity        │   ║
║    │   GET  /api/shipments/:id/history ── Get full event history      │   ║
║    │                                                                  │   ║
║    └──────────────────────┬───────────────────────────────────────────┘   ║
║                           │ Fabric SDK (gRPC)                            ║
╠═══════════════════════════╪══════════════════════════════════════════════╣
║                           ▼                                              ║
║                    LAYER 2: BLOCKCHAIN NETWORK                           ║
║                                                                          ║
║    ┌────────────────┐    ┌────────────────┐    ┌──────────────────┐      ║
║    │ Manufacturer   │    │  Transporter   │    │    Orderer       │      ║
║    │   Peer Node    │    │   Peer Node    │    │   (Raft Mode)    │      ║
║    │                │    │                │    │                  │      ║
║    │  ┌──────────┐  │    │  ┌──────────┐  │    │  Sequences &     │      ║
║    │  │ CouchDB  │  │    │  │ CouchDB  │  │    │  orders all     │      ║
║    │  │ (State)  │  │    │  │ (State)  │  │    │  transactions    │      ║
║    │  └──────────┘  │    │  └──────────┘  │    └──────────────────┘      ║
║    └───────┬────────┘    └───────┬────────┘                              ║
║            │                     │                                       ║
║            └──────────┬──────────┘                                       ║
║                       ▼                                                  ║
║    ┌──────────────────────────────────────────────────────────────────┐   ║
║    │              CHAINCODE (Smart Contract in Go)                    │   ║
║    │                                                                  │   ║
║    │  CreateShipment()  │ TransferCustody() │ UpdateShipmentStatus() │   ║
║    │  GetShipment()     │ VerifyShipment()  │ GetShipmentHistory()   │   ║
║    │  AuthorizeParticipant() │ RevokeParticipant() │ GetAllShipments│   ║
║    └──────────────────────────────────────────────────────────────────┘   ║
║                                                                          ║
╠══════════════════════════════════════════════════════════════════════════╣
║                                                                          ║
║                    LAYER 3: IDENTITY & SECURITY                          ║
║                                                                          ║
║    ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                  ║
║    │  CA: Mfg     │  │  CA: Trans   │  │  CA: Orderer │                  ║
║    │  (Port 7054) │  │  (Port 8054) │  │  (Port 9054) │                  ║
║    └──────────────┘  └──────────────┘  └──────────────┘                  ║
║                                                                          ║
║    CA = Certificate Authority — issues digital identity cards             ║
║    Every participant must authenticate before interacting                 ║
║                                                                          ║
╚══════════════════════════════════════════════════════════════════════════╝
```

### How Do the Layers Talk to Each Other?

```
┌──────────┐          ┌──────────────┐          ┌──────────────┐
│  User /  │  HTTP    │   Express    │  gRPC    │  Fabric Peer │
│  Browser │─────────▶│   REST API   │─────────▶│  (runs the   │
│  or App  │  Request │   (Node.js)  │  via SDK │  chaincode)  │
└──────────┘          └──────────────┘          └──────┬───────┘
                                                       │
                                                       ▼
                                              ┌──────────────┐
                                              │   Orderer    │
                                              │  (sequences  │
                                              │transactions) │
                                              └──────┬───────┘
                                                       │
                                                       ▼
                                              ┌──────────────┐
                                              │  Blockchain  │
                                              │   Ledger     │
                                              │ (immutable)  │
                                              └──────────────┘
```

**In plain English:**
1. A user (or application) sends an HTTP request to our REST API (e.g., "Create a shipment")
2. The Node.js server uses the Fabric SDK to forward this request to a peer node on the blockchain network
3. The peer runs the chaincode (smart contract) — it checks the rules, validates the data, and proposes the transaction
4. The orderer receives the proposal, puts it in the correct order, and creates a new block
5. All peers receive the new block, validate it, and update their local copy of the ledger
6. The response flows back to the user: "Shipment created successfully ✅"

---

## 6. Technologies Used

### Technology Stack Diagram

```
┌────────────────────────────────────────────────────────────────┐
│                        TECHNOLOGY STACK                         │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│   FRONTEND / CLIENT                                            │
│   ┌────────────┐  ┌────────────┐  ┌─────────────────────────┐ │
│   │  Node.js   │  │  Express   │  │  Fabric SDK (Node.js)   │ │
│   │  v18+      │  │  REST API  │  │  fabric-network         │ │
│   └────────────┘  └────────────┘  │  fabric-ca-client        │ │
│                                    └─────────────────────────┘ │
│                                                                │
│   SMART CONTRACT                                               │
│   ┌────────────┐  ┌────────────────────────────────────────┐  │
│   │  Go 1.21+  │  │  fabric-contract-api-go v1.2.2        │  │
│   │            │  │  fabric-chaincode-go (shim)            │  │
│   └────────────┘  └────────────────────────────────────────┘  │
│                                                                │
│   BLOCKCHAIN NETWORK                                           │
│   ┌─────────────────────┐  ┌──────────────┐  ┌────────────┐  │
│   │ Hyperledger Fabric  │  │   Docker &   │  │  CouchDB   │  │
│   │ v2.5.x              │  │   Docker     │  │  v3.3      │  │
│   │                     │  │   Compose    │  │ (State DB) │  │
│   └─────────────────────┘  └──────────────┘  └────────────┘  │
│                                                                │
│   SECURITY & IDENTITY                                          │
│   ┌─────────────────────┐  ┌──────────────────────────────┐   │
│   │  Fabric CA          │  │  X.509 Certificates          │   │
│   │  (Certificate Auth) │  │  MSP (Membership Provider)   │   │
│   └─────────────────────┘  └──────────────────────────────┘   │
│                                                                │
│   CRYPTOGRAPHY                                                 │
│   ┌─────────────────────┐  ┌──────────────────────────────┐   │
│   │  SHA-256 Hashing    │  │  ECDSA Digital Signatures    │   │
│   │  (data integrity)   │  │  (transaction signing)       │   │
│   └─────────────────────┘  └──────────────────────────────┘   │
│                                                                │
│   CONSENSUS                                                    │
│   ┌─────────────────────────────────────────────────────────┐ │
│   │  Raft Consensus Protocol                                 │ │
│   │  (Leader-based ordering — fast & crash-fault tolerant)   │ │
│   └─────────────────────────────────────────────────────────┘ │
│                                                                │
│   TESTING                                                      │
│   ┌─────────────────────────────────────────────────────────┐ │
│   │  Go testing package + Fabric MockStub (shimtest)         │ │
│   │  15 unit tests — all passing ✅                          │ │
│   └─────────────────────────────────────────────────────────┘ │
│                                                                │
└────────────────────────────────────────────────────────────────┘
```

### Technology Breakdown

| Layer | Technology | Why We Used It |
|-------|-----------|----------------|
| **Smart Contract** | Go (Golang) | Fast, strongly typed, first-class Fabric support |
| **Smart Contract API** | `fabric-contract-api-go` v1.2.2 | Official Hyperledger API for writing chaincode |
| **Client App** | Node.js + Express | Simple REST API for interacting with the blockchain |
| **Fabric SDK** | `fabric-network` (Node.js) | Connects the client app to the Fabric peers |
| **Blockchain Platform** | Hyperledger Fabric 2.5 | Enterprise blockchain — permissioned, no crypto fees |
| **Containers** | Docker + Docker Compose | Each peer, orderer, CA, and CouchDB runs in its own container |
| **State Database** | CouchDB 3.3 | Stores the current world state; supports rich JSON queries |
| **Identity** | Fabric CA + X.509 Certs | Issues digital identity certificates for every participant |
| **Consensus** | Raft | Leader-based consensus — fast and crash-fault tolerant |
| **Hashing** | SHA-256 | Creates a unique "fingerprint" of off-chain data for verification |
| **Testing** | Go `testing` + `shimtest` MockStub | Unit tests without needing a live network |

---

## 7. Smart Contract Design — The Rules of the Game

The smart contract (chaincode) is the **brain of our system**. It contains all the business rules. Think of it as an automated judge — it decides what is allowed and what is not.

### Data Models

Our chaincode stores two types of data on the blockchain:

```
┌─────────────────────────────── SHIPMENT ─────────────────────────────────┐
│                                                                          │
│  {                                                                       │
│    "docType":       "shipment",                                          │
│    "shipmentID":    "SHIP-042",              ← Unique ID                 │
│    "status":        "InTransit",             ← Current status            │
│    "origin":        "Factory-Phoenix-AZ",    ← Where it came from        │
│    "destination":   "Warehouse-Dallas-TX",   ← Where it's going          │
│    "currentHolder": "TransporterMSP",        ← Who has it RIGHT NOW      │
│    "participants":  ["ManufacturerMSP",      ← Who is ALLOWED to touch it│
│                      "TransporterMSP",                                    │
│                      "WarehouseMSP",                                      │
│                      "RetailerMSP",                                       │
│                      "RecipientMSP"],                                     │
│    "dataHash":      "a1b2c3d4e5f6...",       ← SHA-256 fingerprint       │
│    "createdAt":     "2026-04-10T08:00:00Z",  ← Birth timestamp           │
│    "updatedAt":     "2026-04-11T14:30:00Z"   ← Last modified             │
│  }                                                                       │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────── SHIPMENT EVENT ────────────────────────────┐
│                                                                          │
│  {                                                                       │
│    "docType":   "shipmentEvent",                                         │
│    "eventType": "CUSTODY_TRANSFER",          ← What happened             │
│    "actor":     "ManufacturerMSP",           ← Who did it                │
│    "location":  "Phoenix Loading Dock",      ← Where it happened         │
│    "timestamp": "2026-04-11T09:15:00Z",      ← When it happened          │
│    "notes":     "Custody transferred to      ← Human-readable notes      │
│                  TransporterMSP"                                          │
│  }                                                                       │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### Smart Contract Functions — What Each One Does

```
┌──────────────────────────────────────────────────────────────────────────┐
│                      SMART CONTRACT FUNCTIONS                            │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  📦 InitLedger()                                                         │
│  └─ Seeds the blockchain with a sample shipment (SHIP-001) for demo      │
│                                                                          │
│  📦 CreateShipment(id, origin, dest, participants, offChainData)         │
│  └─ Registers a new shipment. Caller becomes the initial holder.         │
│     Off-chain data is SHA-256 hashed and stored on-chain.                │
│                                                                          │
│  🔍 GetShipment(id) → Shipment                                          │
│  └─ Returns current shipment details. Only authorized participants.      │
│                                                                          │
│  📊 UpdateShipmentStatus(id, status, location, notes)                    │
│  └─ Changes the status (e.g., "InTransit" → "InWarehouse").             │
│     Only the current holder can update.                                  │
│                                                                          │
│  🤝 TransferCustody(id, newHolder)                                       │
│  └─ Hands the shipment to the next party in the chain.                   │
│     Only the current holder can transfer. New holder must be authorized. │
│                                                                          │
│  ✅ VerifyShipment(id, offChainData) → bool                              │
│  └─ Checks if data has been tampered with by comparing SHA-256 hashes.   │
│                                                                          │
│  📜 GetShipmentHistory(id) → [Events]                                    │
│  └─ Returns the complete timeline of everything that happened.           │
│                                                                          │
│  👤 AuthorizeParticipant(id, participant)                                │
│  └─ Adds a new party to the authorized list. Only current holder can.    │
│                                                                          │
│  🚫 RevokeParticipant(id, participant)                                   │
│  └─ Removes a party from the authorized list. Cannot revoke yourself.    │
│                                                                          │
│  📋 GetAllShipments() → [Shipments]                                      │
│  └─ Returns all shipments on the ledger.                                 │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### Access Control — Who Can Do What?

Security is built into every function. Here is our access control model:

```
┌──────────────────────────────────────────────────────────────────────────┐
│                        ACCESS CONTROL MATRIX                             │
├────────────────────────┬─────────────────────────────────────────────────┤
│  Action                │  Who is allowed?                                │
├────────────────────────┼─────────────────────────────────────────────────┤
│  Create a shipment     │  Any authenticated organization                 │
│  Read a shipment       │  Only authorized participants of that shipment  │
│  Update status         │  Only the CURRENT HOLDER                        │
│  Transfer custody      │  Only the CURRENT HOLDER                        │
│  Verify data           │  Any authenticated organization                 │
│  View history          │  Any authenticated organization                 │
│  Authorize participant │  Only the CURRENT HOLDER                        │
│  Revoke participant    │  Only the CURRENT HOLDER (cannot revoke self)   │
├────────────────────────┼─────────────────────────────────────────────────┤
│  HOW IT WORKS:         │  Every transaction includes the caller's MSP    │
│                        │  ID (digital identity). The chaincode checks    │
│                        │  this ID against the shipment's participant     │
│                        │  list and current holder field BEFORE allowing  │
│                        │  any action.                                    │
└────────────────────────┴─────────────────────────────────────────────────┘
```

### SHA-256 Hashing — How Data Integrity Works

```
                    OFF-CHAIN DATA VERIFICATION
                    ═══════════════════════════

  AT CREATION TIME:
  ─────────────────
  Off-chain data: "weight:25kg,volume:1cbm,count:50"
                            │
                            ▼
                  ┌──────────────────┐
                  │   SHA-256 Hash   │
                  │   Function       │
                  └────────┬─────────┘
                           │
                           ▼
  Hash: "7f83b1657ff1fc53b92dc18148a1d65dfc2d4b1fa3d677284addd200126d9069"
                           │
                           ▼
              Stored on blockchain (immutable)


  AT VERIFICATION TIME:
  ─────────────────────
  Someone provides: "weight:25kg,volume:1cbm,count:50"
                            │
                            ▼
                  ┌──────────────────┐
                  │   SHA-256 Hash   │
                  │   Function       │
                  └────────┬─────────┘
                           │
                           ▼
  Computed hash: "7f83b1657ff1fc53b92dc18148a1d65dfc2d4b1fa3d677284addd200126d9069"

  Compare: computed hash == on-chain hash ?
           ✅ YES → Data is authentic, not tampered with
           ❌ NO  → Data has been changed — ALERT!
```

**Why is this powerful?** Even changing a single character in the off-chain data (e.g., "weight:25kg" → "weight:26kg") produces a completely different hash. It is computationally impossible to find two different inputs that produce the same hash. This is called **collision resistance**.

---

## 8. Workflow Walkthrough — A Day in the Life of a Shipment

Let's follow `SHIP-042` through its entire journey as a **complete flowchart**:

```
                    ╔═══════════════════════════════╗
                    ║  SHIPMENT LIFECYCLE FLOWCHART ║
                    ╚═══════════════════╦═══════════╝
                                        ║
                                        ▼
                    ┌───────────────────────────────────┐
                    │  1. MANUFACTURER creates shipment │
                    │     ID: SHIP-042                  │
                    │     From: Phoenix, AZ             │
                    │     To: Dallas, TX                │
                    │     Status: "Created"             │
                    └────────────────┬──────────────────┘
                                     │
                                     ▼
                    ┌───────────────────────────────────┐
                    │  2. MANUFACTURER authorizes all   │
                    │     supply chain participants     │
                    │     [Transporter, Warehouse,      │
                    │      Retailer, Recipient]         │
                    └────────────────┬──────────────────┘
                                     │
                                     ▼
                    ┌───────────────────────────────────┐
                    │  3. MANUFACTURER transfers        │
                    │     custody → TRANSPORTER         │
                    │     Status: "InTransit"           │
                    │     📝 Event recorded on-chain    │
                    └────────────────┬──────────────────┘
                                     │
                                     ▼
                    ┌───────────────────────────────────┐
                    │  4. TRANSPORTER updates status    │
                    │     Location: "Highway I-10"      │
                    │     Notes: "On schedule"          │
                    └────────────────┬──────────────────┘
                                     │
                                     ▼
                    ┌───────────────────────────────────┐
                    │  5. TRANSPORTER transfers         │
                    │     custody → WAREHOUSE           │
                    │     Status: "InTransit"           │
                    │     📝 Event recorded on-chain    │
                    └────────────────┬──────────────────┘
                                     │
                                     ▼
                    ┌───────────────────────────────────┐
                    │  6. WAREHOUSE updates status      │
                    │     Status: "InWarehouse"         │
                    │     Location: "Dallas Facility"   │
                    └────────────────┬──────────────────┘
                                     │
                                     ▼
                    ┌───────────────────────────────────┐
                    │  7. WAREHOUSE transfers           │
                    │     custody → RETAILER            │
                    │     📝 Event recorded on-chain    │
                    └────────────────┬──────────────────┘
                                     │
                                     ▼
                    ┌───────────────────────────────────┐
                    │  8. RETAILER updates status       │
                    │     Status: "WithRetailer"        │
                    │     Location: "Store #205"        │
                    └────────────────┬──────────────────┘
                                     │
                                     ▼
                    ┌───────────────────────────────────┐
                    │  9. RETAILER transfers            │
                    │     custody → RECIPIENT           │
                    │     📝 Event recorded on-chain    │
                    └────────────────┬──────────────────┘
                                     │
                                     ▼
                    ┌───────────────────────────────────┐
                    │  10. RECIPIENT updates status     │
                    │      Status: "Delivered" ✅       │
                    │      📝 Final event recorded      │
                    │                                   │
                    │  ⛔ Shipment is now LOCKED         │
                    │  No more transfers or updates     │
                    │  allowed — the journey is over.   │
                    └────────────────┬──────────────────┘
                                     │
                                     ▼
                    ┌───────────────────────────────────┐
                    │  11. ANYONE can verify:           │
                    │      VerifyShipment() → ✅/❌     │
                    │      GetShipmentHistory() → 📜    │
                    │                                   │
                    │  The complete, tamper-proof        │
                    │  history lives forever on the      │
                    │  blockchain.                       │
                    └───────────────────────────────────┘
```

### The Sequence of Events on the Blockchain

```
TIME ──────────────────────────────────────────────────────────────────▶

  Manufacturer         Transporter          Warehouse          Retailer           Recipient
      │                     │                    │                  │                   │
      │── CreateShipment ──▶│                    │                  │                   │
      │                     │                    │                  │                   │
      │── TransferCustody ─▶│                    │                  │                   │
      │    (Mfg → Trans)    │                    │                  │                   │
      │                     │── TransferCustody ▶│                  │                   │
      │                     │   (Trans → Whs)    │                  │                   │
      │                     │                    │── TransferCustody│                   │
      │                     │                    │  (Whs → Retail)  │                   │
      │                     │                    │                  │── TransferCustody │
      │                     │                    │                  │  (Retail → Recip) │
      │                     │                    │                  │                   │
      │                     │                    │                  │   UpdateStatus    │
      │                     │                    │                  │   ("Delivered")   │
      │                     │                    │                  │                   │
      ▼                     ▼                    ▼                  ▼                   ▼
  ┌───────────────────────────────────────────────────────────────────────────────────────┐
  │                          IMMUTABLE BLOCKCHAIN LEDGER                                  │
  │   Block 1    Block 2    Block 3    Block 4    Block 5    Block 6    Block 7           │
  │   [Create]   [Transfer] [Transfer] [Transfer] [Transfer] [Deliver]  ...              │
  │                                                                                       │
  │   Every block is linked to the previous one. Changing any block invalidates           │
  │   everything after it. This is what makes the record TAMPER-PROOF.                    │
  └───────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 9. Testing & Verification

We wrote **15 unit tests** to verify every aspect of the smart contract. These tests use Fabric's `MockStub` — a simulated blockchain environment that lets us test the chaincode without spinning up the entire network.

### Test Results

```
┌──────────────────────────────────────────────────────────────────────────┐
│                        UNIT TEST RESULTS                                 │
│                        All 15 / 15 PASSED ✅                             │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ✅ TestInitLedger              — Ledger initializes with sample data    │
│  ✅ TestCreateShipment          — New shipment is created correctly      │
│  ✅ TestCreateShipmentDuplicate — Duplicate IDs are rejected             │
│  ✅ TestUpdateShipmentStatus    — Status updates work correctly          │
│  ✅ TestUpdateShipmentStatusInvalid — Invalid statuses are rejected      │
│  ✅ TestTransferCustody         — Custody transfers work correctly       │
│  ✅ TestTransferCustodyUnauthorized — Unauthorized transfers blocked     │
│  ✅ TestVerifyShipmentValid     — Authentic data passes verification     │
│  ✅ TestVerifyShipmentInvalid   — Tampered data is detected              │
│  ✅ TestAuthorizeParticipant    — New participants can be authorized      │
│  ✅ TestRevokeParticipant       — Participants can be revoked            │
│  ✅ TestFullLifecycle           — Complete shipment journey works         │
│  ✅ TestDeliveredCannotBeUpdated — Delivered shipments are locked         │
│  ✅ TestGetAllShipments         — All shipments are retrievable           │
│  ✅ TestComputeHash             — SHA-256 hashing works correctly         │
│                                                                          │
│  Total time: 0.437 seconds                                               │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### What Do the Key Tests Prove?

```
  Test: "TestFullLifecycle"
  ════════════════════════
  This single test simulates the ENTIRE journey of a shipment:

  Step 1: Initialize the ledger        ──▶ ✅
  Step 2: Create a new shipment        ──▶ ✅
  Step 3: Transfer to Transporter      ──▶ ✅
  Step 4: Transfer to Warehouse        ──▶ ✅
  Step 5: Update status to InWarehouse ──▶ ✅
  Step 6: Transfer to Retailer         ──▶ ✅
  Step 7: Mark as Delivered            ──▶ ✅
  Step 8: Try to update after delivery ──▶ ✅ (correctly REJECTED)
  Step 9: Try to transfer after delivery ─▶ ✅ (correctly REJECTED)

  This proves our entire business logic works end-to-end.
```

---

## 10. Challenges & Limitations

### Challenges We Faced

```
┌──────────────────────────────────────────────────────────────────────────┐
│                           CHALLENGES                                     │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  1. 🔧 MOCK IDENTITY IN TESTS                                           │
│     Problem: Fabric's MockStub doesn't provide a client identity,        │
│     so our access-control functions crashed with nil pointer errors.     │
│     Solution: We created a mock identity using X.509 certificates        │
│     and protobuf serialization to simulate real Fabric identities.       │
│                                                                          │
│  2. 🔧 GO MODULE DEPENDENCIES                                           │
│     Problem: Hyperledger Fabric's Go libraries have complex              │
│     dependency trees with many transitive dependencies.                  │
│     Solution: Careful go.mod management and running `go mod tidy`        │
│     to resolve all indirect dependencies automatically.                  │
│                                                                          │
│  3. 🔧 NETWORK CONFIGURATION                                            │
│     Problem: Setting up a multi-organization Fabric network requires     │
│     Docker Compose files, crypto config, channel config, and shell       │
│     scripts that must all reference the same organization names,         │
│     ports, and certificate paths consistently.                           │
│     Solution: Created a modular script-based approach with               │
│     environment variable helpers for organization switching.             │
│                                                                          │
│  4. 🔧 OFF-CHAIN DATA MANAGEMENT                                        │
│     Problem: Storing large data (images, PDFs, full manifests)           │
│     directly on the blockchain is expensive and slow.                    │
│     Solution: Store only the SHA-256 hash on-chain. The actual data      │
│     lives off-chain (e.g., IPFS, encrypted database). Anyone can         │
│     verify integrity by recomputing the hash.                            │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### Current Limitations

| Limitation | Description | Possible Future Improvement |
|------------|-------------|----------------------------|
| **Two organizations only** | Our network currently has Manufacturer and Transporter peers. Warehouse, Retailer, and Recipient exist as MSP identities but don't have their own dedicated peer nodes. | Add peer nodes for all five stakeholders. |
| **No GUI / Frontend** | Interaction is through REST API or CLI — no visual dashboard. | Build a React/Next.js web dashboard. |
| **Single orderer** | One Raft orderer — no crash fault tolerance in this demo. | Deploy 3 or 5 orderers for production. |
| **No IPFS integration** | Off-chain data hashing is implemented, but we don't have an actual off-chain storage system. | Integrate IPFS for decentralized file storage. |
| **No pagination** | `GetAllShipments()` returns everything at once — not scalable for millions of records. | Add bookmark-based pagination. |

---

## 11. Conclusion

This project demonstrates how blockchain technology — specifically **Hyperledger Fabric** — can transform supply chain management from a fragmented, trust-dependent process into a transparent, tamper-proof, and auditable system.

**What we built:**
- A **smart contract** (647 lines of Go) that enforces business rules automatically — no human middleman needed
- A **permissioned blockchain network** with Docker containers, Certificate Authorities, and Raft consensus
- A **REST API** that makes blockchain interaction as simple as calling an HTTP endpoint
- A **comprehensive test suite** (15 tests, all passing) that proves our logic works correctly

**The core insight:**
> In traditional supply chains, trust is a human problem — "Do I believe what the warehouse says?" In our system, trust is a **mathematical guarantee** — the blockchain provides cryptographic proof that every record is authentic, untampered, and agreed upon by all parties.

Every custody transfer, every status update, every participant authorization — all permanently recorded, timestamped, and digitally signed on a shared ledger that no single party controls.

**This is the promise of blockchain for enterprise: not cryptocurrency, but *provable truth*.**

---

## 12. References

1. D. Shakhbulatov, J. Medina, Z. Dong and R. Rojas-Cessa, "How Blockchain Enhances Supply Chain Management: A Survey," in *IEEE Open Journal of the Computer Society*, vol. 1, pp. 230-249, 2020.

2. S. Oğuz, G. Alkan, B. Yilmaz and C. Kocabaş, "The Use of Blockchain Technology in Logistics and Supply Chain Management (SCM): A Systematic Review," in *IEEE Access*, vol. 12, pp. 166211-166224, 2024.

3. P. Gonczol, P. Katsikouli, L. Herskind and N. Dragoni, "Blockchain Implementations and Use Cases for Supply Chains — A Survey," in *IEEE Access*, vol. 8, pp. 11856-11871, 2020.

4. Hyperledger Fabric Documentation — https://hyperledger-fabric.readthedocs.io/

5. Hyperledger Fabric Contract API for Go — https://pkg.go.dev/github.com/hyperledger/fabric-contract-api-go

6. Docker Documentation — https://docs.docker.com/

---

*Report prepared for CSE 540 – Engineering Blockchain Applications, Arizona State University, Spring B 2026.*
