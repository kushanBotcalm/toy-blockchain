package blockchain

import (
	"testing"
)

// TestGenesisBlockCreation verifies FR-2: A freshly initialized chain contains
// exactly one block at height 0, and its previous-hash equals the fixed genesis value.
func TestGenesisBlockCreation(t *testing.T) {
	// Initialize a new blockchain instance (adjust constructor name if it differs in your code, e.g., NewBlockchain())
	chain := NewBlockchain(3)

	// Check that the chain contains exactly one block
	if len(chain.Blocks) != 1 {
		t.Fatalf("Expected exactly 1 block in a new chain, got %d", len(chain.Blocks))
	}

	genesisBlock := chain.Blocks[0]

	// Check that the height/index is 0
	if genesisBlock.Index != 0 {
		t.Errorf("Expected genesis block index to be 0, got %d", genesisBlock.Index)
	}

	// Check that the previous-hash equals the fixed genesis value
	// (matching your project's chosen fixed string, e.g., 64 zeros)
	expectedPrevHash := "0000000000000000000000000000000000000000000000000000000000000000"
	if genesisBlock.PreviousHash != expectedPrevHash {
		t.Errorf("Expected genesis PreviousHash to be %s, got %s", expectedPrevHash, genesisBlock.PreviousHash)
	}
}
