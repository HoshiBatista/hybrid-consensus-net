package blockchain

import (
	"fmt"
	"sort"
	"sync"
)

// ValidatorInfo bundles an address with its current stake.
// Returned by State.Validators so the PoS engine does not need direct
// access to the internal balance map.
type ValidatorInfo struct {
	Address string
	Stake   uint64
}

// State is the canonical account-balance ledger for the network.
// It serves two roles:
//
//  1. Transaction validation — Debit enforces that a sender cannot spend
//     more than their current balance.
//
//  2. PoS validator selection — Validators returns eligible block producers
//     sorted by address, giving every node a deterministic, identical view of
//     who may mint the next block.
//
// All exported methods are safe for concurrent use.
type State struct {
	mu       sync.RWMutex
	balances map[string]uint64
}

// NewState creates an empty State with no accounts.
func NewState() *State {
	return &State{balances: make(map[string]uint64)}
}

// Balance returns the current balance of address, or 0 if the address is
// unknown.
func (s *State) Balance(address string) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.balances[address]
}

// Credit adds amount to address's balance. It is safe to credit an address
// that has never appeared in the state before.
func (s *State) Credit(address string, amount uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.balances[address] += amount
}

// Debit subtracts amount from address's balance. It returns an error when the
// address's balance is insufficient, leaving the state completely unchanged.
func (s *State) Debit(address string, amount uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.balances[address] < amount {
		return fmt.Errorf("state: insufficient balance for %q: have %d, need %d",
			address, s.balances[address], amount)
	}
	s.balances[address] -= amount
	return nil
}

// TotalStake returns the sum of all account balances. This is the denominator
// used by the PoS weighted-lottery to map a seed into a validator slot.
func (s *State) TotalStake() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var total uint64
	for _, v := range s.balances {
		total += v
	}
	return total
}

// Validators returns every account with a non-zero balance as a slice of
// ValidatorInfo, sorted in ascending order by address.
//
// The stable sort is critical: PoS validator selection is deterministic only
// when every node iterates the validator set in the same order. Map iteration
// in Go is intentionally random, so we always sort before returning.
func (s *State) Validators() []ValidatorInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]ValidatorInfo, 0, len(s.balances))
	for addr, stake := range s.balances {
		if stake > 0 {
			out = append(out, ValidatorInfo{Address: addr, Stake: stake})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Address < out[j].Address
	})
	return out
}
