package blockchain

import (
	"testing"
)

// --- Balance / Credit ----------------------------------------------------

func TestBalance_UnknownAddress(t *testing.T) {
	s := NewState()
	if got := s.Balance("nobody"); got != 0 {
		t.Fatalf("Balance of unknown address = %d, want 0", got)
	}
}

func TestCredit_IncreasesBalance(t *testing.T) {
	s := NewState()
	s.Credit("alice", 100)
	if got := s.Balance("alice"); got != 100 {
		t.Fatalf("Balance after Credit(100) = %d, want 100", got)
	}
}

func TestCredit_Accumulates(t *testing.T) {
	s := NewState()
	s.Credit("alice", 60)
	s.Credit("alice", 40)
	if got := s.Balance("alice"); got != 100 {
		t.Fatalf("Balance after two Credits = %d, want 100", got)
	}
}

func TestCredit_MultipleAccounts(t *testing.T) {
	s := NewState()
	s.Credit("alice", 50)
	s.Credit("bob", 30)
	if a, b := s.Balance("alice"), s.Balance("bob"); a != 50 || b != 30 {
		t.Fatalf("alice=%d bob=%d, want 50 and 30", a, b)
	}
}

// --- Debit ---------------------------------------------------------------

func TestDebit_Success(t *testing.T) {
	s := NewState()
	s.Credit("alice", 100)
	if err := s.Debit("alice", 40); err != nil {
		t.Fatalf("unexpected Debit error: %v", err)
	}
	if got := s.Balance("alice"); got != 60 {
		t.Fatalf("Balance after Debit(40) = %d, want 60", got)
	}
}

func TestDebit_ExactBalance(t *testing.T) {
	s := NewState()
	s.Credit("alice", 50)
	if err := s.Debit("alice", 50); err != nil {
		t.Fatalf("Debit of exact balance should succeed: %v", err)
	}
	if got := s.Balance("alice"); got != 0 {
		t.Fatalf("Balance after full Debit = %d, want 0", got)
	}
}

func TestDebit_InsufficientBalance_ReturnsError(t *testing.T) {
	s := NewState()
	s.Credit("alice", 10)
	if err := s.Debit("alice", 20); err == nil {
		t.Fatal("expected error when debiting more than balance, got nil")
	}
}

func TestDebit_InsufficientBalance_StateUnchanged(t *testing.T) {
	s := NewState()
	s.Credit("alice", 10)
	_ = s.Debit("alice", 20)
	if got := s.Balance("alice"); got != 10 {
		t.Fatalf("balance must be unchanged after failed Debit, got %d", got)
	}
}

func TestDebit_UnknownAddress(t *testing.T) {
	s := NewState()
	if err := s.Debit("ghost", 1); err == nil {
		t.Fatal("expected error when debiting unknown address, got nil")
	}
}

// --- TotalStake ----------------------------------------------------------

func TestTotalStake_Empty(t *testing.T) {
	s := NewState()
	if got := s.TotalStake(); got != 0 {
		t.Fatalf("TotalStake of empty state = %d, want 0", got)
	}
}

func TestTotalStake_SumsAll(t *testing.T) {
	s := NewState()
	s.Credit("alice", 40)
	s.Credit("bob", 60)
	if got := s.TotalStake(); got != 100 {
		t.Fatalf("TotalStake = %d, want 100", got)
	}
}

func TestTotalStake_AfterDebit(t *testing.T) {
	s := NewState()
	s.Credit("alice", 100)
	_ = s.Debit("alice", 30)
	if got := s.TotalStake(); got != 70 {
		t.Fatalf("TotalStake after Debit = %d, want 70", got)
	}
}

// --- Validators ----------------------------------------------------------

func TestValidators_EmptyState(t *testing.T) {
	s := NewState()
	if vs := s.Validators(); len(vs) != 0 {
		t.Fatalf("Validators of empty state = %v, want []", vs)
	}
}

func TestValidators_ExcludesZeroStake(t *testing.T) {
	s := NewState()
	s.Credit("alice", 50)
	s.Credit("zero", 10)
	_ = s.Debit("zero", 10) // drain to zero
	vs := s.Validators()
	if len(vs) != 1 || vs[0].Address != "alice" {
		t.Fatalf("Validators = %v, want [{alice 50}]", vs)
	}
}

func TestValidators_SortedByAddress(t *testing.T) {
	s := NewState()
	// Credit in reverse alphabetical order to confirm sorting.
	s.Credit("charlie", 10)
	s.Credit("alice", 30)
	s.Credit("bob", 20)
	vs := s.Validators()
	if len(vs) != 3 {
		t.Fatalf("expected 3 validators, got %d", len(vs))
	}
	want := []string{"alice", "bob", "charlie"}
	for i, w := range want {
		if vs[i].Address != w {
			t.Fatalf("vs[%d].Address = %q, want %q", i, vs[i].Address, w)
		}
	}
}

func TestValidators_StakesMatch(t *testing.T) {
	s := NewState()
	s.Credit("alice", 75)
	s.Credit("bob", 25)
	vs := s.Validators()
	stakes := map[string]uint64{}
	for _, v := range vs {
		stakes[v.Address] = v.Stake
	}
	if stakes["alice"] != 75 || stakes["bob"] != 25 {
		t.Fatalf("stake mismatch: %v", stakes)
	}
}

// --- Concurrency smoke test ----------------------------------------------

func TestState_ConcurrentAccess(t *testing.T) {
	s := NewState()
	s.Credit("alice", 10_000)

	done := make(chan struct{})
	// 100 concurrent goroutines each debit 1 then credit 1 (net-zero).
	for i := 0; i < 100; i++ {
		go func() {
			_ = s.Debit("alice", 1)
			s.Credit("alice", 1)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 100; i++ {
		<-done
	}
	// Balance must still be 10_000 because every debit was followed by a credit.
	if got := s.Balance("alice"); got != 10_000 {
		t.Fatalf("balance after concurrent ops = %d, want 10000", got)
	}
}
