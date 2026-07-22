package main

import (
	"fmt"
	"os"

	"toy-blockchain/blockchain"
	"toy-blockchain/cli"
	"toy-blockchain/storage"
)

func main() {
	dbFile := "blockchain.json"
	difficulty := 2

	// 1. Initialize a fresh blockchain (creates genesis block if needed)
	bc := blockchain.NewBlockchain(difficulty)

	// 2. Load existing state from disk if the file exists
	if _, err := os.Stat(dbFile); err == nil {
		err = storage.LoadFromFile(dbFile, bc)
		if err != nil {
			fmt.Printf("Error loading blockchain from file: %v\n", err)
			return
		}
		// Rebuild the ledger because ledger balances are skipped in JSON
		bc.RebuildLedger()
	} else {
		// If no file exists yet, save the initial genesis state
		_ = storage.SaveToFile(dbFile, bc)
	}

	// 3. Run the CLI
	appCLI := cli.NewCLI(bc)
	appCLI.Run()

	// 4. Save the state back to disk after the CLI command finishes executing
	err := storage.SaveToFile(dbFile, bc)
	if err != nil {
		fmt.Printf("Error saving blockchain to file: %v\n", err)
	}
}
