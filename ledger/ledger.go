package ledger

import (
	"errors"
	"toy-blockchain/transaction"
)

// Ledger tracks account balances derived from transactions[cite: 1].
type Ledger struct {
	Balances map[string]float64
}

func NewLedger() *Ledger {
	return &Ledger{
		Balances: make(map[string]float64),
	}
}

// ApplyTransaction updates balances or rejects overspending/malformed TX[cite: 1].
func (l *Ledger) ApplyTransaction(tx transaction.Transaction) error {
	if tx.Amount <= 0 {
		return errors.New("transaction amount must be positive[cite: 1]")
	}

	// Faucet or coinbase bypasses balance check
	if tx.Sender != "FAUCET" {
		if l.Balances[tx.Sender] < tx.Amount {
			return errors.New("insufficient balance for transaction")
		}
		l.Balances[tx.Sender] -= tx.Amount
	}

	l.Balances[tx.Receiver] += tx.Amount
	return nil
}

// GetBalance returns the balance of a given account.
func (l *Ledger) GetBalance(address string) float64 {
	return l.Balances[address]
}
