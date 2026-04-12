<div align="center">

# Hybrid Consensus Network

<img src="https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version"/>
<img src="https://img.shields.io/badge/libp2p-v0.48-blueviolet?style=for-the-badge&logo=ipfs&logoColor=white" alt="libp2p"/>
<img src="https://img.shields.io/badge/Consensus-PoW%20%7C%20PoS-orange?style=for-the-badge&logo=bitcoin&logoColor=white" alt="Consensus"/>
<img src="https://img.shields.io/badge/API-REST%20%2B%20WebSocket-green?style=for-the-badge&logo=fastapi&logoColor=white" alt="API"/>
<img src="https://img.shields.io/badge/DB-bbolt-red?style=for-the-badge&logo=databricks&logoColor=white" alt="bbolt"/>
<img src="https://img.shields.io/badge/License-MIT-yellow?style=for-the-badge" alt="License"/>

**A decentralized, peer-to-peer blockchain network implemented in Go,  
featuring hot-swappable consensus (PoW / PoS), a libp2p P2P layer, and a real-time dashboard.**

[Features](#-features) · [Architecture](#-architecture) · [Quick Start](#-quick-start) · [API Reference](#-api-reference) · [Testing](#-testing) · [Contributing](#-contributing)

</div>

---

## Features

| Category | Details |
|---|---|
| **Consensus** | Proof-of-Work (SHA-256, configurable difficulty) and Proof-of-Stake (stake-weighted lottery) — switchable at runtime via REST or flag |
| **P2P Network** | libp2p with GossipSub epidemic broadcast and mDNS zero-config local discovery |
| **Persistence** | bbolt embedded key-value store — no external database needed |
| **REST API** | Gin-powered HTTP API with CORS; 8 endpoints covering chain, mempool, consensus, and faucet |
| **Cryptography** | SHA-256 block hashing, ECDSA transaction signing (secp256k1) |
| **Frontend** | Auto-refreshing HTML dashboard — mine, inspect, and transfer from the browser |
| **Modular Design** | Pluggable `Consensus` interface; swap algorithms without touching core blockchain logic |

---

## Architecture

```
hybrid-consensus-net/
├── cmd/
│   └── node/
│       └── main.go          # Entry point — flag parsing & wiring
├── internal/
│   ├── api/
│   │   └── server.go        # Gin REST server, 8 routes
│   ├── blockchain/
│   │   ├── models.go        # Block & Transaction structs, SHA-256 hashing
│   │   ├── blockchain.go    # Chain persistence via bbolt
│   │   ├── block.go         # Block constructor
│   │   ├── transaction.go   # ECDSA sign / verify
│   │   ├── state.go         # Account balances (in-memory)
│   │   └── mempool.go       # Pending transaction pool
│   ├── consensus/
│   │   ├── interface.go     # Consensus interface (MineBlock / ValidateBlock / Type)
│   │   ├── pow.go           # Proof-of-Work implementation
│   │   └── pos.go           # Proof-of-Stake implementation
│   └── network/
│       └── node.go          # libp2p host, GossipSub, mDNS
└── frontend/
    └── index.html           # Real-time blockchain dashboard
```

### Data Flow

```
Transaction (ECDSA signed)
        │
        ▼
   Mempool.Add()          ← balance check against State
        │
   POST /mine
        │
        ▼
  consensus.MineBlock()   ← PoW (hash with nonce) or PoS (stake lottery)
        │
        ▼
  blockchain.AddBlock()   ← persist to bbolt
        │
        ▼
  State updated (debit sender / credit recipient)
        │
        ▼
  P2P Node.BroadcastBlock() ── GossipSub ──► peers
```

---

## Tech Stack

<div align="center">

![Go](https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white)
![HTML5](https://img.shields.io/badge/HTML5-E34F26?style=flat-square&logo=html5&logoColor=white)
![CSS3](https://img.shields.io/badge/CSS3-1572B6?style=flat-square&logo=css3&logoColor=white)
![Gin](https://img.shields.io/badge/Gin-00ADD8?style=flat-square&logo=go&logoColor=white)
![libp2p](https://img.shields.io/badge/libp2p-65C2CB?style=flat-square&logo=ipfs&logoColor=white)
![bbolt](https://img.shields.io/badge/bbolt-CC2200?style=flat-square&logo=databricks&logoColor=white)
![WebSocket](https://img.shields.io/badge/WebSocket-010101?style=flat-square&logo=socket.io&logoColor=white)

</div>

| Layer | Technology |
|---|---|
| Language | Go 1.24+ |
| HTTP Router | `github.com/gin-gonic/gin` |
| P2P Transport | `github.com/libp2p/go-libp2p` v0.48 |
| Pub-Sub | `github.com/libp2p/go-libp2p-pubsub` (GossipSub) |
| Database | `go.etcd.io/bbolt` v1.4 |
| Cryptography | `crypto/ecdsa` (stdlib), SHA-256 |
| Frontend | Vanilla HTML + CSS + Fetch API |

---

## Quick Start

### Prerequisites

- Go 1.24 or later
- `jq` (optional, for pretty-printing curl output)

### 1 — Clone & build

```bash
git clone <repo-url>
cd hybrid-consensus-net

go mod tidy
go build -o bin/node ./cmd/node/main.go
```

### 2 — Start a PoW node

```bash
./bin/node -consensus pow -difficulty 3 -addr :8080 -db blockchain.db
```

### 3 — Open the dashboard

```
http://localhost:8080
```

The dashboard auto-refreshes every 5 seconds and lets you mine blocks, submit transactions, check balances, and toggle consensus — all from the browser.

### 4 — Start a second node (P2P)

```bash
./bin/node -consensus pow -difficulty 3 -addr :8081 -db blockchain2.db
```

mDNS discovers both nodes automatically on the same LAN — no manual peer configuration needed.

---

## CLI Flags

| Flag | Default | Description |
|---|---|---|
| `-consensus` | `pow` | Consensus mechanism: `pow` or `pos` |
| `-difficulty` | `4` | PoW difficulty (number of leading zero nibbles) |
| `-addr` | `:8080` | HTTP API listen address |
| `-db` | `blockchain.db` | Path to bbolt database file |

---

## API Reference

### Base URL: `http://localhost:8080`

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/status` | Node status: consensus type, height, pending txs |
| `GET` | `/blocks` | All blocks, ordered oldest → newest |
| `GET` | `/mempool` | Pending (unconfirmed) transactions |
| `GET` | `/balance/:address` | Token balance for an address |
| `POST` | `/transactions` | Submit a signed transaction |
| `POST` | `/mine` | Mine a new block from the mempool |
| `POST` | `/consensus` | Switch consensus mechanism at runtime |
| `POST` | `/faucet` | Credit tokens to an address (dev / PoS testing) |

#### POST `/transactions`

```json
{
  "sender":    "<hex-encoded public key>",
  "recipient": "<hex-encoded public key>",
  "amount":    42,
  "signature": "<hex-encoded ECDSA signature>"
}
```

#### POST `/consensus`

```json
{ "type": "pos" }
{ "type": "pow", "difficulty": 3 }
```

#### POST `/faucet`

```json
{ "address": "validator1", "amount": 1000 }
```

---

## Consensus Mechanisms

### Proof of Work (PoW)

- Iterates the block's `Nonce` until `SHA-256(block)` has the required number of leading zero nibbles.
- Difficulty is set via the `-difficulty` flag or the `POST /consensus` endpoint.
- CPU-bound; suitable for local testing and academic demonstration.

### Proof of Stake (PoS)

- Selects the block producer via a stake-weighted random lottery drawn from `State` account balances.
- Requires at least one funded account — use `POST /faucet` to seed validators before switching.
- Deterministic given the same `math/rand` seed; instant block production (no hashing loop).

### Hot-Swap at Runtime

```bash
# Switch to PoS (validators must be funded first)
curl -X POST http://localhost:8080/consensus \
  -H "Content-Type: application/json" \
  -d '{"type":"pos"}'

# Switch back to PoW with difficulty 2
curl -X POST http://localhost:8080/consensus \
  -H "Content-Type: application/json" \
  -d '{"type":"pow","difficulty":2}'
```

---

## Testing

```bash
# Run all tests
go test ./...

# Verbose output
go test ./... -v

# Race detector
go test -race ./...

# Single package
go test ./internal/consensus/...
go test ./internal/blockchain/...
go test ./internal/network/...
```

See [TESTING.md](TESTING.md) for the complete end-to-end testing guide including curl scripts for PoW, PoS, and consensus-switch scenarios.

---

## Development

```bash
# Format code
go fmt ./...

# Vet
go vet ./...

# Build
go build -o bin/node ./cmd/node/main.go

# Rebuild & run in one line
go build -o bin/node ./cmd/node/main.go && ./bin/node -difficulty 2
```

---

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request.

Key extension points:

- **New consensus algorithm** — implement the `consensus.Consensus` interface (`internal/consensus/interface.go`) and wire it in `cmd/node/main.go`.
- **Persistent state** — replace the in-memory `State` with a bbolt-backed implementation using the `Store` interface pattern.
- **WAN peer discovery** — add `libp2p-kad-dht` bootstrap nodes alongside the existing mDNS service in `internal/network/node.go`.

---

## Security

This project is an **academic prototype**. See [SECURITY.md](SECURITY.md) for known limitations before using in any production or adversarial context.

---

## License

MIT — see [LICENSE](LICENSE) for details.

---

<div align="center">
Built with Go · libp2p · bbolt · Gin
</div>
