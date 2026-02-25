package blockchain

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
)

// Transaction представляет собой перевод средств
type Transaction struct {
	Sender    []byte `json:"sender"`    // Публичный ключ отправителя
	Recipient []byte `json:"recipient"` // Публичный ключ получателя
	Amount    int    `json:"amount"`
	Signature []byte `json:"signature"` // Цифровая подпись ECDSA
	TxID      []byte `json:"tx_id"`     // Хеш транзакции
}

// NewTransaction создает новую транзакцию
func NewTransaction(sender []byte, recipient []byte, amount int) *Transaction {
	tx := &Transaction{
		Sender:    sender,
		Recipient: recipient,
		Amount:    amount,
	}
	tx.TxID = tx.CalculateHash()

	return tx
}

// CalculateHash вычисляет SHA-256 хеш транзакции (без учета подписи)
func (tx *Transaction) CalculateHash() []byte {
	// Собираем данные для хеширования (без подписи)
	data, _ := json.Marshal(struct {
		Sender    []byte
		Recipient []byte
		Amount    int
	}{
		Sender:    tx.Sender,
		Recipient: tx.Recipient,
		Amount:    tx.Amount,
	})

	hash := sha256.Sum256(data)

	return hash[:]
}

// Sign подписывает транзакцию приватным ключом
func (tx *Transaction) Sign(privKey *ecdsa.PrivateKey) error {
	hash := tx.CalculateHash()
	
	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash)
	if err != nil {
		return err
	}

	// Объединяем R и S в одну подпись
	signature := append(r.Bytes(), s.Bytes()...)
	tx.Signature = signature

	return nil
}

// Verify проверяет подпись транзакции
func (tx *Transaction) Verify() bool {
	if len(tx.Signature) == 0 {
		return false
	}

	// Восстанавливаем публичный ключ отправителя (в реальности нужно его десериализовать)
	// Для упрощения здесь логика подразумевает, что tx.Sender — это сериализованный публичный ключ
	// В учебном проекте мы будем использовать вспомогательные функции для парсинга ключей
	
	// Это заглушка
	return true 
}