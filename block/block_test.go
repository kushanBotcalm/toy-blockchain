package block

import (
	"testing"
	"time"

	"toy-blockchain/transaction"
)

// TestCalculateHashDeterministic verifies FR-3: Hashing the same block twice
// must always yield the exact same result.
func TestCalculateHashDeterministic(v *testing.T) {
	txs := []transaction.Transaction{
		{Sender: "Alice", Receiver: "Bob", Amount: 50},
	}

	b := Block{
		Index:        1,
		Timestamp:    1718000000,
		Transactions: txs,
		PreviousHash: "0000000000000000000000000000000000000000000000000000000000000000",
		Nonce:        42,
	}

	hash1 := b.CalculateHash()
	hash2 := b.CalculateHash()

	if hash1 != hash2 {
		v.Errorf("Expected identical hashes, got %s and %s", hash1, hash2)
	}

	if len(hash1) != 64 {
		v.Errorf("Expected SHA-256 hex string length of 64, got %d", len(hash1))
	}
}

// TestCalculateHashExcludesHashField verifies that changing or setting
// the block's own Hash field does not affect the CalculateHash result.
func TestCalculateHashExcludesHashField(v *testing.T) {
	b1 := Block{
		Index:        1,
		Timestamp:    time.Now().Unix(),
		Transactions: []transaction.Transaction{},
		PreviousHash: "abc",
		Nonce:        100,
		Hash:         "initial_random_hash",
	}

	b2 := Block{
		Index:        1,
		Timestamp:    b1.Timestamp,
		Transactions: []transaction.Transaction{},
		PreviousHash: "abc",
		Nonce:        100,
		Hash:         "completely_different_hash",
	}

	if b1.CalculateHash() != b2.CalculateHash() {
		v.Error("CalculateHash should exclude the 'Hash' field itself, but output changed based on b.Hash")
	}
}

// TestCalculateHashSensitivity verifies that altering any structural field
// produces a completely different hash (crucial for tamper detection).
func TestCalculateHashSensitivity(v *testing.T) {
	baseBlock := Block{
		Index:        1,
		Timestamp:    1718000000,
		Transactions: []transaction.Transaction{{Sender: "Alice", Receiver: "Bob", Amount: 10}},
		PreviousHash: "prev_hash_abc",
		Nonce:        123,
	}

	baseHash := baseBlock.CalculateHash()

	// Test changing Index
	bIndex := baseBlock
	bIndex.Index = 2
	if bIndex.CalculateHash() == baseHash {
		v.Error("Hash did not change when Index was modified")
	}

	// Test changing Nonce
	bNonce := baseBlock
	bNonce.Nonce = 124
	if bNonce.CalculateHash() == baseHash {
		v.Error("Hash did not change when Nonce was modified")
	}

	// Test changing Transaction details
	bTx := baseBlock
	bTx.Transactions = []transaction.Transaction{{Sender: "Alice", Receiver: "Bob", Amount: 11}}
	if bTx.CalculateHash() == baseHash {
		v.Error("Hash did not change when Transaction amount was modified")
	}
}
