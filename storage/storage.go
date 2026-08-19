package storage

import (
	"encoding/json"
	"os"
	"strings"
	"toy-blockchain/blockchain"
)

// BlockchainFileForListenAddr returns the chain file associated with a node port.
func BlockchainFileForListenAddr(listenAddr string) string {
	addr := strings.TrimSpace(listenAddr)
	addr = strings.TrimPrefix(addr, ":")
	addr = strings.TrimPrefix(addr, "http://")
	addr = strings.TrimPrefix(addr, "https://")
	addr = strings.TrimPrefix(addr, "localhost")
	addr = strings.Trim(addr, ":/")
	addr = strings.ReplaceAll(addr, ":", "_")
	addr = strings.ReplaceAll(addr, "/", "-")
	if addr == "" {
		addr = "8080"
	}
	return "blockchain_" + addr + ".json"
}

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
