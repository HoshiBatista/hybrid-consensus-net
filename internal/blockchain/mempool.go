package blockchain

import (
	"fmt"
	"sync"
)

// Mempool holds validated transactions that have been broadcast but not yet
// included in a block. It is safe for concurrent use.
type Mempool struct {
	mu   sync.RWMutex
	txs  map[string]*Transaction // keyed by hex TxID
}

// NewMempool returns an empty Mempool.
func NewMempool() *Mempool {
	return &Mempool{txs: make(map[string]*Transaction)}
}

// Add validates tx against state and, if valid, stores it in the mempool.
// Returns an error if the transaction signature is missing, the sender has
// insufficient balance, or the transaction is already present.
func (m *Mempool) Add(tx *Transaction, state *State) error {
	if !tx.Verify() {
		return fmt.Errorf("mempool: transaction %x has invalid or missing signature", tx.TxID)
	}

	senderAddr := fmt.Sprintf("%x", tx.Sender)
	if err := state.Debit(senderAddr, uint64(tx.Amount)); err != nil {
		return fmt.Errorf("mempool: %w", err)
	}
	// Debit was a speculative check — credit it back so state is unchanged.
	state.Credit(senderAddr, uint64(tx.Amount))

	key := fmt.Sprintf("%x", tx.TxID)

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.txs[key]; exists {
		return fmt.Errorf("mempool: transaction %s already pending", key)
	}
	m.txs[key] = tx
	return nil
}

// Flush atomically removes and returns up to limit pending transactions.
// Pass 0 to retrieve all pending transactions.
func (m *Mempool) Flush(limit int) []*Transaction {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]*Transaction, 0, len(m.txs))
	for key, tx := range m.txs {
		out = append(out, tx)
		delete(m.txs, key)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

// Pending returns a snapshot of all transactions currently in the mempool
// without removing them.
func (m *Mempool) Pending() []*Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]*Transaction, 0, len(m.txs))
	for _, tx := range m.txs {
		out = append(out, tx)
	}
	return out
}

// Size returns the number of pending transactions.
func (m *Mempool) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.txs)
}
