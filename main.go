package main

import (
	"flag"
	"fmt"
	"os"

	"toy-blockchain/blockchain"
	"toy-blockchain/cli"
	"toy-blockchain/node"
	"toy-blockchain/storage"
	"toy-blockchain/wallet"
)

func main() {
	mode := flag.String("mode", "cli", "Run mode: cli or serve")
	addr := flag.String("addr", ":8080", "HTTP listen address for serve mode")
	configPath := flag.String("config", "node_config.json", "Node configuration file path")
	flag.Parse()

	dbFile := "blockchain.json"
	walletFile := "wallet.json"
	difficulty := 2

	bc := blockchain.NewBlockchain(difficulty)

	if _, err := os.Stat(dbFile); err == nil {
		err = storage.LoadFromFile(dbFile, bc)
		if err != nil {
			fmt.Printf("Error loading blockchain from file: %v\n", err)
			return
		}
		bc.RebuildLedger()
	} else {
		_ = storage.SaveToFile(dbFile, bc)
	}

	nodeWallet, err := wallet.LoadOrCreateWallet(walletFile)
	if err != nil {
		fmt.Printf("Error loading wallet: %v\n", err)
		return
	}
	nodeWallet.SyncBalance(bc.Ledger)

	if *mode == "serve" {
		fmt.Printf("Loading node config from %q\n", *configPath)
		cfg, err := node.LoadConfig(*configPath)
		if err != nil {
			fmt.Printf("Error loading node config: %v\n", err)
			return
		}
		if envPeers := os.Getenv("NODE_PEERS"); envPeers != "" {
			cfg.Peers = node.ParsePeers(envPeers)
		}
		n := node.NewNode(bc, nodeWallet, cfg)
		if err := n.Start(*addr); err != nil {
			fmt.Printf("HTTP server failed: %v\n", err)
		}
		return
	}

	appCLI := cli.NewCLI(bc, nodeWallet)
	appCLI.Run()

	nodeWallet.SyncBalance(bc.Ledger)
	if err := wallet.SaveWallet(walletFile, nodeWallet); err != nil {
		fmt.Printf("Error saving wallet to file: %v\n", err)
	}

	err = storage.SaveToFile(dbFile, bc)
	if err != nil {
		fmt.Printf("Error saving blockchain to file: %v\n", err)
	}
}
