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
	// mu protects Blocks and Ledger state during read/write and reorganisation
	mu sync.RWMutex `json:"-"`
}

func (bc *Blockchain) GetHeight() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return len(bc.Blocks) - 1
}

// GetLastBlock returns the head block safely
func (bc *Blockchain) GetLastBlock() block.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.Blocks[len(bc.Blocks)-1]
}

// GetBlocksCopy returns a copy of blocks from index 'from' to end, safe for gossiping
func (bc *Blockchain) GetBlocksCopy(from int) []block.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if from < 0 || from >= len(bc.Blocks) {
		return nil
	}
	result := make([]block.Block, len(bc.Blocks)-from)
	copy(result, bc.Blocks[from:])
	return result
}

// GetAllBlocksCopy returns a safe copy of entire chain
func (bc *Blockchain) GetAllBlocksCopy() []block.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
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

// Reorganise replaces the current chain with a competing chain if it's valid and longer.
// It reverts orphaned transactions back to the pending pool and rebuilds ledger state.
func (bc *Blockchain) Reorganise(newBlocks []block.Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// 1. Find common ancestor (last block in both chains)
	commonAncestorIdx := 0
	for i := 1; i < len(bc.Blocks) && i < len(newBlocks); i++ {
		if bc.Blocks[i].Hash == newBlocks[i].Hash {
			commonAncestorIdx = i
		} else {
			break
		}
	}

	// 2. Collect orphaned transactions from current chain beyond common ancestor
	var orphanedTxs []transaction.Transaction
	for i := len(bc.Blocks) - 1; i > commonAncestorIdx; i-- {
		orphanedTxs = append(orphanedTxs, bc.Blocks[i].Transactions...)
	}

	// 3. Rebuild ledger by replaying all blocks up to common ancestor
	bc.Ledger = ledger.NewLedger()
	for i := 0; i <= commonAncestorIdx; i++ {
		for _, tx := range bc.Blocks[i].Transactions {
			_ = bc.Ledger.ApplyTransaction(tx)
		}
	}

	// 4. Apply new blocks beyond common ancestor
	for i := commonAncestorIdx + 1; i < len(newBlocks); i++ {
		for _, tx := range newBlocks[i].Transactions {
			if err := bc.Ledger.ApplyTransaction(tx); err != nil {
				return fmt.Errorf("failed to apply tx from new chain: %w", err)
			}
		}
	}

	// 5. Replace chain with new blocks
	bc.Blocks = make([]block.Block, len(newBlocks))
	copy(bc.Blocks, newBlocks)

	// 6. Add orphaned transactions back to pending pool (only if still valid)
	bc.PendingMu.Lock()
	defer bc.PendingMu.Unlock()

	for _, tx := range orphanedTxs {
		// Check if sender still has funds
		if bc.Ledger.GetBalance(tx.Sender) >= tx.Amount {
			// Check if not already in pending pool
			if txID, err := transaction.ID(&tx); err == nil {
				if _, exists := bc.PendingIndex[txID]; !exists {
					bc.PendingTxPool = append(bc.PendingTxPool, tx)
					bc.PendingIndex[txID] = struct{}{}
				}
			}
		}
	}

	fmt.Printf("Chain reorganised: common ancestor at index %d, current chain height: %d\n", commonAncestorIdx, len(bc.Blocks)-1)
	return nil
}

// IsCompetingChainValid checks if a competing chain is valid and longer
func (bc *Blockchain) IsCompetingChainValid(competingBlocks []block.Block) (bool, error) {
	if len(competingBlocks) == 0 {
		return false, fmt.Errorf("empty competing chain")
	}

	// Competing chain must be longer
	if len(competingBlocks) <= len(bc.Blocks) {
		return false, nil
	}

	// Validate PoW for all blocks in competing chain
	for _, blk := range competingBlocks {
		if !mining.ValidatePoW(blk, bc.Difficulty) {
			return false, fmt.Errorf("invalid PoW in competing block")
		}
	}

	// Validate linkage (each block points to previous)
	for i := 1; i < len(competingBlocks); i++ {
		if competingBlocks[i].PreviousHash != competingBlocks[i-1].Hash {
			return false, fmt.Errorf("broken chain linkage at block %d", i)
		}
	}

	// Validate all transactions in competing chain
	tempLedger := ledger.NewLedger()
	for _, blk := range competingBlocks {
		for _, tx := range blk.Transactions {
			if !transaction.VerifyTransaction(&tx) {
				return false, fmt.Errorf("invalid tx signature in competing block")
			}
			if err := tempLedger.ApplyTransaction(tx); err != nil {
				return false, fmt.Errorf("invalid tx in competing block: %w", err)
			}
		}
	}

	fmt.Printf("Competing chain validated: length %d vs current %d\n", len(competingBlocks), len(bc.Blocks))
	return true, nil
}

// AppendBlock appends a block to the blockchain under lock
func (bc *Blockchain) AppendBlock(blk block.Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Apply transactions to ledger
	for _, tx := range blk.Transactions {
		if err := bc.Ledger.ApplyTransaction(tx); err != nil {
			return err
		}
		// Mark as seen in deduper
		if bc.Deduper != nil {
			if txID, err := transaction.ID(&tx); err == nil {
				bc.Deduper.SeenTransaction(txID)
			}
		}
	}

	bc.Blocks = append(bc.Blocks, blk)

	// Mark block as seen
	if bc.Deduper != nil && blk.Hash != "" {
		bc.Deduper.SeenBlock(blk.Hash)
	}

	return nil
}

// RemoveIncludedTransactions removes transactions that were included in a block from the pending pool
func (bc *Blockchain) RemoveIncludedTransactions(blockTxs []transaction.Transaction) {
	bc.PendingMu.Lock()
	defer bc.PendingMu.Unlock()

	if len(bc.PendingTxPool) == 0 {
		return
	}

	keep := make([]transaction.Transaction, 0, len(bc.PendingTxPool))
	for _, ptx := range bc.PendingTxPool {
		id, err := transaction.ID(&ptx)
		if err != nil {
			keep = append(keep, ptx)
			continue
		}
		included := false
		for _, btx := range blockTxs {
			bid, err := transaction.ID(&btx)
			if err == nil && bid == id {
				included = true
				break
			}
		}
		if !included {
			keep = append(keep, ptx)
		}
	}
	bc.PendingTxPool = keep
	bc.PendingIndex = make(map[string]struct{})
	for _, ptx := range bc.PendingTxPool {
		if id, err := transaction.ID(&ptx); err == nil {
			bc.PendingIndex[id] = struct{}{}
		}
	}
}
