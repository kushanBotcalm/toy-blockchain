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
	peers := flag.String("peers", "", "Comma-separated list of peer addresses (e.g., http://localhost:8081,http://localhost:8082)")
	flag.Parse()

	listenAddr := *addr
	if *mode != "serve" {
		listenAddr = ":8080"
	}
	dbFile := storage.BlockchainFileForListenAddr(listenAddr)
	walletFile := wallet.WalletFileForListenAddr(listenAddr)
	difficulty := 2

	bc := blockchain.NewBlockchain(difficulty)

	if _, err := os.Stat(dbFile); err == nil {
		err = storage.LoadFromFile(dbFile, bc)
		if err != nil {
			fmt.Printf("Error loading blockchain from file: %v\n", err)
			return
		}
		bc.RebuildDeduper()
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
	if err := wallet.SyncAllWalletBalances(bc.Ledger); err != nil {
		fmt.Printf("Warning: failed to update wallet balances: %v\n", err)
	}

	if *mode == "serve" {
		cfg, err := node.LoadConfig(*configPath)
		if err != nil {
			fmt.Printf("Error loading node config: %v\n", err)
			return
		}

		if *peers != "" {
			cfg.Peers = node.ParsePeers(*peers)
		} else if envPeers := os.Getenv("NODE_PEERS"); envPeers != "" {
			cfg.Peers = node.ParsePeers(envPeers)
		}

		fmt.Printf("Node blockchain file: %s\n", dbFile)
		fmt.Printf("Node wallet file: %s\n", walletFile)
		fmt.Printf("Node wallet address: %s\n", nodeWallet.Address)
		fmt.Printf("Node peers: %v\n", cfg.Peers)
		n := node.NewNode(bc, nodeWallet, cfg, dbFile)
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
