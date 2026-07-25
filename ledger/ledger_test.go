package ledger

import (
	"testing"

	"toy-blockchain/transaction"
)

// TestOverspendingTransactionRejected verifies FR-4: Given an account whose balance is 100,
// when a transaction attempts to send 150, the transaction is rejected and balance is unchanged.
func TestOverspendingTransactionRejected(t *testing.T) {
	// Initialize ledger
	l := NewLedger()
	sender := "Alice"
	receiver := "Bob"

	// Given an account whose balance is 100
	l.Balances[sender] = 100.0

	// Precondition check
	if l.GetBalance(sender) != 100.0 {
		t.Fatalf("Precondition failed: expected balance to be 100.0, got %.2f", l.GetBalance(sender))
	}

	// When a transaction attempts to send 150 from that account
	tx := transaction.Transaction{
		Sender:   sender,
		Receiver: receiver,
		Amount:   150.0,
	}

	err := l.ApplyTransaction(tx)

	// Then the transaction is rejected
	if err == nil {
		t.Error("Expected overspending transaction to be rejected with an error, but it succeeded")
	}

	// And the account balance is unchanged
	balanceAfter := l.GetBalance(sender)
	if balanceAfter != 100.0 {
		t.Errorf("Expected balance to remain unchanged at 100.0, but got %.2f", balanceAfter)
	}
}
