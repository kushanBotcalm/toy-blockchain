package dedup

import (
	"encoding/json"
	"sync"
)

// deduperJSON is the JSON representation used for persistence.
type deduperJSON struct {
	Transactions []string `json:"transactions"`
	Blocks       []string `json:"blocks"`
}

// Deduper tracks processed transaction IDs and block hashes
// so the node does not process or gossip duplicate items.
// It is safe for concurrent access.
type Deduper struct {
	mu           sync.RWMutex
	transactions map[string]struct{}
	blocks       map[string]struct{}
}

// NewDeduper creates an initialized Deduper.
func NewDeduper() *Deduper {
	return &Deduper{
		transactions: make(map[string]struct{}),
		blocks:       make(map[string]struct{}),
	}
}

// MarshalJSON serializes the deduper state so it can be stored in the blockchain JSON.
func (d *Deduper) MarshalJSON() ([]byte, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	txs := make([]string, 0, len(d.transactions))
	for tx := range d.transactions {
		txs = append(txs, tx)
	}

	blocks := make([]string, 0, len(d.blocks))
	for block := range d.blocks {
		blocks = append(blocks, block)
	}

	return json.Marshal(deduperJSON{
		Transactions: txs,
		Blocks:       blocks,
	})
}

// UnmarshalJSON deserializes the deduper state from JSON.
func (d *Deduper) UnmarshalJSON(data []byte) error {
	var dj deduperJSON
	if err := json.Unmarshal(data, &dj); err != nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.transactions = make(map[string]struct{})
	for _, tx := range dj.Transactions {
		d.transactions[tx] = struct{}{}
	}

	d.blocks = make(map[string]struct{})
	for _, block := range dj.Blocks {
		d.blocks[block] = struct{}{}
	}

	return nil
}

// SeenTransaction reports whether the transaction ID has already been processed.
// If it has not been seen, it marks it as processed and returns false.
// If it has already been seen, it returns true.
func (d *Deduper) SeenTransaction(txID string) bool {
	d.mu.RLock()
	_, seen := d.transactions[txID]
	d.mu.RUnlock()
	if seen {
		return true
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	_, seen = d.transactions[txID]
	if seen {
		return true
	}
	d.transactions[txID] = struct{}{}
	return false
}

// SeenBlock reports whether the block hash has already been processed.
// If it has not been seen, it marks it as processed and returns false.
// If it has already been seen, it returns true.
func (d *Deduper) SeenBlock(hash string) bool {
	d.mu.RLock()
	_, seen := d.blocks[hash]
	d.mu.RUnlock()
	if seen {
		return true
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	_, seen = d.blocks[hash]
	if seen {
		return true
	}
	d.blocks[hash] = struct{}{}
	return false
}

// Reset clears all stored transaction IDs and block hashes.
func (d *Deduper) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.transactions = make(map[string]struct{})
	d.blocks = make(map[string]struct{})
}
