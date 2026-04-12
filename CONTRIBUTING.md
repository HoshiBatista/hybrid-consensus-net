# Contributing to Hybrid Consensus Network

Thank you for your interest in contributing! This document explains how to set up a development environment, follow code conventions, and submit a pull request.

---

## Table of Contents

- [Development Environment](#development-environment)
- [Project Layout](#project-layout)
- [Code Conventions](#code-conventions)
- [Adding a New Consensus Mechanism](#adding-a-new-consensus-mechanism)
- [Writing Tests](#writing-tests)
- [Submitting a Pull Request](#submitting-a-pull-request)
- [Commit Message Style](#commit-message-style)

---

## Development Environment

### Requirements

| Tool | Version |
|---|---|
| Go | 1.24+ |
| git | any recent |
| jq | optional (curl testing) |

### Setup

```bash
# Clone the repo
git clone <repo-url>
cd hybrid-consensus-net

# Download dependencies
go mod tidy

# Build
go build -o bin/node ./cmd/node/main.go

# Run all tests
go test ./...

# Format
go fmt ./...
```

---

## Project Layout

```
cmd/node/main.go          ← entry point only; no business logic
internal/
  api/server.go           ← HTTP routes
  blockchain/             ← block, tx, state, mempool, persistence
  consensus/              ← Consensus interface + PoW + PoS
  network/node.go         ← libp2p + GossipSub + mDNS
frontend/index.html       ← dashboard
```

Keep business logic inside `internal/`. The `cmd/` layer only wires components together.

---

## Code Conventions

These follow the project's style preferences in `CLAUDE.md`:

- **Exported** identifiers: `PascalCase`
- **Unexported** identifiers: `camelCase`
- **Functions**: single responsibility; prefer short functions
- **Errors**: use `fmt.Errorf("package: context: %w", err)` wrapping
- **Global state**: avoid; use dependency injection
- **Comments**: document all exported types and functions with a doc comment
- **Format**: always run `go fmt ./...` before committing

---

## Adding a New Consensus Mechanism

1. Create `internal/consensus/<name>.go`.
2. Implement the `Consensus` interface:

```go
type Consensus interface {
    MineBlock(block *blockchain.Block) error
    ValidateBlock(block *blockchain.Block) bool
    Type() string
}
```

3. Return a unique, lowercase string from `Type()` (e.g. `"pbft"`).
4. Wire it in `cmd/node/main.go` under a new `case` in the `switch *consensusType` block.
5. Handle it in `internal/api/server.go`'s `switchConsensus` handler if runtime switching is desired.
6. Add tests in `internal/consensus/<name>_test.go`.

---

## Writing Tests

- Place test files next to the package they test: `foo_test.go` alongside `foo.go`.
- Use table-driven tests where multiple cases apply.
- Keep tests deterministic — seed any RNG with a fixed value.
- Use the race detector locally: `go test -race ./...`
- Name tests descriptively: `TestMineBlock_SetsHashAndNonce`, `TestDebit_InsufficientBalance_ReturnsError`.

```go
func TestMyConsensus_MineBlock(t *testing.T) {
    block := blockchain.NewBlock(nil, []byte("prev"), 1)
    cs := consensus.NewMyConsensus()
    if err := cs.MineBlock(block); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !cs.ValidateBlock(block) {
        t.Error("mined block did not validate")
    }
}
```

---

## Submitting a Pull Request

1. Create a feature branch from `main`:
   ```bash
   git checkout -b feature/my-feature
   ```
2. Make focused, atomic commits (one logical change per commit).
3. Ensure all tests pass and code is formatted:
   ```bash
   go test ./... && go fmt ./...
   ```
4. Open a PR against `main` with a clear title and description explaining **what** and **why**.
5. Link any related issues in the PR description.

---

## Commit Message Style

```
<type>: <short summary in imperative mood>

<optional body: what and why, not how>
```

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`

Examples:

```
feat: add PBFT consensus implementation
fix: prevent zero-division in PoS when state is empty
test: add race-detector test for concurrent mempool access
docs: document P2P block gossip flow in ARCHITECTURE.md
```

Keep the summary line under 72 characters.
