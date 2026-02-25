package blockchain

import (
	"time"
)

// Block представляет единицу данных в цепи
type Block struct {
	Timestamp     int64          `json:"timestamp"`
	Transactions  []*Transaction `json:"transactions"`
	PrevBlockHash []byte         `json:"prev_block_hash"`
	Hash          []byte         `json:"hash"`
	Nonce         int            `json:"nonce"`          // Для PoW
	Validator     string         `json:"validator"`      // Для PoS (адрес того, кто создал блок)
	Height        int            `json:"height"`         // Порядковый номер блока
}

// NewBlock создает новый блок (пока без вычисления хеша — это сделает консенсус)
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