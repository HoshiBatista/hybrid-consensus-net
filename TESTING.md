# Testing Guide — Hybrid Consensus Network

## Table of Contents

- [Setup](#setup)
- [Running the Node](#running-the-node)
- [Web Interface](#web-interface)
- [API Reference](#api-reference)
- [PoW Testing](#pow-testing)
- [PoS Testing](#pos-testing)
- [Switching Consensus at Runtime](#switching-consensus-at-runtime)
- [Unit Tests](#unit-tests)
- [Full PoS Sequence](#full-pos-sequence)
- [Transaction Fields Explained](#transaction-fields-explained)

---

## Setup

```bash
cd /Users/crissyro/university/hybrid-consensus-net

# Download dependencies
go mod tidy

# Build binary
go build -o bin/node ./cmd/node/main.go
```

---

## Running the Node

### Proof-of-Work (default)

```bash
./bin/node -consensus pow -difficulty 3 -addr :8080 -db blockchain.db
```

### Proof-of-Stake

```bash
./bin/node -consensus pos -addr :8080 -db blockchain.db
```

> **Flags**
>
> | Flag | Default | Description |
> |---|---|---|
> | `-consensus` | `pow` | Consensus mechanism: `pow` or `pos` |
> | `-difficulty` | `4` | PoW difficulty (leading zero nibbles) |
> | `-addr` | `:8080` | HTTP API listen address |
> | `-db` | `blockchain.db` | Path to bbolt database file |

---

## Web Interface

Once the node is running, open **http://localhost:8080** in your browser.

The dashboard auto-refreshes every 5 seconds and provides:

- Live blockchain visualization (click any block to expand transactions)
- Mempool view
- One-click block mining
- Balance checker
- Faucet (credit tokens for PoS testing)
- Transaction submission with **🎲 Demo** auto-fill
- PoW / PoS toggle in the header

---

## API Reference

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/status` | Node status: consensus type, height, pending txs |
| `GET` | `/blocks` | All blocks, oldest first |
| `GET` | `/mempool` | Pending transactions |
| `GET` | `/balance/:address` | Balance for an address |
| `POST` | `/transactions` | Submit a transaction |
| `POST` | `/mine` | Mine a new block |
| `POST` | `/consensus` | Switch consensus at runtime |
| `POST` | `/faucet` | Credit tokens to an address (testing only) |

---

## PoW Testing

### 1. Check node status

```bash
curl -s http://localhost:8080/status | jq .
```

Expected output:

```json
{
  "blocks": 1,
  "consensus": "pow",
  "difficulty": 3,
  "height": 0,
  "pending_txs": 0
}
```

### 2. Mine an empty block

```bash
curl -s -X POST http://localhost:8080/mine | jq .
```

### 3. Submit transactions

```bash
curl -s -X POST http://localhost:8080/transactions \
  -H "Content-Type: application/json" \
  -d '{
    "sender":    "deadbeef0102030405060708",
    "recipient": "cafebabe0a0b0c0d0e0f1011",
    "amount":    42,
    "signature": "aabbccdd"
  }' | jq .

curl -s -X POST http://localhost:8080/transactions \
  -H "Content-Type: application/json" \
  -d '{
    "sender":    "cafebabe0a0b0c0d0e0f1011",
    "recipient": "deadbeef0102030405060708",
    "amount":    10,
    "signature": "eeff0011"
  }' | jq .
```

### 4. Check mempool

```bash
curl -s http://localhost:8080/mempool | jq .
```

### 5. Mine block with transactions

```bash
curl -s -X POST http://localhost:8080/mine | jq .
```

### 6. View the chain

```bash
curl -s http://localhost:8080/blocks | jq .
```

### 7. Check balances

```bash
curl -s http://localhost:8080/balance/deadbeef0102030405060708 | jq .
curl -s http://localhost:8080/balance/cafebabe0a0b0c0d0e0f1011 | jq .
```

---

## PoS Testing

PoS requires at least one account with a positive balance to act as a validator.

### 1. Fund validators via faucet

```bash
curl -s -X POST http://localhost:8080/faucet \
  -H "Content-Type: application/json" \
  -d '{"address":"validator1","amount":700}' | jq .

curl -s -X POST http://localhost:8080/faucet \
  -H "Content-Type: application/json" \
  -d '{"address":"validator2","amount":300}' | jq .
```

### 2. Switch consensus to PoS

```bash
curl -s -X POST http://localhost:8080/consensus \
  -H "Content-Type: application/json" \
  -d '{"type":"pos"}' | jq .
```

### 3. Submit a transaction

```bash
curl -s -X POST http://localhost:8080/transactions \
  -H "Content-Type: application/json" \
  -d '{"sender":"aabb","recipient":"ccdd","amount":10,"signature":"ff"}' | jq .
```

### 4. Mine a PoS block

```bash
curl -s -X POST http://localhost:8080/mine | jq .
```

### 5. Verify the validator field

```bash
curl -s http://localhost:8080/blocks | jq '.[].validator'
```

The output will show which validator won the stake-weighted lottery.  
`validator1` has 70% stake so it wins ~70% of slots.

---

## Switching Consensus at Runtime

Switch from PoW to PoS:

```bash
curl -s -X POST http://localhost:8080/consensus \
  -H "Content-Type: application/json" \
  -d '{"type":"pos"}' | jq .
```

Switch back to PoW (optionally change difficulty):

```bash
curl -s -X POST http://localhost:8080/consensus \
  -H "Content-Type: application/json" \
  -d '{"type":"pow","difficulty":2}' | jq .
```

> **Note:** Switching to PoS fails if no accounts have a balance.  
> Use `POST /faucet` first.

---

## Unit Tests

### Run all tests

```bash
go test ./...
```

### Run with verbose output

```bash
go test ./... -v
```

### Run a specific package

```bash
go test ./internal/blockchain/...
go test ./internal/consensus/...
go test ./internal/network/...
```

### Run a single test by name

```bash
go test ./internal/consensus/... -run TestMineBlock_SetsHashAndNonce -v
go test ./internal/blockchain/... -run TestDebit_InsufficientBalance_ReturnsError -v
```

### Run with race detector

```bash
go test -race ./...
```

---

## Full PoS Sequence

Copy and run the entire block to test PoS end-to-end:

```bash
# 1. Fund two validators
curl -s -X POST http://localhost:8080/faucet \
  -H "Content-Type: application/json" \
  -d '{"address":"validator1","amount":700}' | jq .

curl -s -X POST http://localhost:8080/faucet \
  -H "Content-Type: application/json" \
  -d '{"address":"validator2","amount":300}' | jq .

# 2. Confirm balances
curl -s http://localhost:8080/balance/validator1 | jq .
curl -s http://localhost:8080/balance/validator2 | jq .

# 3. Switch to PoS
curl -s -X POST http://localhost:8080/consensus \
  -H "Content-Type: application/json" \
  -d '{"type":"pos"}' | jq .

# 4. Submit a transaction
curl -s -X POST http://localhost:8080/transactions \
  -H "Content-Type: application/json" \
  -d '{"sender":"aabb","recipient":"ccdd","amount":50,"signature":"ff"}' | jq .

# 5. Mine
curl -s -X POST http://localhost:8080/mine | jq .

# 6. Check which validator was selected
curl -s http://localhost:8080/blocks | jq '.[] | {height, validator}'

# 7. Mine 5 more blocks to observe stake distribution
for i in $(seq 1 5); do
  curl -s -X POST http://localhost:8080/mine | jq '{height, validator}'
done
```

---

## Transaction Fields Explained

| Field | Type | Description |
|---|---|---|
| `sender` | hex string | Bytes representing the sender's public key |
| `recipient` | hex string | Bytes representing the recipient's public key |
| `amount` | integer ≥ 1 | Number of tokens to transfer |
| `signature` | hex string | ECDSA signature (r‖s) over the transaction hash |

**For testing**, `signature` only needs to be a non-empty hex string — the current `Verify()` check confirms presence only.  
Any of these work: `"ff"`, `"aabbccdd"`, `"deadbeef"`.

Real ECDSA signing uses `transaction.Sign(*ecdsa.PrivateKey)` from `internal/blockchain/transaction.go:43`.
