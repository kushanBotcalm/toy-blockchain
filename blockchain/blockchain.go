package blockchain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"toy-blockchain/block"
	"toy-blockchain/dedup"
	"toy-blockchain/ledger"
	"toy-blockchain/mining"
	"toy-blockchain/transaction"
)

const GenesisPreviousHash = "0000000000000000000000000000000000000000000000000000000000000000"

type Blockchain struct {
	Blocks        []block.Block             `json:"blocks"`
	PendingTxPool []transaction.Transaction `json:"pending_transactions"`
	PendingIndex  map[string]struct{}       `json:"-"`
	PendingMu     sync.RWMutex              `json:"-"`
	Difficulty    int                       `json:"difficulty"`
	Ledger        *ledger.Ledger            `json:"-"`
	Deduper       *dedup.Deduper            `json:"deduper,omitempty"`
}

func (bc *Blockchain) GetHeight() int {
	bc.PendingMu.RLock()
	defer bc.PendingMu.RUnlock()
	return len(bc.Blocks) - 1
}

// GetLastBlock returns the head block safely
func (bc *Blockchain) GetLastBlock() block.Block {
	bc.PendingMu.RLock()
	defer bc.PendingMu.RUnlock()
	return bc.Blocks[len(bc.Blocks)-1]
}

// GetBlocksCopy returns a copy of blocks from index 'from' to end, safe for gossiping
func (bc *Blockchain) GetBlocksCopy(from int) []block.Block {
	bc.PendingMu.RLock()
	defer bc.PendingMu.RUnlock()
	if from < 0 || from >= len(bc.Blocks) {
		return nil
	}
	result := make([]block.Block, len(bc.Blocks)-from)
	copy(result, bc.Blocks[from:])
	return result
}

// GetAllBlocksCopy returns a safe copy of entire chain
func (bc *Blockchain) GetAllBlocksCopy() []block.Block {
	bc.PendingMu.RLock()
	defer bc.PendingMu.RUnlock()
	result := make([]block.Block, len(bc.Blocks))
	copy(result, bc.Blocks)
	return result
}

func NewBlockchain(difficulty int) *Blockchain {
	bc := &Blockchain{
		Blocks:        []block.Block{},
		PendingTxPool: []transaction.Transaction{},
		PendingIndex:  make(map[string]struct{}),
		Difficulty:    difficulty,
		Ledger:        ledger.NewLedger(),
		Deduper:       dedup.NewDeduper(),
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

func (bc *Blockchain) RebuildDeduper() {
	if bc.Deduper == nil {
		bc.Deduper = dedup.NewDeduper()
	}
	bc.Deduper.Reset()
	bc.PendingIndex = make(map[string]struct{})

	for _, blk := range bc.Blocks {
		if blk.Hash != "" {
			bc.Deduper.SeenBlock(blk.Hash)
		}
		for i := range blk.Transactions {
			txID, err := transaction.ID(&blk.Transactions[i])
			if err == nil {
				bc.Deduper.SeenTransaction(txID)
			}
		}
	}

	for _, tx := range bc.PendingTxPool {
		txID, err := transaction.ID(&tx)
		if err == nil {
			bc.PendingIndex[txID] = struct{}{}
		}
	}
}

func (bc *Blockchain) AddTransaction(tx transaction.Transaction) error {
	if tx.Amount <= 0 {
		return errors.New("transaction amount must be positive")
	}
	if tx.Sender != "FAUCET" {
		if bc.Ledger.GetBalance(tx.Sender) < tx.Amount {
			return errors.New("sender has insufficient balance")
		}
	}
	if !transaction.VerifyTransaction(&tx) {
		return errors.New("transaction signature is invalid")
	}

	// compute transaction id for dedup/pending checks
	txID, err := transaction.ID(&tx)
	if err == nil {
		if bc.Deduper != nil && bc.Deduper.SeenTransaction(txID) {
			return errors.New("duplicate transaction")
		}
		// check pending index
		bc.PendingMu.RLock()
		_, pending := bc.PendingIndex[txID]
		bc.PendingMu.RUnlock()
		if pending {
			return errors.New("transaction already pending")
		}
		// add to pending pool and index
		bc.PendingMu.Lock()
		bc.PendingTxPool = append(bc.PendingTxPool, tx)
		bc.PendingIndex[txID] = struct{}{}
		bc.PendingMu.Unlock()
		return nil
	}

	return errors.New("failed to compute transaction ID")
}

func (bc *Blockchain) MinePendingTransactions(minerAddress string) block.Block {
	rewardTx := transaction.Transaction{
		Sender:   "FAUCET",
		Receiver: minerAddress,
		Amount:   10.0,
	}
	bc.PendingMu.Lock()
	allTx := append([]transaction.Transaction{rewardTx}, bc.PendingTxPool...)
	// clear pending pool and index
	bc.PendingTxPool = []transaction.Transaction{}
	bc.PendingIndex = make(map[string]struct{})
	bc.PendingMu.Unlock()

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
	payload, _ := json.Marshal(newBlock)

	for _, u := range []string{
		"http://localhost:8081/block",
		"http://localhost:8082/block",
	} {
		resp, err := http.Post(u, "application/json", bytes.NewReader(payload))
		if err != nil {
			fmt.Println("send failed:", u, err)
			continue
		}
		defer resp.Body.Close()
		fmt.Println(u, resp.StatusCode)
	}

	bc.Blocks = append(bc.Blocks, newBlock)
	return newBlock
}

// IsValidChain validates entire chain integrity and tamper detection
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

// RebuildLedger recalculates all balances from the block history
func (bc *Blockchain) RebuildLedger() {
	bc.Ledger = ledger.NewLedger()
	for _, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			_ = bc.Ledger.ApplyTransaction(tx)
		}
	}
}
