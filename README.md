# Go Hybrid Blockchain (PoW/PoS)

A decentralized, peer-to-peer blockchain network implemented in Go, featuring swappable consensus mechanisms: Proof-of-Work (PoW) and Proof-of-Stake (PoS).

## Features
- **Dual Consensus**: Support for both mining (PoW) and staking (PoS) algorithms.
- **P2P Networking**: Custom TCP-based peer-to-peer communication using JSON messaging.
- **Security**: ECDSA (Elliptic Curve Digital Signature Algorithm) for transaction signing and verification.
- **Persistence**: Embedded key-value storage using BoltDB.
- **Integrity**: SHA-256 hashing for blocks and Merkle-root-like transaction sets.
- **HTTP API**: RESTful endpoints to interact with the node, manage wallets, and monitor the chain.

## Project Structure
- `cmd/`: Entry points for the node application.
- `internal/blockchain/`: Core logic for blocks, transactions, and the chain.
- `internal/consensus/`: Implementation of PoW and PoS algorithms.
- `internal/network/`: P2P server and synchronization logic.
- `internal/storage/`: Database interaction layer (BoltDB).
- `internal/wallet/`: Cryptography tools and wallet management.
- `api/`: HTTP server handlers and API definitions.

## Getting Started

### Prerequisites
- Go 1.24 or higher
- BoltDB (installed as a Go dependency)

### Installation
1. Clone the repository:
   ```bash
   git clone https://github.com/HoshiBatista/go-hybrid-blockchain.git