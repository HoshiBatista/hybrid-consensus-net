# Changelog

All notable changes to this project are documented here.  
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [Unreleased]

### Planned
- Persistent state ledger (bbolt-backed `State`)
- Fork choice rule for PoW (longest chain)
- Minimum stake threshold for PoS validators
- Full ECDSA signature verification on mempool admission
- WAN peer discovery via `libp2p-kad-dht`

---

## [0.4.0] — Full Refactor

### Changed
- Complete architecture refactor into standard Go project layout (`/cmd`, `/internal`, `/pkg`)
- Consensus mechanism extracted behind a `Consensus` interface — PoW and PoS are now interchangeable without touching core blockchain logic
- `State` and `Mempool` separated into independent components
- All packages documented with Go doc comments

### Added
- `internal/network/node.go`: libp2p + GossipSub + mDNS P2P layer
- `internal/api/server.go`: 8-endpoint Gin REST API with CORS middleware
- Runtime consensus switching via `POST /consensus`
- `POST /faucet` endpoint for PoS validator funding
- `GET /mempool` endpoint
- `GET /balance/:address` endpoint
- Comprehensive unit tests for consensus, blockchain, and network packages
- `TESTING.md` end-to-end test guide with curl scripts

---

## [0.3.0] — Interface Layer

### Added
- `consensus.Consensus` interface (`MineBlock`, `ValidateBlock`, `Type`)
- `ProofOfWork` implementation (SHA-256 nonce iteration)
- `ProofOfStake` implementation (stake-weighted random selection)
- Flag-driven consensus selection at startup (`-consensus pow|pos`)

---

## [0.2.0] — Demo

### Added
- Working demo of block mining and chain inspection
- Basic frontend HTML dashboard
- REST API (early version)

---

## [0.1.0] — Initial Wallets & Chain

### Added
- Basic `Block` and `Transaction` structs
- SHA-256 block hashing
- ECDSA key generation and transaction signing
- Genesis block creation
- bbolt persistence layer
