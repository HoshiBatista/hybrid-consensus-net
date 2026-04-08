package consensus

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"

	"go-hybrid-blockchain/internal/blockchain"
)

// ProofOfStake implements the Consensus interface using a stake-weighted
// deterministic validator lottery.
//
// # Validator selection
//
// Given a block's PrevBlockHash and the current State:
//  1. Collect all accounts with stake > 0, sorted alphabetically by address.
//  2. Treat the first 8 bytes of PrevBlockHash as a big-endian uint64 seed.
//  3. Reduce the seed modulo totalStake to obtain a position in [0, totalStake).
//  4. Walk the cumulative-stake distribution until the position falls inside a
//     validator's range — that validator wins the slot.
//
// Because the algorithm is a pure function of (PrevBlockHash, State), any node
// can independently re-derive the expected validator and detect impersonation.
//
// # Block hash
//
// PoS does not require iterative work. The block hash is a single SHA-256 over:
//
//	PrevBlockHash ‖ HashTransactions ‖ Timestamp ‖ Validator
//
// The Validator field ties the hash to the identity of the block producer,
// making the hash invalid if the Validator field is altered.
type ProofOfStake struct {
	state *blockchain.State
}

// NewProofOfStake returns a PoS engine that reads validator stakes from state.
func NewProofOfStake(state *blockchain.State) *ProofOfStake {
	return &ProofOfStake{state: state}
}

// selectValidator deterministically picks a validator proportional to their
// stake using prevHash as an entropy source.
//
// Short prevHash slices (including the empty genesis hash) are zero-padded to
// 8 bytes before the seed is derived, so no panic can occur.
func (p *ProofOfStake) selectValidator(prevHash []byte) (string, error) {
	validators := p.state.Validators() // sorted alphabetically — deterministic
	if len(validators) == 0 {
		return "", errors.New("pos: no validators with positive stake in state")
	}

	total := p.state.TotalStake()
	if total == 0 {
		return "", errors.New("pos: total stake is zero")
	}

	// Derive a uint64 seed from the first 8 bytes of prevHash.
	// Zero-padding ensures short or empty hashes are handled safely.
	var buf [8]byte
	copy(buf[:], prevHash)
	seed := binary.BigEndian.Uint64(buf[:]) % total

	// Walk the cumulative-stake distribution to find the winning slot.
	var cumulative uint64
	for _, v := range validators {
		cumulative += v.Stake
		if seed < cumulative {
			return v.Address, nil
		}
	}

	// Unreachable in practice (seed < total always lands in some bucket),
	// but included as a safe fallback for floating-point-free integer arithmetic.
	return validators[len(validators)-1].Address, nil
}

// blockHash computes the canonical PoS hash for a fully-populated block:
//
//	SHA-256(PrevBlockHash ‖ HashTransactions ‖ Timestamp ‖ Validator)
func (p *ProofOfStake) blockHash(block *blockchain.Block) []byte {
	data := bytes.Join(
		[][]byte{
			block.PrevBlockHash,
			block.HashTransactions(),
			blockchain.IntToHex(block.Timestamp),
			[]byte(block.Validator),
		},
		[]byte{},
	)
	hash := sha256.Sum256(data)
	return hash[:]
}

// MineBlock selects the stake-weighted validator for block, assigns it to
// block.Validator, and writes the deterministic PoS hash to block.Hash.
// No iterative work loop is required; the "proof" is the validator's stake.
//
// Returns an error when:
//   - The State contains no validators with positive stake.
//   - The selected validator's stake is zero at the moment of assignment
//     (guards against a concurrent Debit between selection and minting).
func (p *ProofOfStake) MineBlock(block *blockchain.Block) error {
	validator, err := p.selectValidator(block.PrevBlockHash)
	if err != nil {
		return fmt.Errorf("pos MineBlock: %w", err)
	}

	// Re-check: guard against a stake depletion that raced with selectValidator.
	if p.state.Balance(validator) == 0 {
		return fmt.Errorf("pos MineBlock: selected validator %q stake was depleted", validator)
	}

	block.Validator = validator
	block.Hash = p.blockHash(block)
	return nil
}

// ValidateBlock returns true only when all three conditions hold:
//
//  1. block.Validator has a positive stake in the current State — a validator
//     with zero stake cannot legally produce blocks.
//
//  2. block.Validator equals the deterministically expected winner for
//     block.PrevBlockHash — prevents a high-stake node from claiming every
//     slot or a peer from substituting a different validator address.
//
//  3. block.Hash equals the recomputed PoS hash — prevents hash-field
//     tampering after the block was legitimately minted.
func (p *ProofOfStake) ValidateBlock(block *blockchain.Block) bool {
	// 1. Validator must have positive stake.
	if p.state.Balance(block.Validator) == 0 {
		return false
	}

	// 2. Validator must be the expected lottery winner for this prev hash.
	expected, err := p.selectValidator(block.PrevBlockHash)
	if err != nil || expected != block.Validator {
		return false
	}

	// 3. Stored hash must match the recomputed PoS hash.
	return bytes.Equal(p.blockHash(block), block.Hash)
}

// Type satisfies the Consensus interface.
func (p *ProofOfStake) Type() string {
	return "pos"
}
