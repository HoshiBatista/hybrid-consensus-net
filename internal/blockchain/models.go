package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
)

// Block is the fundamental unit of the blockchain. Each block links to the
// previous one via PrevBlockHash, forming an immutable chain.
type Block struct {
	Timestamp     int64          `json:"timestamp"`
	Transactions  []*Transaction `json:"transactions"`
	PrevBlockHash []byte         `json:"prev_block_hash"`
	Hash          []byte         `json:"hash"`
	Nonce         int            `json:"nonce"`      // Used by PoW
	Validator     string         `json:"validator"`  // Used by PoS (address of block producer)
	Height        int            `json:"height"`     // Ordinal position in the chain
}

// Transaction represents a transfer of value between two parties.
// All transactions are signed with ECDSA before broadcast.
type Transaction struct {
	Sender    []byte `json:"sender"`    // ECDSA public key of the sender
	Recipient []byte `json:"recipient"` // ECDSA public key of the recipient
	Amount    int    `json:"amount"`
	Signature []byte `json:"signature"` // ECDSA signature over TxID
	TxID      []byte `json:"tx_id"`     // SHA-256 hash of the transaction contents
}

// CalculateHash computes the SHA-256 hash of the block's contents:
// PrevBlockHash || HashTransactions || Timestamp || Height.
// Consensus implementations call this to produce or verify a block's Hash field.
func (b *Block) CalculateHash() []byte {
	data := bytes.Join(
		[][]byte{
			b.PrevBlockHash,
			b.HashTransactions(),
			IntToHex(b.Timestamp),
			IntToHex(int64(b.Height)),
		},
		[]byte{},
	)
	hash := sha256.Sum256(data)
	return hash[:]
}

// HashTransactions builds a SHA-256 digest over the concatenation of every
// transaction ID in the block. This acts as a simplified Merkle root.
func (b *Block) HashTransactions() []byte {
	var txHashes [][]byte
	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.TxID)
	}
	hash := sha256.Sum256(bytes.Join(txHashes, []byte{}))
	return hash[:]
}

// IntToHex encodes num as a big-endian 8-byte slice, suitable for
// inclusion in hash pre-images.
func IntToHex(num int64) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, num)
	return buf.Bytes()
}
