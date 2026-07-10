package blockchain

import (
	"toy-blockchain/transaction"

	"toy-blockchain/block"
)

// The fixed previous hash value for genesis block.
const GenesisPreviousHash = "0000000000000000000000000000000000000000000000000000000000000000"

// Blockchain represents the complete chain of blocks.
type Blockchain struct {
	Blocks []block.Block `json:"blocks"`
}

// NewBlockchain creates a new blockchain
// with only the genesis block.
func NewBlockchain() *Blockchain {

	genesisBlock := block.Block{
		Index:        0,
		Timestamp:    0,
		Transactions: []transaction.Transaction{},
		PreviousHash: GenesisPreviousHash,
		Nonce:        0,
		Hash:         "",
	}

	return &Blockchain{
		Blocks: []block.Block{
			genesisBlock,
		},
	}
}
