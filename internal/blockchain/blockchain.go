package blockchain

import (
	"fmt"
	"log"

	"go.etcd.io/bbolt"
)

const blocksBucket = "blocks"

// Blockchain manages the persistent chain of blocks. Tip holds the hash of
// the most recent block; all blocks are stored in a bbolt database keyed by
// their hash.
type Blockchain struct {
	Tip []byte
	DB  *bbolt.DB
}

// CreateBlockchain opens (or creates) the bbolt database at dbPath and
// ensures a genesis block exists. The genesis block uses 32 zero bytes as a
// fixed sentinel hash so every fresh chain starts from the same known state.
func CreateBlockchain(dbPath string) *Blockchain {
	var tip []byte

	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Panicf("blockchain: open db %q: %v", dbPath, err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		if b == nil {
			fmt.Println("No existing blockchain found. Creating genesis block...")

			genesis := NewBlock([]*Transaction{}, []byte{}, 0)
			genesis.Hash = make([]byte, 32) // 32 zero bytes as genesis sentinel

			bucket, createErr := tx.CreateBucket([]byte(blocksBucket))
			if createErr != nil {
				return fmt.Errorf("create bucket: %w", createErr)
			}
			if err := bucket.Put(genesis.Hash, genesis.Serialize()); err != nil {
				return fmt.Errorf("store genesis block: %w", err)
			}
			if err := bucket.Put([]byte("l"), genesis.Hash); err != nil {
				return fmt.Errorf("store chain tip: %w", err)
			}
			tip = genesis.Hash
		} else {
			tip = b.Get([]byte("l"))
		}

		return nil
	})
	if err != nil {
		log.Panicf("blockchain: init: %v", err)
	}

	return &Blockchain{Tip: tip, DB: db}
}

// AddBlock persists block to the database and advances the chain tip.
func (bc *Blockchain) AddBlock(block *Block) {
	err := bc.DB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		if err := b.Put(block.Hash, block.Serialize()); err != nil {
			return fmt.Errorf("store block: %w", err)
		}
		if err := b.Put([]byte("l"), block.Hash); err != nil {
			return fmt.Errorf("update tip: %w", err)
		}

		bc.Tip = block.Hash
		return nil
	})
	if err != nil {
		log.Panicf("blockchain: AddBlock: %v", err)
	}
}

// ValidateChain iterates every block from genesis to tip and verifies two
// invariants for each block:
//
//  1. The block's Hash is non-empty (it was finalised by a consensus engine).
//  2. The block's PrevBlockHash exactly matches the preceding block's Hash,
//     ensuring the chain is unbroken and no block has been silently replaced.
//
// An optional validateFn can be supplied to add consensus-specific checks
// (e.g. PoW target, PoS validator). Pass nil to skip that check.
// Returns true only when all blocks satisfy both invariants.
func (bc *Blockchain) ValidateChain(validateFn func(*Block) bool) bool {
	blocks := bc.GetAllBlocks()
	if len(blocks) == 0 {
		return true
	}

	// Genesis block: PrevBlockHash must be empty (all-zero sentinel).
	if !blocks[0].IsValid(blocks[0].PrevBlockHash) {
		return false
	}
	if validateFn != nil && !validateFn(blocks[0]) {
		return false
	}

	for i := 1; i < len(blocks); i++ {
		if !blocks[i].IsValid(blocks[i-1].Hash) {
			return false
		}
		if validateFn != nil && !validateFn(blocks[i]) {
			return false
		}
	}
	return true
}

// GetAllBlocks iterates from the tip back to genesis and returns the full
// chain ordered oldest-first. The loop terminates naturally when it reaches
// the genesis block whose PrevBlockHash is an empty slice.
func (bc *Blockchain) GetAllBlocks() []*Block {
	var blocks []*Block

	err := bc.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		currentHash := b.Get([]byte("l"))

		for len(currentHash) > 0 {
			data := b.Get(currentHash)
			if data == nil {
				break
			}
			block := DeserializeBlock(data)
			if block == nil {
				break
			}
			blocks = append([]*Block{block}, blocks...) // prepend → oldest-first result
			currentHash = block.PrevBlockHash
		}

		return nil
	})
	if err != nil {
		log.Printf("blockchain: GetAllBlocks: %v", err)
	}

	return blocks
}
