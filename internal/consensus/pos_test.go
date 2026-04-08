package consensus

import (
	"crypto/sha256"
	"testing"

	"go-hybrid-blockchain/internal/blockchain"
)

// --- helpers -------------------------------------------------------------

// stateWith returns a State pre-loaded with the given address→stake pairs.
func stateWith(pairs ...any) *blockchain.State {
	s := blockchain.NewState()
	for i := 0; i+1 < len(pairs); i += 2 {
		s.Credit(pairs[i].(string), uint64(pairs[i+1].(int)))
	}
	return s
}

// mintedBlock mines a fresh block and fails the test on error.
func mintedBlock(t *testing.T, pos *ProofOfStake, prevHash []byte) *blockchain.Block {
	t.Helper()
	block := blockchain.NewBlock([]*blockchain.Transaction{}, prevHash, 1)
	if err := pos.MineBlock(block); err != nil {
		t.Fatalf("MineBlock unexpectedly failed: %v", err)
	}
	return block
}

// --- interface conformance -----------------------------------------------

func TestPoS_ImplementsConsensus(t *testing.T) {
	var _ Consensus = NewProofOfStake(blockchain.NewState())
}

// --- Type ----------------------------------------------------------------

func TestPoS_Type(t *testing.T) {
	pos := NewProofOfStake(blockchain.NewState())
	if got := pos.Type(); got != "pos" {
		t.Fatalf("Type() = %q, want \"pos\"", got)
	}
}

// --- MineBlock -----------------------------------------------------------

func TestPoS_MineBlock_SetsValidatorField(t *testing.T) {
	pos := NewProofOfStake(stateWith("alice", 100))
	block := mintedBlock(t, pos, []byte("prevhash"))
	if block.Validator == "" {
		t.Fatal("block.Validator must be non-empty after MineBlock")
	}
}

func TestPoS_MineBlock_SetsHashField(t *testing.T) {
	pos := NewProofOfStake(stateWith("alice", 100))
	block := mintedBlock(t, pos, []byte("prevhash"))
	if len(block.Hash) == 0 {
		t.Fatal("block.Hash must be non-empty after MineBlock")
	}
}

func TestPoS_MineBlock_SingleValidator_AlwaysSelected(t *testing.T) {
	pos := NewProofOfStake(stateWith("alice", 100))
	// Even with different prevHashes, the only eligible validator is alice.
	prevHashes := [][]byte{
		[]byte(""),
		[]byte("genesis"),
		[]byte("some other hash"),
	}
	for _, ph := range prevHashes {
		block := mintedBlock(t, pos, ph)
		if block.Validator != "alice" {
			t.Fatalf("expected validator alice, got %q (prevHash=%q)", block.Validator, ph)
		}
	}
}

func TestPoS_MineBlock_ValidatorHasPositiveStake(t *testing.T) {
	s := stateWith("alice", 50, "bob", 200, "carol", 10)
	pos := NewProofOfStake(s)
	block := mintedBlock(t, pos, []byte("some-prev"))
	if s.Balance(block.Validator) == 0 {
		t.Fatalf("selected validator %q must have positive stake", block.Validator)
	}
}

func TestPoS_MineBlock_NoValidators_ReturnsError(t *testing.T) {
	pos := NewProofOfStake(blockchain.NewState())
	block := blockchain.NewBlock([]*blockchain.Transaction{}, []byte("prev"), 1)
	if err := pos.MineBlock(block); err == nil {
		t.Fatal("expected error when state has no validators, got nil")
	}
}

func TestPoS_MineBlock_AllZeroStake_ReturnsError(t *testing.T) {
	s := blockchain.NewState()
	s.Credit("alice", 50)
	_ = s.Debit("alice", 50) // drain to zero
	pos := NewProofOfStake(s)
	block := blockchain.NewBlock([]*blockchain.Transaction{}, []byte("prev"), 1)
	if err := pos.MineBlock(block); err == nil {
		t.Fatal("expected error when all validators have zero stake, got nil")
	}
}

func TestPoS_MineBlock_WithTransactions(t *testing.T) {
	pos := NewProofOfStake(stateWith("alice", 100))
	tx := blockchain.NewTransaction([]byte("sender"), []byte("receiver"), 10)
	block := blockchain.NewBlock([]*blockchain.Transaction{tx}, []byte("prev"), 2)
	if err := pos.MineBlock(block); err != nil {
		t.Fatalf("MineBlock with transactions failed: %v", err)
	}
	if !pos.ValidateBlock(block) {
		t.Fatal("block with transactions must validate after MineBlock")
	}
}

// --- ValidateBlock -------------------------------------------------------

func TestPoS_ValidateBlock_ValidBlock(t *testing.T) {
	pos := NewProofOfStake(stateWith("alice", 100))
	block := mintedBlock(t, pos, []byte("prevhash"))
	if !pos.ValidateBlock(block) {
		t.Fatal("ValidateBlock must return true for a freshly minted block")
	}
}

func TestPoS_ValidateBlock_TamperedValidator(t *testing.T) {
	s := stateWith("alice", 100, "bob", 200)
	pos := NewProofOfStake(s)
	block := mintedBlock(t, pos, []byte("prevhash"))

	// Swap validator to the other eligible address (bob has stake, but is
	// not the expected winner for this prevHash).
	if block.Validator == "alice" {
		block.Validator = "bob"
	} else {
		block.Validator = "alice"
	}

	if pos.ValidateBlock(block) {
		t.Fatal("ValidateBlock must return false when Validator field is tampered with")
	}
}

func TestPoS_ValidateBlock_TamperedHash(t *testing.T) {
	pos := NewProofOfStake(stateWith("alice", 100))
	block := mintedBlock(t, pos, []byte("prevhash"))

	// Flip a bit in the stored hash — the recomputed hash won't match.
	block.Hash[0] ^= 0xFF

	if pos.ValidateBlock(block) {
		t.Fatal("ValidateBlock must return false when block.Hash is tampered with")
	}
}

func TestPoS_ValidateBlock_UnknownValidator(t *testing.T) {
	pos := NewProofOfStake(stateWith("alice", 100))
	block := mintedBlock(t, pos, []byte("prevhash"))

	// Replace with a validator address that does not exist in state.
	block.Validator = "nobody"

	if pos.ValidateBlock(block) {
		t.Fatal("ValidateBlock must return false for a validator with zero stake")
	}
}

func TestPoS_ValidateBlock_StakeDepletedAfterMinting(t *testing.T) {
	s := stateWith("alice", 100)
	pos := NewProofOfStake(s)
	block := mintedBlock(t, pos, []byte("prevhash"))

	// Drain alice's stake to zero after the block was minted.
	_ = s.Debit("alice", 100)

	// The block was validly minted, but the current state no longer recognises
	// alice as an eligible validator.
	if pos.ValidateBlock(block) {
		t.Fatal("ValidateBlock must return false when the validator's stake has been depleted")
	}
}

func TestPoS_ValidateBlock_UnminedBlock(t *testing.T) {
	pos := NewProofOfStake(stateWith("alice", 100))
	block := blockchain.NewBlock([]*blockchain.Transaction{}, []byte("prev"), 1)
	// block.Validator == "" and block.Hash == nil — must not validate.
	if pos.ValidateBlock(block) {
		t.Fatal("an unminted block must not pass validation")
	}
}

// --- Weighted lottery distribution ---------------------------------------

// TestSelectValidator_WeightedDistribution verifies that validator selection
// is stake-proportional. Setup: "minor" has 1/10 of total stake, "major" has
// 9/10. Across 500 different prevHashes the observed selection frequency for
// "major" must exceed 70% (the true expectation is 90%; the loose bound
// prevents flakiness while still catching a broken weighting).
func TestSelectValidator_WeightedDistribution(t *testing.T) {
	const samples = 500
	const minMajorFraction = 0.70

	s := stateWith("major", 9, "minor", 1) // total stake = 10
	pos := NewProofOfStake(s)

	counts := map[string]int{}
	for i := 0; i < samples; i++ {
		// Derive a unique, realistic-looking prevHash for each sample.
		h := sha256.Sum256([]byte{byte(i >> 8), byte(i), 0xDE, 0xAD})
		block := blockchain.NewBlock([]*blockchain.Transaction{}, h[:], 1)
		if err := pos.MineBlock(block); err != nil {
			t.Fatalf("MineBlock failed at sample %d: %v", i, err)
		}
		counts[block.Validator]++
	}

	majorFraction := float64(counts["major"]) / float64(samples)
	if majorFraction < minMajorFraction {
		t.Fatalf(
			"major validator (9/10 stake) won %.1f%% of slots, want >= %.1f%%\ncounts: %v",
			majorFraction*100, minMajorFraction*100, counts,
		)
	}
}

// TestSelectValidator_TwoEqualStakes checks that with two validators of
// identical stake each wins approximately half the time.
func TestSelectValidator_TwoEqualStakes(t *testing.T) {
	const samples = 500
	const minFraction = 0.35 // each should win ~50%; allow ±15%

	s := stateWith("alice", 5, "bob", 5)
	pos := NewProofOfStake(s)

	counts := map[string]int{}
	for i := 0; i < samples; i++ {
		h := sha256.Sum256([]byte{byte(i >> 8), byte(i), 0xCA, 0xFE})
		block := blockchain.NewBlock([]*blockchain.Transaction{}, h[:], 1)
		if err := pos.MineBlock(block); err != nil {
			t.Fatalf("MineBlock failed: %v", err)
		}
		counts[block.Validator]++
	}

	for _, name := range []string{"alice", "bob"} {
		f := float64(counts[name]) / float64(samples)
		if f < minFraction {
			t.Fatalf(
				"%s won %.1f%% of slots with equal stake, want >= %.1f%%\ncounts: %v",
				name, f*100, minFraction*100, counts,
			)
		}
	}
}

// TestSelectValidator_DeterministicForSamePrevHash asserts that the same
// prevHash always produces the same validator regardless of how many times
// MineBlock is called. This is the property that makes re-validation possible.
func TestSelectValidator_DeterministicForSamePrevHash(t *testing.T) {
	s := stateWith("alice", 3, "bob", 7)
	pos := NewProofOfStake(s)
	prevHash := []byte("fixed-prev-hash")

	// Mine 20 independent blocks with the same prevHash; all must agree.
	first := mintedBlock(t, pos, prevHash)
	for i := 1; i < 20; i++ {
		block := mintedBlock(t, pos, prevHash)
		if block.Validator != first.Validator {
			t.Fatalf(
				"non-deterministic selection: iteration %d returned %q, want %q",
				i, block.Validator, first.Validator,
			)
		}
	}
}
