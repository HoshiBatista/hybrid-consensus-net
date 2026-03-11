package blockchain

import (
	"crypto/sha256"
	"fmt"
	"time"

	"go.etcd.io/bbolt"
)

// Простейший PoS: валидатор просто подписывает блок своим именем
// В реальном PoS нужно проверять баланс (stake)
func CreatePoSBlock(prevHash []byte, height int, validator string, txs []*Transaction) *Block {
	block := NewBlock(txs, prevHash, height)
	block.Validator = validator
	
	// В PoS нет nonce, мы просто хешируем данные один раз
	data := fmt.Sprintf("%x%d%s", prevHash, block.Timestamp, validator)
	hash := sha256.Sum256([]byte(data))
	block.Hash = hash[:]
	
	return block
}

// MineBlockPoS имитирует создание блока через Proof-of-Stake
func (bc *Blockchain) MineBlockPoS(transactions []*Transaction, validator string) *Block {
	var lastHash []byte
	var lastHeight int

	bc.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte("l"))
		lastBlock := DeserializeBlock(b.Get(lastHash))
		lastHeight = lastBlock.Height
		return nil
	})

	// В PoS мы не ищем nonce, а просто создаем блок
	newBlock := NewBlock(transactions, lastHash, lastHeight+1)
	newBlock.Validator = validator
	newBlock.Timestamp = time.Now().Unix()

	// Хеш создается просто из данных блока и имени валидатора
	data := fmt.Sprintf("%x%d%s%d", newBlock.PrevBlockHash, newBlock.Timestamp, validator, newBlock.Height)
	hash := sha256.Sum256([]byte(data))
	newBlock.Hash = hash[:]

	bc.AddBlock(newBlock)
	return newBlock
}