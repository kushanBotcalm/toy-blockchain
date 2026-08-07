package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"toy-blockchain/blockchain"
	"toy-blockchain/transaction"
)

type CLI struct {
	Blockchain *blockchain.Blockchain
}

func NewCLI(bc *blockchain.Blockchain) *CLI {
	return &CLI{Blockchain: bc}
}

// printUsage displays available commands clearly for the user
func printUsage() {
	fmt.Println("=== Toy Blockchain CLI Simulator ===")
	fmt.Println("Usage:")
	fmt.Println("  go run main.go <command> [arguments]")
	fmt.Println()
	fmt.Println("Available Commands:")
	fmt.Println("  add       Add a transaction to the pending pool (-s SENDER -r RECEIVER -a AMOUNT)")
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
	var sender, receiver string
	var amount float64

	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-s":
			if i+1 < len(os.Args) {
				sender = os.Args[i+1]
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

	if sender == "" || receiver == "" || amount <= 0 {
		fmt.Println("Error: Invalid or missing arguments.")
		fmt.Println("Example usage: go run main.go add -s FAUCET -r Alice -a 100")
		return
	}

	_, priv, err := transaction.GenerateKeyPair()
	if err != nil {
		fmt.Printf("Failed to generate key pair: %v\n", err)
		return
	}

	tx := transaction.Transaction{
		Sender:   sender,
		Receiver: receiver,
		Amount:   amount,
	}
	if err := transaction.SignTransaction(&tx, priv); err != nil {
		fmt.Printf("Failed to sign transaction: %v\n", err)
		return
	}

	err = cli.Blockchain.AddTransaction(tx)
	if err != nil {
		fmt.Printf("Failed to add transaction: %v\n", err)
		return
	}
	fmt.Println("Success: Transaction added to the pending pool")
}

func (cli *CLI) handleMineBlock() {
	var minerAddress string

	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "-m" && i+1 < len(os.Args) {
			minerAddress = os.Args[i+1]
		}
	}

	if minerAddress == "" {
		fmt.Println("Error: Missing miner address.")
		fmt.Println("Example usage: go run main.go mine -m MinerNode")
		return
	}

	fmt.Println("Mining new block... Please wait")
	minedBlock := cli.Blockchain.MinePendingTransactions(minerAddress)

	fmt.Printf("Success! Block mined at Index %d with Hash: %s (Nonce: %d)\n", minedBlock.Index, minedBlock.Hash, minedBlock.Nonce)
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

	balance := cli.Blockchain.Ledger.GetBalance(address)
	fmt.Printf("Account Balance for '%s': %.2f[cite: 1]\n", address, balance)
}
