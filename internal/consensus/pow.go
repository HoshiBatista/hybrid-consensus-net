package consensus

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"

	"go-hybrid-blockchain/internal/blockchain"
)

// ProofOfWork implements the Consensus interface using a hashcash-style
// proof-of-work algorithm. A valid block hash must be numerically less than a
// target derived from Difficulty: the higher the difficulty, the smaller the
// target and the more work required.
//
// The PoW pre-image is:
//
//	PrevBlockHash ‖ HashTransactions ‖ Timestamp ‖ Difficulty ‖ Nonce
//
// Difficulty and Nonce are both included so that:
//   - Changing the difficulty invalidates all previously mined solutions.
//   - The nonce is the only free variable the miner iterates over.
type ProofOfWork struct {
	Difficulty int
}

// NewProofOfWork returns a PoW engine with the given difficulty level.
// Difficulty is measured in leading zero nibbles of the hash: difficulty=4
// means the hash must start with "0000…" in hex (≈ 2^16 expected hashes).
func NewProofOfWork(difficulty int) *ProofOfWork {
	return &ProofOfWork{Difficulty: difficulty}
}

// target returns the numeric upper bound a valid hash must be below.
// It is computed as 1 << (256 - difficulty*4), so each increment of
// difficulty halves the probability of a random hash being acceptable.
func (p *ProofOfWork) target() *big.Int {
	t := big.NewInt(1)
	t.Lsh(t, uint(256-p.Difficulty*4))
	return t
}

// prepareData assembles the byte slice that is SHA-256'd during mining.
// All fields that must be immutable once the block is mined are included;
// only the nonce varies between attempts.
func (p *ProofOfWork) prepareData(block *blockchain.Block, nonce int) []byte {
	return bytes.Join(
		[][]byte{
			block.PrevBlockHash,
			block.HashTransactions(),
			blockchain.IntToHex(block.Timestamp),
			blockchain.IntToHex(int64(p.Difficulty)),
			blockchain.IntToHex(int64(nonce)),
		},
		[]byte{},
	)
}

// MineBlock runs the proof-of-work search loop against block, iterating the
// nonce until a SHA-256 hash below the difficulty target is found. On success
// it sets block.Nonce and block.Hash and returns nil. An error is returned
// only if the entire int64 nonce space is exhausted (astronomically unlikely
// for any practical difficulty setting).
func (p *ProofOfWork) MineBlock(block *blockchain.Block) error {
	target := p.target()

	var hashInt big.Int
	var hash [32]byte
	nonce := 0

	fmt.Printf("Mining (difficulty=%d)...", p.Difficulty)
	for nonce < math.MaxInt64 {
		data := p.prepareData(block, nonce)
		hash = sha256.Sum256(data)
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(target) == -1 {
			fmt.Printf("\rMined block at nonce=%d hash=%x\n", nonce, hash)
			break
		}
		nonce++
	}

	if nonce == math.MaxInt64 {
		return fmt.Errorf("pow: exhausted nonce space at difficulty %d", p.Difficulty)
	}

	block.Nonce = nonce
	block.Hash = hash[:]
	return nil
}

// ValidateBlock re-hashes block using its stored Nonce and returns true only
// when both conditions hold:
//  1. The recomputed hash is numerically below the difficulty target.
//  2. The recomputed hash matches the hash stored in block.Hash.
//
// Condition 2 prevents accepting a block whose Hash field has been overwritten
// after mining without re-running the PoW.
func (p *ProofOfWork) ValidateBlock(block *blockchain.Block) bool {
	target := p.target()

	data := p.prepareData(block, block.Nonce)
	hash := sha256.Sum256(data)

	var hashInt big.Int
	hashInt.SetBytes(hash[:])

	return hashInt.Cmp(target) == -1 && bytes.Equal(hash[:], block.Hash)
}

// Type satisfies the Consensus interface and identifies this mechanism.
func (p *ProofOfWork) Type() string {
	return "pow"
}
