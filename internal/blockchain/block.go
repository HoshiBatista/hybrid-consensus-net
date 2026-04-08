package blockchain

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// NewBlock constructs a Block with the given transactions, previous hash, and
// chain height. The Hash field is intentionally left empty; it must be set by
// the active Consensus implementation before the block is persisted.
func NewBlock(transactions []*Transaction, prevHash []byte, height int) *Block {
	return &Block{
		Timestamp:     time.Now().Unix(),
		Transactions:  transactions,
		PrevBlockHash: prevHash,
		Hash:          []byte{},
		Nonce:         0,
		Height:        height,
	}
}

// Serialize encodes the block to JSON bytes for persistence.
func (b *Block) Serialize() []byte {
	data, _ := json.Marshal(b)
	return data
}

// IsValid reports whether the block is structurally self-consistent:
//   - Hash must be non-empty (the block has been finalised by a consensus engine).
//   - PrevBlockHash must match expectedPrev exactly (the chain link is intact).
//
// Consensus-specific validity (PoW target, PoS validator) is not checked here;
// call consensus.ValidateBlock for that.
func (b *Block) IsValid(expectedPrev []byte) bool {
	if len(b.Hash) == 0 {
		return false
	}
	return bytes.Equal(b.PrevBlockHash, expectedPrev)
}

// DeserializeBlock restores a Block from its JSON representation.
// Returns nil if the data cannot be decoded.
func DeserializeBlock(data []byte) *Block {
	var block Block
	if err := json.Unmarshal(data, &block); err != nil {
		return nil
	}
	return &block
}

// MarshalJSON encodes Hash and PrevBlockHash as hex strings instead of the
// default base64 that encoding/json uses for []byte fields.
func (b *Block) MarshalJSON() ([]byte, error) {
	type Alias Block
	return json.Marshal(&struct {
		*Alias
		Hash          string `json:"hash"`
		PrevBlockHash string `json:"prev_block_hash"`
	}{
		Alias:         (*Alias)(b),
		Hash:          hex.EncodeToString(b.Hash),
		PrevBlockHash: hex.EncodeToString(b.PrevBlockHash),
	})
}

// UnmarshalJSON decodes hex-encoded Hash and PrevBlockHash fields, delegating
// all other fields to the standard decoder.
func (b *Block) UnmarshalJSON(data []byte) error {
	type Alias Block
	aux := &struct {
		*Alias
		Hash          string `json:"hash"`
		PrevBlockHash string `json:"prev_block_hash"`
	}{
		Alias: (*Alias)(b),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	var err error
	if b.Hash, err = hex.DecodeString(aux.Hash); err != nil {
		return fmt.Errorf("block: decode hash: %w", err)
	}
	if b.PrevBlockHash, err = hex.DecodeString(aux.PrevBlockHash); err != nil {
		return fmt.Errorf("block: decode prev_block_hash: %w", err)
	}
	return nil
}
