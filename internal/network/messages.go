package network

import (
	"encoding/json"
	"go-hybrid-blockchain/internal/blockchain"
)

// Типы сообщений
const (
	TypeTx    = "new_tx"    // Новая транзакция
	TypeBlock = "new_block" // Новый блок
	TypeGetChain = "get_chain" // Запрос всей цепочки
	TypeChain    = "chain"     // Передача всей цепочки
)

// Message — общая структура сетевого сообщения
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"` // Используем RawMessage!
}

// GetChainRequest — для запроса цепочки у пира
type GetChainRequest struct {
	FromAddress string `json:"from_address"`
}

// ChainResponse — для передачи цепочки
type ChainResponse struct {
	Blocks []*blockchain.Block `json:"blocks"`
}