package consensus

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"go-hybrid-blockchain/internal/blockchain" 
)

// Difficulty — сложность майнинга (количество ведущих нулей в шестнадцатеричном хеше)
// В реальных сетях она динамическая, для курсовой зафиксируем ее.
const Difficulty = 4 

type ProofOfWork struct {
	Block  *blockchain.Block
	Target *big.Int
}

// NewProofOfWork создает новый объект PoW для блока
func NewProofOfWork(b *blockchain.Block) *ProofOfWork {
	target := big.NewInt(1)
	// Сдвигаем 1 на (256 - сложность) бит. 
	// Чем больше Difficulty, тем меньше Target и тем сложнее найти хеш.
	target.Lsh(target, uint(256-Difficulty*4)) 

	return &ProofOfWork{b, target}
}

// prepareData объединяет данные блока с nonce для хеширования
func (pow *ProofOfWork) prepareData(nonce int) []byte {
	data := bytes.Join(
		[][]byte{
			pow.Block.PrevBlockHash,
			pow.prepareTransactionsHash(), // Хеш всех транзакций блока
			IntToHex(pow.Block.Timestamp),
			IntToHex(int64(Difficulty)),
			IntToHex(int64(nonce)),
		},
		[]byte{},
	)
	return data
}

// Run — основной цикл майнинга
func (pow *ProofOfWork) Run() (int, []byte) {
	var hashInt big.Int
	var hash [32]byte
	nonce := 0

	fmt.Printf("Mining a new block...")
	for nonce < math.MaxInt64 {
		data := pow.prepareData(nonce)
		hash = sha256.Sum256(data)
		
		hashInt.SetBytes(hash[:])

		// Если hashInt < target, то мы нашли решение
		if hashInt.Cmp(pow.Target) == -1 {
			fmt.Printf("\rSuccess! Hash: %x\n", hash)
			break
		} else {
			nonce++
		}
	}

	return nonce, hash[:]
}

// Validate проверяет, валиден ли хеш блока согласно сложности
func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int

	data := pow.prepareData(pow.Block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])

	return hashInt.Cmp(pow.Target) == -1
}

// Вспомогательная функция для хеширования транзакций в блоке
func (pow *ProofOfWork) prepareTransactionsHash() []byte {
	var txHashes [][]byte
	for _, tx := range pow.Block.Transactions {
		txHashes = append(txHashes, tx.TxID)
	}
	txHash := sha256.Sum256(bytes.Join(txHashes, []byte{}))
	return txHash[:]
}

// IntToHex конвертирует int64 в байты
func IntToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		return nil
	}
	return buff.Bytes()
}