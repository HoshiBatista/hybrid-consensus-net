package blockchain

import (
	"encoding/json"
	"fmt"
	"log"

	"go.etcd.io/bbolt"
)

const dbFile = "blockchain.db"
const blocksBucket = "blocks"

// Blockchain управляет связью с базой данных и хранит хеш последнего блока
type Blockchain struct {
	Tip []byte   // Хеш последнего блока в цепочке
	DB  *bbolt.DB
}

// CreateBlockchain открывает БД и создает Genesis блок, если цепь пуста
func CreateBlockchain(dbFile string) *Blockchain {
	var tip []byte
	db, err := bbolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic("Could not open bbolt db:", err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		// Если бакета (таблицы) нет, создаем её и Genesis блок
		if b == nil {
			fmt.Println("No existing blockchain found. Creating genesis...")
			genesis := NewBlock([]*Transaction{}, []byte{}, 0)
			// Задаем начальный хеш вручную для Genesis блока
			genesis.Hash = []byte("00000000000000000000000000000000") 

			bucket, err := tx.CreateBucket([]byte(blocksBucket))
			if err != nil {
				return err
			}

			// Сохраняем блок: ключ - его хеш, значение - сериализованные данные
			err = bucket.Put(genesis.Hash, genesis.Serialize())
			// "l" - специальный ключ, хранящий хеш последнего блока (Last)
			err = bucket.Put([]byte("l"), genesis.Hash)
			tip = genesis.Hash
		} else {
			// Если база есть, просто читаем хеш последнего блока
			tip = b.Get([]byte("l"))
		}

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	return &Blockchain{tip, db}
}

// AddBlock сохраняет новый блок в базу данных
func (bc *Blockchain) AddBlock(block *Block) {
	err := bc.DB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		// Сохраняем сам блок
		err := b.Put(block.Hash, block.Serialize())
		if err != nil {
			return err
		}

		// Обновляем указатель на последний блок
		err = b.Put([]byte("l"), block.Hash)
		if err != nil {
			return err
		}

		bc.Tip = block.Hash
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}

// Serialize переводит блок в JSON-байты (для хранения в БД)
func (b *Block) Serialize() []byte {
	data, _ := json.Marshal(b)
	return data
}

// DeserializeBlock восстанавливает блок из байтов
func DeserializeBlock(d []byte) *Block {
	var block Block
	err := json.Unmarshal(d, &block)
	if err != nil {
		return nil
	}
	return &block
}