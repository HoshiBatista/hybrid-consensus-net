# Security — Hybrid Consensus Network

This document describes the security properties of the project, known limitations, and guidance for safe use.

---

## Scope

This project is an **academic prototype** built for a university course. It demonstrates blockchain mechanics (PoW/PoS, P2P networking, ECDSA transactions) but is **not hardened for production use**.

---

## What Is Implemented

| Mechanism | Status | Notes |
|---|---|---|
| ECDSA transaction signing | Implemented | `transaction.Sign(*ecdsa.PrivateKey)` |
| SHA-256 block hashing | Implemented | Used in both PoW and block ID |
| Signature presence check | Implemented | `Verify()` currently checks non-empty only |
| Full ECDSA signature verification | Partial | Signature bytes stored; full cryptographic verification is present in `transaction.go` |
| Balance pre-validation on mempool add | Implemented | Prevents spending beyond balance |

---

## Known Limitations

### 1. Signature Verification Is Partial for Testing

`mempool.Add` accepts transactions with any non-empty `Signature` hex string. This is intentional for test convenience (the TESTING.md guide uses `"ff"` as a stub signature). In a production system, every transaction must be cryptographically verified against the sender's public key before mempool admission.

### 2. In-Memory State Is Not Persistent

Account balances (`internal/blockchain/state.go`) are held in a `map[string]uint64` that resets on node restart. The balance ledger is not replayed from transaction history on startup. This means balances are lost on crash.

### 3. No Sybil Protection

The PoS implementation selects validators purely by balance. There is no minimum stake threshold, slashing, or validator set governance. An attacker who controls a large balance can dominate block production.

### 4. No Fork Choice Rule

The node accepts the first valid block it receives and does not implement a longest-chain or highest-stake fork choice rule. Competing chains from different miners are not resolved.

### 5. mDNS Only (No WAN Peer Discovery)

Peer discovery uses mDNS, which is local-network only. Nodes on separate networks cannot find each other automatically. There is no DHT, bootstrap node list, or peer exchange protocol.

### 6. No Rate Limiting or DoS Protection

The REST API has no rate limiting. The mempool has no maximum size cap. A malicious client can flood the mempool with transactions.

### 7. CORS Is Open

`Access-Control-Allow-Origin: *` is set globally. This is acceptable for a local development node but must be restricted in any Internet-facing deployment.

### 8. No TLS on the REST API

The HTTP API runs over plain HTTP. Sensitive operations (faucet, consensus switch) have no authentication. Do not expose the API port to the public internet.

---

## Recommendations Before Any Non-Academic Use

1. Implement full ECDSA signature verification on every mempool admission.
2. Persist and replay the state ledger from the block store on startup.
3. Add a fork choice rule (longest chain for PoW; highest finality for PoS).
4. Add mempool size limits and per-IP rate limiting.
5. Restrict CORS to known origins and add API key or JWT authentication.
6. Enable TLS on the REST API listener.
7. Add a minimum stake requirement and slashing conditions for PoS.

---

## Reporting Issues

If you discover a security concern relevant to this project's academic context, open an issue in the repository with the label `security`.
