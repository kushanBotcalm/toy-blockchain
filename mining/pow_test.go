package mining

import (
	"strings"
	"testing"

	"toy-blockchain/block"
	"toy-blockchain/transaction"
)

// TestMineBlockSatisfiesDifficulty verifies FR-5: A mined block's hash begins
// with at least N zeros, and ValidatePoW confirms the target and integrity[cite: 1].
func TestMineBlockSatisfiesDifficulty(t *testing.T) {
	difficulty := 3

	b := block.Block{
		Index:        1,
		Transactions: []transaction.Transaction{{Sender: "Alice", Receiver: "Bob", Amount: 10}},
		PreviousHash: "0000000000000000000000000000000000000000000000000000000000000000",
		Nonce:        0,
	}

	MineBlock(&b, difficulty)
	target := strings.Repeat("0", difficulty)
	if !strings.HasPrefix(b.Hash, target) {
		t.Errorf("Expected mined block hash to start with %d zeros ('%s'), but got '%s'", difficulty, target, b.Hash)
	}

	if !ValidatePoW(b, difficulty) {
		t.Errorf("ValidatePoW failed for block with hash %s and difficulty %d[cite: 1]", b.Hash, difficulty)
	}
}
