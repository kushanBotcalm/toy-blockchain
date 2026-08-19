package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"toy-blockchain/blockchain"
	"toy-blockchain/storage"
	"toy-blockchain/transaction"
	"toy-blockchain/wallet"
)

type CLI struct {
	Blockchain *blockchain.Blockchain
	Wallet     *wallet.Wallet
}

func NewCLI(bc *blockchain.Blockchain, w *wallet.Wallet) *CLI {
	return &CLI{Blockchain: bc, Wallet: w}
}

// printUsage displays available commands clearly for the user
func printUsage() {
	fmt.Println("=== Toy Blockchain CLI Simulator ===")
	fmt.Println("Usage:")
	fmt.Println("  go run main.go <command> [arguments]")
	fmt.Println()
	fmt.Println("Available Commands:")
	fmt.Println("  add       Add a transaction to the pending pool (-s SENDER_ADDRESS -r RECEIVER -a AMOUNT)")
	fmt.Println("  mine      Mine a new block from the pending pool (-m MINER_ADDRESS)")
	fmt.Println("  print     Print the entire blockchain in a readable form")
	fmt.Println("  validate  Validate the entire chain and check for tampering")
	fmt.Println("  balance   Show the account balance for a given address (-a ADDRESS)")
	fmt.Println("  help      Show this help message")
}

func (cli *CLI) Run() {
	// If no arguments or user requests help, show the command guide first
	if len(os.Args) < 2 || os.Args[1] == "help" || os.Args[1] == "-h" || os.Args[1] == "--help" {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "add":
		cli.handleAddTransaction()
	case "mine":
		cli.handleMineBlock()
	case "print":
		cli.handlePrintChain()
	case "validate":
		cli.handleValidateChain()
	case "balance":
		cli.handleBalance()
	default:
		fmt.Printf("Unknown command: '%s'\n\n", command)
		printUsage()
	}
}

func (cli *CLI) handleAddTransaction() {
	var senderAddress string
	var receiver string
	var amount float64

	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-s":
			if i+1 < len(os.Args) {
				senderAddress = os.Args[i+1]
			}
		case "-r":
			if i+1 < len(os.Args) {
				receiver = os.Args[i+1]
			}
		case "-a":
			if i+1 < len(os.Args) {
				parsed, err := strconv.ParseFloat(os.Args[i+1], 64)
				if err == nil {
					amount = parsed
				}
			}
		}
	}

	senderWallet, err := cli.walletForAddress(senderAddress)
	if senderWallet == nil || receiver == "" || amount <= 0 {
		fmt.Println("Error: Invalid or missing arguments.")
		if err != nil {
			fmt.Printf("Sender wallet error: %v\n", err)
		}
		fmt.Println("Example usage: go run main.go add -s SENDER_ADDRESS -r RECEIVER -a 100")
		return
	}

	tx := transaction.Transaction{
		Sender:   senderWallet.Address,
		Receiver: receiver,
		Amount:   amount,
	}
	if err := transaction.SignTransaction(&tx, senderWallet.PrivateKey); err != nil {
		fmt.Printf("Failed to sign transaction: %v\n", err)
		return
	}
	payload, _ := json.Marshal(tx)

	// Send transaction to port 8080 (main server) which will gossip to other peers
	resp, err := http.Post("http://localhost:8080/tx", "application/json", bytes.NewReader(payload))
	if err != nil {
		fmt.Println("Failed to send transaction to server:", err)
		return
	}
	defer resp.Body.Close()

	// Read and print the response body for debugging
	respBody, _ := io.ReadAll(resp.Body)
	fmt.Println("Transaction sent to server (port 8080) with status:", resp.StatusCode)
	if resp.StatusCode != http.StatusAccepted {
		fmt.Printf("Server error response: %s\n", string(respBody))
		return
	}

	// Keep the CLI copy in sync only after the server accepts the transaction.
	err = cli.Blockchain.AddTransaction(tx)
	if err != nil {
		fmt.Printf("Failed to add transaction: %v\n", err)
		return
	}
	fmt.Println("Success: Transaction added to the pending pool")
}

func (cli *CLI) handleMineBlock() {
	minerAddress := ""

	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "-m" && i+1 < len(os.Args) {
			minerAddress = os.Args[i+1]
		}
	}

	if minerAddress == "" && cli.Wallet != nil {
		minerAddress = cli.Wallet.Address
	}

	if minerAddress == "" {
		fmt.Println("Error: Missing miner address.")
		fmt.Println("Example usage: go run main.go mine -m MINER_ADDRESS")
		return
	}

	fmt.Println("Mining new block... Please wait")
	minedBlock := cli.Blockchain.MinePendingTransactions(minerAddress)
	if err := storage.SaveToFile(storage.BlockchainFileForListenAddr(":8080"), cli.Blockchain); err != nil {
		fmt.Printf("Warning: failed to save blockchain: %v\n", err)
	}
	if err := cli.syncWalletBalances(); err != nil {
		fmt.Printf("Warning: failed to update wallet balances: %v\n", err)
	}

	fmt.Printf("Success! Block mined at Index %d with Hash: %s (Nonce: %d)\n", minedBlock.Index, minedBlock.Hash, minedBlock.Nonce)
}

func (cli *CLI) syncWalletBalances() error {
	return wallet.SyncAllWalletBalances(cli.Blockchain.Ledger)
}

func (cli *CLI) walletForAddress(address string) (*wallet.Wallet, error) {
	if cli.Wallet != nil && (address == "" || cli.Wallet.Address == address) {
		return cli.Wallet, nil
	}
	if address == "" {
		return nil, nil
	}

	files, err := os.ReadDir(".")
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		name := file.Name()
		if file.IsDir() || len(name) < len("wallet_.json") || name[:len("wallet_")] != "wallet_" || name[len(name)-len(".json"):] != ".json" {
			continue
		}
		candidate, err := wallet.LoadWallet(name)
		if err == nil && candidate.Address == address {
			return candidate, nil
		}
	}
	return nil, fmt.Errorf("no wallet found for sender address %s", address)
}

func (cli *CLI) handlePrintChain() {
	data, err := json.MarshalIndent(cli.Blockchain, "", "  ")
	if err != nil {
		fmt.Printf("Error serializing chain: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

func (cli *CLI) handleValidateChain() {
	valid, index := cli.Blockchain.IsValidChain()
	if valid {
		fmt.Println("Validation Result: PASS. The blockchain is honest and valid")
	} else {
		fmt.Printf("Validation Result: FAIL. Tampering detected at block index %d\n", index)
	}
}

func (cli *CLI) handleBalance() {
	var address string

	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "-a" && i+1 < len(os.Args) {
			address = os.Args[i+1]
		}
	}

	if address == "" {
		fmt.Println("Error: Missing address.")
		fmt.Println("Example usage: go run main.go balance -a Alice")
		return
	}

	if err := cli.syncWalletBalances(); err != nil {
		fmt.Printf("Warning: failed to update wallet balances: %v\n", err)
	}
	balance := cli.Blockchain.Ledger.GetBalance(address)
	fmt.Printf("Account Balance for '%s': %.2f\n", address, balance)
}
