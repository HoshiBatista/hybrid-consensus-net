package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"log"
)

// Wallet хранит приватный и публичный ключи
type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  []byte
}

// NewWallet создает новую пару ключей
func NewWallet() *Wallet {
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Panic(err)
	}
	
	// Сериализуем публичный ключ в байты
	pubBytes, _ := x509.MarshalPKIXPublicKey(&private.PublicKey)
	
	return &Wallet{private, pubBytes}
}

// GetAddress возвращает строковый адрес (hex публичного ключа)
func (w *Wallet) GetAddress() string {
	return hex.EncodeToString(w.PublicKey)
}

// Помощник: Десериализация публичного ключа из байт
func PublicKeyFromBytes(pubBytes []byte) *ecdsa.PublicKey {
	pub, err := x509.ParsePKIXPublicKey(pubBytes)
	if err != nil {
		log.Fatal(err)
	}
	return pub.(*ecdsa.PublicKey)
}