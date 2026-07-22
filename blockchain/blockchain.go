package blockchain

import (
	"errors"
	"toy-blockchain/block"
	"toy-blockchain/ledger"
	"toy-blockchain/mining"
	"toy-blockchain/transaction"
)

const GenesisPreviousHash = "0000000000000000000000000000000000000000000000000000000000000000"

type Blockchain struct {
	Blocks        []block.Block             `json:"blocks"`
	PendingTxPool []transaction.Transaction `json:"pending_transactions"`
	Difficulty    int                       `json:"difficulty"`
	Ledger        *ledger.Ledger            `json:"-"`
}

func NewBlockchain(difficulty int) *Blockchain {
	bc := &Blockchain{
		Blocks:        []block.Block{},
		PendingTxPool: []transaction.Transaction{},
		Difficulty:    difficulty,
		Ledger:        ledger.NewLedger(),
	}

	// Create Genesis Block[cite: 1]
	genesisBlock := block.Block{
		Index:        0,
		Timestamp:    0,
		Transactions: []transaction.Transaction{},
		PreviousHash: GenesisPreviousHash,
		Nonce:        0,
	}
	genesisBlock.Hash = genesisBlock.CalculateHash()
	bc.Blocks = append(bc.Blocks, genesisBlock)

	return bc
}

func (bc *Blockchain) AddTransaction(tx transaction.Transaction) error {
	if tx.Amount <= 0 {
		return errors.New("transaction amount must be non-positive[cite: 1]")
	}
	if tx.Sender != "FAUCET" {
		if bc.Ledger.GetBalance(tx.Sender) < tx.Amount {
			return errors.New("sender has insufficient balance[cite: 1]")
		}
	}
	bc.PendingTxPool = append(bc.PendingTxPool, tx)
	return nil
}

func (bc *Blockchain) MinePendingTransactions(minerAddress string) block.Block {
	rewardTx := transaction.Transaction{
		Sender:   "FAUCET",
		Receiver: minerAddress,
		Amount:   10.0,
	}
	allTx := append([]transaction.Transaction{rewardTx}, bc.PendingTxPool...)

	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := block.Block{
		Index:        prevBlock.Index + 1,
		Transactions: allTx,
		PreviousHash: prevBlock.Hash,
		Nonce:        0,
	}

	mining.MineBlock(&newBlock, bc.Difficulty)

	// Apply transactions to ledger
	for _, tx := range allTx {
		_ = bc.Ledger.ApplyTransaction(tx)
	}

	bc.Blocks = append(bc.Blocks, newBlock)
	bc.PendingTxPool = []transaction.Transaction{}
	return newBlock
}

// IsValidChain validates entire chain integrity and tamper detection[cite: 1].
func (bc *Blockchain) IsValidChain() (bool, int) {
	for i := 1; i < len(bc.Blocks); i++ {
		current := bc.Blocks[i]
		prev := bc.Blocks[i-1]

		if current.Hash != current.CalculateHash() {
			return false, current.Index
		}
		if current.PreviousHash != prev.Hash {
			return false, current.Index
		}
		if !mining.ValidatePoW(current, bc.Difficulty) {
			return false, current.Index
		}
	}
	return true, -1
}
