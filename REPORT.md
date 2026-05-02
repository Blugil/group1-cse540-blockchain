# 📦 Blockchain-Based Shipment Tracking System
### **CSE 540 – Engineering Blockchain Applications | Spring B 2026 | Group 1**
**Dominick Agnello · Ritish Abrol · Vatsal Patel · Shashikant Nanda · Anushree Bhure**

---

## Abstract

Modern supply chains suffer from fragmented, siloed record-keeping that creates fertile ground for fraud, disputes, and untraceable losses. This report presents a **Blockchain-Based Shipment Tracking System** built on **Hyperledger Fabric 2.5**, a permissioned enterprise blockchain. Our system provides an immutable, shared chain-of-custody ledger that records every shipment handoff, status update, and participant authorization with cryptographic proof. The smart contract, written in Go, implements ten chaincode functions covering full lifecycle management — from shipment creation through final delivery — with role-based access control and SHA-256 off-chain data integrity verification. The system is deployed as a two-organization Fabric network (ManufacturerMSP, TransporterMSP) with Raft consensus, CouchDB state storage, and a Node.js/Express REST API. All 15 unit tests pass. The system demonstrates how blockchain transforms supply chain trust from a human problem into a mathematical guarantee.

---

## Table of Contents

1. [Introduction — The Story](#1-introduction--the-story)
2. [Problem Statement — What Goes Wrong Today](#2-problem-statement--what-goes-wrong-today)
3. [Prior Work — Literature Review](#3-prior-work--literature-review)
4. [Why Blockchain? — A Simple Explanation](#4-why-blockchain--a-simple-explanation)
5. [How Our System Works — Step by Step](#5-how-our-system-works--step-by-step)
6. [System Architecture — The Big Picture](#6-system-architecture--the-big-picture)
7. [Technologies Used](#7-technologies-used)
8. [Smart Contract Design — The Rules of the Game](#8-smart-contract-design--the-rules-of-the-game)
9. [Workflow Walkthrough — A Day in the Life of a Shipment](#9-workflow-walkthrough--a-day-in-the-life-of-a-shipment)
10. [Testing & Verification](#10-testing--verification)
11. [Analysis](#11-analysis)
12. [Challenges & Limitations](#12-challenges--limitations)
13. [Innovation & Impact](#13-innovation--impact)
14. [Future Work](#14-future-work)
15. [Contributions](#15-contributions)
16. [Conclusion](#16-conclusion)
17. [References](#17-references)

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

## 3. Prior Work — Literature Review

Blockchain for supply chain management has been an active research area since 2016. This section summarizes the most relevant prior work and identifies the gaps our system addresses.

**Shakhbulatov et al. (2020)** [1] conducted a comprehensive survey of blockchain in supply chain, identifying five core value propositions: traceability, transparency, immutability, disintermediation, and smart contract automation. Their survey found that **Hyperledger Fabric** was the most widely adopted permissioned blockchain for enterprise supply chains due to its modular architecture and identity management. However, they noted that most academic prototypes stopped at smart contract design without deploying a live, end-to-end network — a gap our project directly addresses.

**Oğuz et al. (2024)** [2] performed a systematic review of 87 studies on blockchain in logistics and SCM. They found that the dominant use cases were pharmaceutical traceability (40%), food safety (30%), and general goods (30%). Critically, they identified **access control** and **off-chain data management** as the two most underaddressed technical challenges. Our system directly tackles both: we implement fine-grained RBAC at the chaincode level and use SHA-256 hashing to link off-chain data to on-chain records without storing sensitive data on the blockchain.

**Gonczol et al. (2020)** [3] analyzed 34 real-world blockchain supply chain implementations and found that none achieved full decentralization across all five stakeholder tiers in a single system. Most deployed blockchain for 2–3 actors and relied on off-chain coordination for the rest. Our architecture mirrors this pragmatic finding — we implement full on-chain logic for all five stakeholder MSP roles while running dedicated peer nodes for the two primary actors (Manufacturer, Transporter), consistent with industry practice.

**Key gap addressed by our work:** Existing literature largely presents system designs and theoretical analyses. Our contribution is a **fully deployed, live Hyperledger Fabric 2.5 network** with on-chain smart contract logic verified by 15 passing unit tests and accessible via a REST API — bridging the gap between academic design and functional implementation.

---

## 4. Why Blockchain? — A Simple Explanation

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

## 5. How Our System Works — Step by Step

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

## 6. System Architecture — The Big Picture

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

## 7. Technologies Used

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

## 8. Smart Contract Design — The Rules of the Game

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

## 9. Workflow Walkthrough — A Day in the Life of a Shipment

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

## 10. Testing & Verification

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

## 11. Analysis

### Scalability

Hyperledger Fabric's architecture separates transaction execution (endorsement) from ordering and validation, enabling significantly higher throughput than public blockchains. In benchmarks, Fabric achieves **2,000–3,000+ TPS** under optimal conditions [4], compared to Bitcoin (~7 TPS) and Ethereum (~30 TPS). For supply chain applications, even a large global retailer (e.g., Walmart) processes approximately 1 million transactions per day — roughly **12 TPS** — well within Fabric's throughput envelope.

Our current deployment uses a **single orderer node**, which is a single point of failure. For production, a Raft cluster of 3 or 5 orderers would be required. CouchDB enables rich JSON queries against the world state, but `GetAllShipments()` performs a full-table scan — pagination via CouchDB bookmarks would be required at scale (>100,000 shipments).

### Gas Cost / Transaction Cost

Unlike Ethereum, Hyperledger Fabric has **no gas fees**. Transaction costs are operational (infrastructure): Docker containers, compute, and storage. For our two-peer network:

| Resource | Estimated Cost (AWS equivalent) |
|----------|--------------------------------|
| 2 Fabric peer nodes (t3.medium) | ~$60/month |
| 1 orderer node (t3.small) | ~$15/month |
| 2 CouchDB instances | ~$30/month |
| **Total** | **~$105/month** |

This is dramatically lower than Ethereum smart contract costs. A comparable Ethereum deployment with 10,000 custody transfers/month at ~21,000 gas each at 30 gwei ≈ **$180–$900/month** in gas alone (at $3,000/ETH), and that ignores infrastructure.

### Data Management

Our system stores two categories of data:
- **On-chain (immutable):** Shipment state (ID, status, participants, current holder, timestamps, SHA-256 hash) — approximately **800 bytes per shipment record**
- **Off-chain (not yet implemented):** Raw metadata (weight, volume, photos, certificates) — referenced only by hash

CouchDB provides the world state (current values) while the Fabric ledger provides the full history. This dual-storage model gives us both fast queries and tamper-proof audit trails. A limitation is that CouchDB is a trusted component — if the CouchDB instance is compromised, query results could be manipulated (though this would be detectable by cross-referencing the ledger hash).

### Privacy & Regulatory Considerations

Hyperledger Fabric supports **private data collections** (PDC) — a feature we did not implement in this prototype but is critical for production. Without PDC, all channel members see all shipment data. In a production deployment:

- **GDPR compliance**: Recipient personal data (delivery address, name) should NOT be stored on-chain. Our system stores only MSP IDs (organizational identities), which are not personally identifiable — a compliant design choice.
- **Trade secrets**: Pricing, supplier relationships, and proprietary routes would use private data collections or be stored only as off-chain hashes.
- **Regulatory audit**: The immutable ledger provides a built-in audit trail satisfying requirements from FDA (drug tracking), USDA (food safety), and customs authorities — without requiring a separate audit system.

### Comparison to Traditional (Non-Blockchain) Approaches

| Dimension | Traditional EDI/Database | Our Blockchain System |
|-----------|--------------------------|----------------------|
| **Trust model** | Trust the system owner | Cryptographic proof — no trust required |
| **Dispute resolution** | "He said / she said" — manual investigation | Query `GetShipmentHistory()` — immutable proof in <1 second |
| **Data integrity** | Admin can modify records | Tamper-evident — any change breaks the chain |
| **Audit trail** | Optional, company-controlled | Always-on, append-only, cryptographically signed |
| **Multi-party coordination** | Requires EDI agreements, APIs, and integration work | All parties transact on the same shared ledger |
| **Setup complexity** | Low (centralized DB) | Higher (Fabric network setup) |
| **Operational cost** | Low | Medium (infrastructure + ops) |
| **Single point of failure** | Yes (central DB) | No (distributed across peers) |
| **Throughput** | Very high (millions TPS) | High (3,000+ TPS for Fabric) |

The blockchain approach trades simplicity and raw performance for **trust, transparency, and auditability** — a worthwhile trade-off in supply chains where the cost of fraud and disputes (estimated at $500B+ annually globally) far exceeds infrastructure costs.

---

## 12. Challenges & Limitations

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
│  5. 🔧 CRYPTOGEN vs. FABRIC CA IDENTITY                                 │
│     Problem: The Fabric SDK's `enrollAdmin` / `registerUser` scripts     │
│     enroll against the Fabric CA, but the network was bootstrapped       │
│     with cryptogen — so CA-issued certificates are not in the peer       │
│     MSP trusted roots, causing "access denied" errors.                   │
│     Solution: Created `importCryptoIdentity.js` which directly loads     │
│     the cryptogen-generated Admin and User1 X.509 certificates into      │
│     the SDK wallet. These are already trusted by the peers since they    │
│     come from the same root CA used during network bootstrap.            │
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

## 13. Innovation & Impact

### What Differentiates This System

Most academic blockchain supply chain prototypes stop at chaincode design or a simulated environment. Our system makes four contributions that distinguish it from prior work:

| Innovation | Description |
|------------|-------------|
| **Live end-to-end deployment** | A fully operational Hyperledger Fabric 2.5 network — not a simulation. Two peer organizations, Raft orderer, CouchDB, and a REST API all running on Docker, with real transactions committed to an immutable ledger. |
| **Per-shipment dynamic RBAC** | Access control is encoded in chaincode at the individual shipment level, not at the network policy level. The current holder can authorize or revoke a participant (e.g., a customs auditor) for one shipment without any network reconfiguration. |
| **Chaincode-enforced terminal-state immutability** | A `Delivered` shipment cannot be mutated by any caller regardless of identity — the guard runs before any ledger read or write, making bypass via SDK manipulation impossible. |
| **SHA-256 off-chain integrity bridge** | `VerifyShipment()` provides a storage-agnostic link between on-chain records and any off-chain data (relational DB, IPFS, object store) — verification works as long as the hash matches, independent of where the raw data lives. |

### Who Benefits

| Stakeholder | Benefit |
|-------------|---------|
| **Manufacturers** | Cryptographic proof of dispatch — eliminates "we never received it" disputes |
| **Transporters** | Signed chain-of-custody at each handoff — liability clearly scoped to their custody window |
| **Regulators / Auditors** | Immutable, court-admissible audit trail with timestamp and digital signature on every event — no separate audit system required |
| **Recipients / Consumers** | Verifiable provenance — can confirm goods were not tampered with or substituted in transit |
| **Insurance companies** | Precise loss attribution — `GetShipmentHistory()` identifies exact custody period of loss or damage |

### Target Industries & Regulatory Fit

- **Pharmaceuticals**: US Drug Supply Chain Security Act (DSCSA) requires serialized traceability for all prescription drugs. Our immutable custody records provide a DSCSA-compliant backbone.
- **Food safety**: FDA Food Safety Modernization Act (FSMA) requires contamination-source identification within 24 hours. Our composite-key history provides instant, ledger-verifiable traceability.
- **Luxury goods**: Immutable provenance records prevent counterfeiting even by a compromised seller — every ownership transfer is cryptographically signed.
- **General logistics**: Any multi-party custody chain benefits from the `GetShipmentHistory()` dispute resolution function.

### Real-World Adoption Challenges

Our prototype proves the technology works. Moving to production introduces challenges beyond the technical:

| Challenge | Description |
|-----------|-------------|
| **Organizational on-boarding** | Each new organization must provision cryptographic key material, configure an MSP, run a Fabric peer, and integrate the REST API into existing ERP/WMS systems — a significant barrier for small suppliers |
| **Governance & consortium management** | Channel config changes (adding an org, upgrading chaincode) require majority signatures from all channel members; without a formal governance framework, protocol stagnation is possible |
| **Certificate lifecycle management** | X.509 certificates expire; production deployments require automated rotation, CRL management, and HSM-backed key storage — none implemented in this prototype |
| **Legacy system interoperability** | Most existing logistics platforms use EDI X12 / EDIFACT; bridging these formats to Fabric transactions requires transformation middleware not yet built |

These challenges are fundamentally sociotechnical, not cryptographic. The consensus and smart contract mechanisms are solved; organizational coordination and total cost of ownership remain the primary adoption barriers.

---

## 14. Future Work

While our system demonstrates a fully functional end-to-end blockchain shipment tracking solution, several enhancements would move it toward production readiness:

| Enhancement | Description | Priority |
|-------------|-------------|----------|
| **Additional peer organizations** | Add dedicated peer nodes for WarehouseMSP, RetailerMSP, and RecipientMSP so all five stakeholders have on-chain representation | High |
| **Private data collections (PDC)** | Use Fabric's PDC feature to allow selective data sharing — e.g., pricing visible only to Manufacturer + Transporter | High |
| **IPFS integration** | Store documents, images, and certificates on IPFS; store only the IPFS content hash on-chain | High |
| **Multi-orderer Raft cluster** | Deploy 3 or 5 orderer nodes for crash fault tolerance in production | High |
| **React/Next.js web dashboard** | Visual UI showing shipment map, status timeline, and custody chain for non-technical stakeholders | Medium |
| **IoT sensor integration** | Accept real-time temperature, humidity, and GPS telemetry from IoT devices as signed status updates | Medium |
| **CouchDB pagination** | Implement bookmark-based pagination in `GetAllShipments()` for scalability beyond 100,000 records | Medium |
| **Cross-channel interoperability** | Use Fabric's inter-channel communication or a relay bridge to connect separate supply chain networks | Low |
| **Automated dispute resolution** | Smart contract logic that auto-flags discrepancies (e.g., declared weight vs. IoT-measured weight) and triggers alerts | Low |

---

## 15. Contributions

All team members contributed meaningfully across design, implementation, and documentation phases.

| Team Member | Primary Contributions |
|-------------|----------------------|
| **Dominick Agnello** | Network architecture design; Docker Compose configuration; `network.sh` lifecycle scripting; peer and orderer setup |
| **Ritish Abrol** | Smart contract function design (`TransferCustody`, `AuthorizeParticipant`, `RevokeParticipant`); access control matrix; chaincode unit tests |
| **Vatsal Patel** | Chaincode implementation (`CreateShipment`, `UpdateShipmentStatus`, `GetShipmentHistory`); SHA-256 hashing logic; `InitLedger` seeding |
| **Shashikant Nanda** | Node.js REST API (`app.js`); Fabric SDK integration; wallet management (`enrollAdmin.js`, `importCryptoIdentity.js`); end-to-end test workflow |
| **Anushree Bhure** | Technical report writing; system architecture diagrams; literature review; analysis section; presentation preparation |

All members participated in system integration testing, debugging, and final demo preparation.

---

## 16. Conclusion

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

## 17. References

1. D. Shakhbulatov, J. Medina, Z. Dong and R. Rojas-Cessa, "How Blockchain Enhances Supply Chain Management: A Survey," in *IEEE Open Journal of the Computer Society*, vol. 1, pp. 230-249, 2020.

2. S. Oğuz, G. Alkan, B. Yilmaz and C. Kocabaş, "The Use of Blockchain Technology in Logistics and Supply Chain Management (SCM): A Systematic Review," in *IEEE Access*, vol. 12, pp. 166211-166224, 2024.

3. P. Gonczol, P. Katsikouli, L. Herskind and N. Dragoni, "Blockchain Implementations and Use Cases for Supply Chains — A Survey," in *IEEE Access*, vol. 8, pp. 11856-11871, 2020.

4. Hyperledger Fabric Documentation — https://hyperledger-fabric.readthedocs.io/

5. Hyperledger Fabric Contract API for Go — https://pkg.go.dev/github.com/hyperledger/fabric-contract-api-go

6. Docker Documentation — https://docs.docker.com/

---

*Report prepared for CSE 540 – Engineering Blockchain Applications, Arizona State University, Spring B 2026.*
