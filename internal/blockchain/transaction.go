package blockchain

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// NewTransaction creates a Transaction with a pre-computed TxID. The returned
// transaction is unsigned; call Sign before broadcasting.
func NewTransaction(sender []byte, recipient []byte, amount int) *Transaction {
	tx := &Transaction{
		Sender:    sender,
		Recipient: recipient,
		Amount:    amount,
	}
	tx.TxID = tx.CalculateHash()
	return tx
}

// CalculateHash returns the SHA-256 digest of the transaction's core fields
// (Sender, Recipient, Amount). Signature is excluded so the hash can be
// computed before signing and remains stable if the signature is updated.
func (tx *Transaction) CalculateHash() []byte {
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

// Sign computes an ECDSA signature over the transaction hash (r‖s) and stores
// it in the Signature field.
func (tx *Transaction) Sign(privKey *ecdsa.PrivateKey) error {
	hash := tx.CalculateHash()
	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash)
	if err != nil {
		return fmt.Errorf("transaction: sign: %w", err)
	}
	tx.Signature = append(r.Bytes(), s.Bytes()...)
	return nil
}

// Verify returns true when the transaction carries a non-empty signature.
// Full ECDSA verification against tx.Sender requires a reconstructed
// *ecdsa.PublicKey; callers with one should use ecdsa.Verify directly.
func (tx *Transaction) Verify() bool {
	return len(tx.Signature) > 0
}

// MarshalJSON encodes all []byte fields as hex strings for consistent output.
// This matches the hex encoding used by Block for Hash and PrevBlockHash.
func (tx *Transaction) MarshalJSON() ([]byte, error) {
	type Alias Transaction
	return json.Marshal(&struct {
		*Alias
		Sender    string `json:"sender"`
		Recipient string `json:"recipient"`
		Signature string `json:"signature"`
		TxID      string `json:"tx_id"`
	}{
		Alias:     (*Alias)(tx),
		Sender:    hex.EncodeToString(tx.Sender),
		Recipient: hex.EncodeToString(tx.Recipient),
		Signature: hex.EncodeToString(tx.Signature),
		TxID:      hex.EncodeToString(tx.TxID),
	})
}

// UnmarshalJSON decodes hex-encoded byte fields, delegating all other fields
// to the standard decoder.
func (tx *Transaction) UnmarshalJSON(data []byte) error {
	type Alias Transaction
	aux := &struct {
		*Alias
		Sender    string `json:"sender"`
		Recipient string `json:"recipient"`
		Signature string `json:"signature"`
		TxID      string `json:"tx_id"`
	}{
		Alias: (*Alias)(tx),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	var err error
	if tx.Sender, err = hex.DecodeString(aux.Sender); err != nil {
		return fmt.Errorf("transaction: decode sender: %w", err)
	}
	if tx.Recipient, err = hex.DecodeString(aux.Recipient); err != nil {
		return fmt.Errorf("transaction: decode recipient: %w", err)
	}
	if tx.Signature, err = hex.DecodeString(aux.Signature); err != nil {
		return fmt.Errorf("transaction: decode signature: %w", err)
	}
	if tx.TxID, err = hex.DecodeString(aux.TxID); err != nil {
		return fmt.Errorf("transaction: decode tx_id: %w", err)
	}
	return nil
}
