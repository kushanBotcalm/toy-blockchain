package blockchain

import (
	"testing"
	"toy-blockchain/block"
	"toy-blockchain/mining"
	"toy-blockchain/transaction"
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
func TestHonestChainValidatesSuccessfully(t *testing.T) {
	// Initialize a new blockchain (starts with genesis block)
	difficulty := 2
	chain := NewBlockchain(difficulty)

	// Create and mine Block 1
	b1 := block.Block{
		Index:        1,
		Transactions: []transaction.Transaction{{Sender: "Alice", Receiver: "Bob", Amount: 10}},
		PreviousHash: chain.Blocks[len(chain.Blocks)-1].Hash,
		Nonce:        0,
	}
	mining.MineBlock(&b1, difficulty)
	chain.Blocks = append(chain.Blocks, b1)

	// Create and mine Block 2
	b2 := block.Block{
		Index:        2,
		Transactions: []transaction.Transaction{{Sender: "Bob", Receiver: "Charlie", Amount: 5}},
		PreviousHash: chain.Blocks[len(chain.Blocks)-1].Hash,
		Nonce:        0,
	}
	mining.MineBlock(&b2, difficulty)
	chain.Blocks = append(chain.Blocks, b2)

	// Validate the entire honest chain
	isValid, _ := chain.IsValidChain() // Replace with your actual validation method name if it differs

	if !isValid {
		t.Errorf("Expected honest chain of %d blocks to validate successfully, but validation failed", len(chain.Blocks))
	}
}
