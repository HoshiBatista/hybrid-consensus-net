# Architecture — Hybrid Consensus Network

## Table of Contents

- [Overview](#overview)
- [Package Responsibilities](#package-responsibilities)
- [Core Interfaces](#core-interfaces)
- [Block Lifecycle](#block-lifecycle)
- [Consensus Internals](#consensus-internals)
- [P2P Layer](#p2p-layer)
- [State Model](#state-model)
- [Persistence](#persistence)
- [REST API](#rest-api)
- [Design Decisions](#design-decisions)

---

## Overview

The node is a single binary (`cmd/node/main.go`) that wires four independent subsystems:

```
┌─────────────────────────────────────────────────────────────┐
│                       Node Process                          │
│                                                             │
│  ┌──────────┐   ┌────────────┐   ┌──────────┐  ┌─────────┐  │
│  │  REST    │   │ Blockchain │   │Consensus │  │  P2P    │  │
│  │  API     │──►│  + State   │◄──│ PoW/PoS  │  │libp2p   │  │
│  │  (Gin)   │   │  + Mempool │   │          │  │GossipSub│  │
│  └──────────┘   └─────┬──────┘   └──────────┘  └───┬─────┘  │
│                       │   bbolt                    │        │
│                       └────────────────────────────┘        │
└─────────────────────────────────────────────────────────────┘
```

Each subsystem is isolated behind a Go interface or a concrete struct. Dependency injection in `main.go` is the only place where concrete types are named.

---

## Package Responsibilities

### `internal/blockchain`

| File | Responsibility |
|---|---|
| `models.go` | `Block` and `Transaction` structs; SHA-256 hash calculation |
| `blockchain.go` | Chain persistence and retrieval via bbolt; `Tip` pointer management |
| `block.go` | `NewBlock` constructor |
| `transaction.go` | `NewTransaction`, ECDSA `Sign`, `Verify` |
| `state.go` | In-memory account balance ledger (`Credit`, `Debit`, `Balance`, `TotalStake`) |
| `mempool.go` | Pending transaction pool with balance pre-validation (`Add`, `Flush`, `Pending`, `Size`) |

### `internal/consensus`

| File | Responsibility |
|---|---|
| `interface.go` | `Consensus` interface — the only contract the blockchain core depends on |
| `pow.go` | PoW: nonce iteration until `sha256(block)` meets difficulty target |
| `pos.go` | PoS: stake-weighted random validator selection |

### `internal/network`

| File | Responsibility |
|---|---|
| `node.go` | libp2p host, GossipSub router, mDNS discovery, `BroadcastBlock`, `BlockStream` |

### `internal/api`

| File | Responsibility |
|---|---|
| `server.go` | Gin HTTP server; registers 8 REST routes; CORS middleware; runtime consensus switch |

### `cmd/node`

| File | Responsibility |
|---|---|
| `main.go` | Flag parsing, construction of all components, wiring, blocking `srv.Run` |

---

## Core Interfaces

### `consensus.Consensus`

```go
type Consensus interface {
    MineBlock(block *blockchain.Block) error
    ValidateBlock(block *blockchain.Block) bool
    Type() string
}
```

Any new consensus mechanism implements these three methods. The `blockchain` and `api` packages depend only on this interface — never on `PoW` or `PoS` directly.

---

## Block Lifecycle

```
1. Client submits tx → POST /transactions
         │
         ▼
2. blockchain.NewTransaction(sender, recipient, amount)
   mempool.Add(tx, state)   ← checks sender balance
         │
         ▼
3. Client triggers → POST /mine
         │
         ▼
4. mempool.Flush(0)          ← drain all pending txs
   blockchain.NewBlock(txs, tip, height)
         │
         ▼
5. consensus.MineBlock(block)
   ┌─────────────────────┬──────────────────────┐
   │ PoW                 │ PoS                  │
   │ iterate Nonce until │ stake-weighted rand  │
   │ hash meets target   │ selects Validator    │
   └─────────────────────┴──────────────────────┘
         │
         ▼
6. blockchain.AddBlock(block) → bbolt bucket "blocks"
         │
         ▼
7. state.Debit(sender) / state.Credit(recipient)
         │
         ▼
8. node.BroadcastBlock(ctx, block) → GossipSub → peers
```

---

## Consensus Internals

### Proof of Work

```
target = 1 << (256 - difficulty*4)   // difficulty nibbles of leading zeros

loop:
    block.Nonce++
    hash = sha256(PrevHash || TxHash || Timestamp || Height || Nonce)
    if hash < target → break
```

- Difficulty `d` means the first `d` hex characters of the hash must be `0`.
- Default difficulty is `4` (16^4 = 65 536 attempts on average).

### Proof of Stake

```
totalStake = sum of all account balances in State

roll = rand.Intn(totalStake)

for each (address, balance) in State:
    if roll < balance → this address is the validator; break
    roll -= balance
```

- Probability of selection is proportional to balance.
- No hashing loop — block production is instantaneous.
- Requires `state.TotalStake() > 0` to avoid a zero-division panic.

---

## P2P Layer

```
libp2p Host
   │
   ├── TCP transport (default)
   ├── TLS 1.3 security (automatic)
   ├── Yamux stream multiplexer
   │
   ├── GossipSub router
   │   └── Topic: "hybrid-blockchain/blocks/v1"
   │       ├── WithFloodPublish(true) — bypasses mesh for small networks
   │       └── Subscription — BlockStream() goroutine
   │
   └── mDNS service (tag: "hybrid-blockchain-mdns")
       └── HandlePeerFound → host.Connect() on LAN discovery
```

**Why FloodPublish?** GossipSub's default mesh requires ≥ 6 peers to stabilise. For academic/demo networks with 2–5 nodes, flood publish ensures every directly-connected peer receives every block regardless of mesh state.

**Self-loop prevention**: `BlockStream()` filters messages where `msg.ReceivedFrom == n.Host.ID()`, so a node never re-processes its own broadcasts.

---

## State Model

The account model is **in-memory** and rebuilt from transaction history on restart (current implementation) or can be persisted independently.

```
State map[string]uint64
   address (hex string) → balance (uint64 tokens)
```

Key operations:

| Method | Behaviour |
|---|---|
| `Credit(addr, amount)` | Adds `amount` to balance; creates account if absent |
| `Debit(addr, amount)` | Subtracts `amount`; returns error if insufficient funds |
| `Balance(addr)` | Returns current balance (0 if address unknown) |
| `TotalStake()` | Sum of all balances — used by PoS |

---

## Persistence

bbolt stores blocks in a single bucket named `"blocks"`. The key is the block `Hash` (bytes); the value is JSON-encoded `Block`.

```
bbolt file: blockchain.db
└── bucket "blocks"
    ├── key: <block-0-hash> → value: JSON(genesis block)
    ├── key: <block-1-hash> → value: JSON(block 1)
    └── ...
```

The chain `Tip` (hash of the latest block) is stored in a separate bucket key `"tip"`. On startup, `CreateBlockchain` reads `Tip` and, if absent, writes the genesis block.

---

## REST API

The Gin router is configured with a global CORS middleware (Allow-Origin: `*`) to support browser-based dashboards from any origin.

```
GET  /status           → consensus type, difficulty, height, block count, pending txs
GET  /blocks           → []Block (all blocks, oldest first)
GET  /mempool          → { count, transactions[] }
GET  /balance/:address → { address, balance }
POST /transactions     → add tx to mempool; returns tx_id
POST /mine             → mine block; returns the new Block
POST /consensus        → switch PoW/PoS at runtime
POST /faucet           → credit tokens (dev only)
GET  /                 → frontend/index.html
```

The `mu sync.RWMutex` inside `Server` guards only `cs` (active consensus) and `powDiff`. Blockchain and state operations are not currently locked at the API layer (acceptable for a single-node academic demo).

---

## Design Decisions

| Decision | Rationale |
|---|---|
| In-memory State | Simplicity for academic context; easily replaced with a bbolt-backed `Store` interface |
| Gin over stdlib `net/http` | Param routing, JSON binding, and middleware with minimal boilerplate |
| GossipSub over raw TCP | Future-proof: scales to large networks; handles fan-out automatically |
| mDNS discovery | Zero configuration for LAN demos; a DHT bootstrap layer can be added for WAN |
| FloodPublish enabled | Guarantees delivery in small (< 6 node) networks where GossipSub mesh won't form |
| bbolt over PostgreSQL | Embedded, no external process; suitable for a single-node or small cluster demo |
| `Consensus` interface | Allows PoW/PoS swap without touching blockchain or API code |
| JSON-encoded blocks on disk | Human-readable for debugging; easy to inspect with `jq` |
