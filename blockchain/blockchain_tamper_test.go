package blockchain

import (
	"testing"

	"toy-blockchain/block"
	"toy-blockchain/mining"
	"toy-blockchain/transaction"
)

// TestTamperDetection verifies FR-6: Given a valid chain, if a transaction inside
// an earlier block is modified, validation fails and identifies the first offending block index.
func TestTamperDetection(t *testing.T) {
	// 1. Initialize chain and mine an honest chain of several blocks
	difficulty := 2
	chain := NewBlockchain(difficulty)

	// Block 1
	b1 := block.Block{
		Index:        1,
		Transactions: []transaction.Transaction{{Sender: "Alice", Receiver: "Bob", Amount: 50}},
		PreviousHash: chain.Blocks[len(chain.Blocks)-1].Hash,
		Nonce:        0,
	}
	mining.MineBlock(&b1, difficulty)
	chain.Blocks = append(chain.Blocks, b1)

	// Block 2
	b2 := block.Block{
		Index:        2,
		Transactions: []transaction.Transaction{{Sender: "Bob", Receiver: "Charlie", Amount: 20}},
		PreviousHash: chain.Blocks[len(chain.Blocks)-1].Hash,
		Nonce:        0,
	}
	mining.MineBlock(&b2, difficulty)
	chain.Blocks = append(chain.Blocks, b2)

	// Verify the honest chain passes initially
	valid, _ := chain.IsValidChain()
	if !valid {
		t.Fatal("Precondition failed: Honest chain should be valid before tampering")
	}

	// 2. Tamper with a transaction inside an earlier block (Block 1)
	chain.Blocks[1].Transactions[0].Amount = 999 // Maliciously change amount

	// 3. Validate the tampered chain
	valid, offendingIndex := chain.IsValidChain()

	// 4. Assertions: Validation must fail and point directly to Block 1
	if valid {
		t.Error("Expected chain validation to FAIL after tampering with an earlier block, but it passed")
	}

	expectedOffendingIndex := 1
	if offendingIndex != expectedOffendingIndex {
		t.Errorf("Expected tamper detection to identify block index %d, got %d", expectedOffendingIndex, offendingIndex)
	}
}
