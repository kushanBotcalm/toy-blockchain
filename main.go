package main

import (
	"fmt"
	"os"

	"toy-blockchain/blockchain"
	"toy-blockchain/cli"
	"toy-blockchain/storage"
	"toy-blockchain/wallet"
)

func main() {
	dbFile := "blockchain.json"
	walletFile := "wallet.json"
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

	// 3. Load or create the node wallet.
	nodeWallet, err := wallet.LoadOrCreateWallet(walletFile)
	if err != nil {
		fmt.Printf("Error loading wallet: %v\n", err)
		return
	}
	// Ensure wallet balance reflects the blockchain ledger
	nodeWallet.SyncBalance(bc.Ledger)

	// 4. Run the CLI
	appCLI := cli.NewCLI(bc, nodeWallet)
	appCLI.Run()

	// Update wallet balance from the ledger and persist changes.
	nodeWallet.SyncBalance(bc.Ledger)
	if err := wallet.SaveWallet(walletFile, nodeWallet); err != nil {
		fmt.Printf("Error saving wallet to file: %v\n", err)
	}

	// 5. Save the state back to disk after the CLI command finishes executing
	err = storage.SaveToFile(dbFile, bc)
	if err != nil {
		fmt.Printf("Error saving blockchain to file: %v\n", err)
	}
}
