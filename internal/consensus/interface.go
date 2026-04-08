// Package consensus defines the pluggable consensus interface used by the
// blockchain core. New mechanisms (PoW, PoS, or others) are added by
// implementing this interface without touching core blockchain logic.
package consensus

import "go-hybrid-blockchain/internal/blockchain"

// Consensus is the contract every block-production and block-validation
// strategy must satisfy. The blockchain engine calls MineBlock to produce a
// new block and ValidateBlock to accept or reject a received block.
type Consensus interface {
	// MineBlock finalizes block by computing and setting its Hash field
	// (and Nonce for PoW, or Validator for PoS). Returns an error if the
	// algorithm fails to produce a valid block.
	MineBlock(block *blockchain.Block) error

	// ValidateBlock returns true when block's Hash correctly satisfies all
	// rules of this consensus mechanism (difficulty target, stake weight, etc.).
	ValidateBlock(block *blockchain.Block) bool

	// Type returns a short, stable identifier for the mechanism, e.g. "pow"
	// or "pos". Used by config/flags to select the active implementation.
	Type() string
}
