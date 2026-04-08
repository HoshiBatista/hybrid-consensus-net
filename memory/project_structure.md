---
name: Project structure and architecture decisions
description: Key layout decisions and what lives where in the blockchain package
type: project
---

Module: `go-hybrid-blockchain`, Go 1.25, bbolt for persistence.

**Package layout (internal/):**
- `blockchain/models.go` — canonical `Block` + `Transaction` struct definitions, `Block.CalculateHash()` (SHA-256), `Block.HashTransactions()`, `IntToHex()` util
- `blockchain/block.go` — `NewBlock()` constructor, `MarshalJSON()` (hex-encodes Hash/PrevBlockHash)
- `blockchain/transaction.go` — `NewTransaction()`, `Transaction.CalculateHash()`, `Sign()`, `Verify()`
- `blockchain/blockchain.go` — `Blockchain` (bbolt DB), `CreateBlockchain()`, `AddBlock()`, `GetAllBlocks()`, `MineBlock()` (PoW), `MineBlockPoS()`
- `blockchain/pow.go` — `ProofOfWork`, `Difficulty=4`, `Run()`, `Validate()`; uses `Block.HashTransactions()` and `IntToHex` from models.go
- `blockchain/pos.go` — `CreatePoSBlock()`, `MineBlockPoS()`
- `consensus/interface.go` — `Consensus` interface: `MineBlock(*Block) error`, `ValidateBlock(*Block) bool`, `Type() string`
- `network/` — TCP P2P server with `TypeBlock/TypeTx/TypeGetChain/TypeChain` messages
- `api/` — HTTP handlers `/chain`, `/mine`, `/mine/pos` + embedded dashboard HTML
- `wallet/` — ECDSA key pair generation, `GetAddress()`, `PublicKeyFromBytes()`

**Key design decisions:**
- `NewBlock()` does NOT set Hash — consensus implementations set it via `CalculateHash()` or PoW search
- Genesis block hash is hardcoded in `CreateBlockchain()` as `[]byte("00000000...")`
- `consensus` package imports `blockchain` (one-way); no circular dependency

**Why:** CLAUDE.md mandates "Interfaces First" and Consensus must be swappable via config without touching core logic.

**How to apply:** When adding a new consensus mechanism, implement `consensus.Consensus` in a new file; wire it to `blockchain.MineBlock` via flag.
