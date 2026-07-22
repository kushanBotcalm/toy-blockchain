package storage

import (
	"encoding/json"
	"os"
	"toy-blockchain/blockchain"
)

// SaveToFile saves the blockchain state as JSON[cite: 1].
func SaveToFile(filename string, bc *blockchain.Blockchain) error {
	data, err := json.MarshalIndent(bc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// LoadFromFile loads the blockchain state from disk.
func LoadFromFile(filename string, bc *blockchain.Blockchain) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, bc)
}
