package consensus

import (
	"bytes"
	"testing"

	"go-hybrid-blockchain/internal/blockchain"
)

// difficulty used across all tests — low enough to finish in milliseconds.
const testDifficulty = 1

// newTestBlock is a helper that builds a minimal block suitable for mining tests.
func newTestBlock(prevHash []byte) *blockchain.Block {
	return blockchain.NewBlock([]*blockchain.Transaction{}, prevHash, 1)
}

// --- interface conformance -----------------------------------------------

// TestImplementsConsensus is a compile-time assertion: if ProofOfWork ever
// stops satisfying Consensus the file will fail to compile, not just the test.
func TestImplementsConsensus(t *testing.T) {
	var _ Consensus = NewProofOfWork(testDifficulty)
}

// --- Type ----------------------------------------------------------------

func TestType(t *testing.T) {
	pow := NewProofOfWork(testDifficulty)
	if got := pow.Type(); got != "pow" {
		t.Fatalf("Type() = %q, want \"pow\"", got)
	}
}

// --- MineBlock -----------------------------------------------------------

func TestMineBlock_SetsHashAndNonce(t *testing.T) {
	pow := NewProofOfWork(testDifficulty)
	block := newTestBlock([]byte("genesis"))

	if err := pow.MineBlock(block); err != nil {
		t.Fatalf("MineBlock returned unexpected error: %v", err)
	}

	if len(block.Hash) == 0 {
		t.Fatal("block.Hash must be non-empty after mining")
	}
	if block.Nonce < 0 {
		t.Fatalf("block.Nonce must be >= 0 after mining, got %d", block.Nonce)
	}
}

func TestMineBlock_HashBelowTarget(t *testing.T) {
	pow := NewProofOfWork(testDifficulty)
	block := newTestBlock([]byte("genesis"))

	if err := pow.MineBlock(block); err != nil {
		t.Fatalf("MineBlock error: %v", err)
	}

	// The first nibble of a hash that satisfies difficulty=1 must be 0x0.
	if block.Hash[0]>>4 != 0 {
		t.Fatalf("mined hash %x does not satisfy difficulty=%d (leading nibble must be 0)", block.Hash, testDifficulty)
	}
}

func TestMineBlock_WithTransactions(t *testing.T) {
	pow := NewProofOfWork(testDifficulty)

	tx := blockchain.NewTransaction([]byte("alice"), []byte("bob"), 42)
	block := blockchain.NewBlock([]*blockchain.Transaction{tx}, []byte("prevhash"), 2)

	if err := pow.MineBlock(block); err != nil {
		t.Fatalf("MineBlock error: %v", err)
	}

	if !pow.ValidateBlock(block) {
		t.Fatal("block containing transactions should be valid after mining")
	}
}

func TestMineBlock_DifferentPrevHash_DifferentResult(t *testing.T) {
	pow := NewProofOfWork(testDifficulty)

	b1 := newTestBlock([]byte("parent-a"))
	b2 := newTestBlock([]byte("parent-b"))

	if err := pow.MineBlock(b1); err != nil {
		t.Fatalf("MineBlock(b1) error: %v", err)
	}
	if err := pow.MineBlock(b2); err != nil {
		t.Fatalf("MineBlock(b2) error: %v", err)
	}

	if bytes.Equal(b1.Hash, b2.Hash) {
		t.Fatal("blocks with different PrevBlockHash must produce different hashes")
	}
}

// --- ValidateBlock -------------------------------------------------------

func TestValidateBlock_ValidAfterMining(t *testing.T) {
	pow := NewProofOfWork(testDifficulty)
	block := newTestBlock([]byte("genesis"))

	if err := pow.MineBlock(block); err != nil {
		t.Fatalf("MineBlock error: %v", err)
	}

	if !pow.ValidateBlock(block) {
		t.Fatal("ValidateBlock must return true for a freshly mined block")
	}
}

func TestValidateBlock_TamperedHash(t *testing.T) {
	pow := NewProofOfWork(testDifficulty)
	block := newTestBlock([]byte("genesis"))

	if err := pow.MineBlock(block); err != nil {
		t.Fatalf("MineBlock error: %v", err)
	}

	// Flip a bit in the stored Hash — the nonce still produces the original
	// hash, so the equality check inside ValidateBlock must catch the mismatch.
	block.Hash[0] ^= 0xFF

	if pow.ValidateBlock(block) {
		t.Fatal("ValidateBlock must return false when block.Hash is tampered with")
	}
}

func TestValidateBlock_TamperedNonce(t *testing.T) {
	pow := NewProofOfWork(testDifficulty)
	block := newTestBlock([]byte("genesis"))

	if err := pow.MineBlock(block); err != nil {
		t.Fatalf("MineBlock error: %v", err)
	}

	// Corrupt the nonce: the re-hashed result will (almost certainly) differ
	// from block.Hash and is also unlikely to meet the target.
	block.Nonce = ^block.Nonce // bitwise NOT ≈ large, clearly wrong value

	if pow.ValidateBlock(block) {
		t.Fatal("ValidateBlock must return false when Nonce is corrupted")
	}
}

func TestValidateBlock_InsufficientDifficulty(t *testing.T) {
	// Mine at low difficulty, then validate with a stricter engine.
	easy := NewProofOfWork(testDifficulty)
	block := newTestBlock([]byte("genesis"))

	if err := easy.MineBlock(block); err != nil {
		t.Fatalf("MineBlock error: %v", err)
	}

	// A difficulty-16 engine requires 16 leading zero nibbles — a block mined
	// at difficulty=1 has essentially zero chance of satisfying this.
	hard := NewProofOfWork(16)
	if hard.ValidateBlock(block) {
		t.Fatal("block mined at difficulty=1 must not satisfy difficulty=16")
	}
}

func TestValidateBlock_UnminedBlock(t *testing.T) {
	pow := NewProofOfWork(testDifficulty)
	block := newTestBlock([]byte("genesis"))
	// block.Hash is empty and block.Nonce is 0 — almost certainly invalid.

	if pow.ValidateBlock(block) {
		t.Fatal("an unmined block should not pass validation")
	}
}
