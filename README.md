# Toy Blockchain

A small Go-based blockchain simulator that demonstrates core concepts such as blocks, transactions, a simple proof-of-work mining process, a balance ledger, and a basic CLI.

## Features

- Creates a genesis block automatically
- Adds transactions to a pending pool
- Mines new blocks with a simple proof-of-work implementation
- Tracks balances in a ledger
- Persists blockchain state to `blockchain.json`
- Provides a command-line interface for common blockchain actions

## Requirements

- Go 1.26 or newer

## Build

From the project root, run:

```bash
go build ./...
```

You can also build a binary directly:

```bash
go build -o toy-blockchain .
```

## Run

Start the CLI with:

```bash
go run main.go
```

Or, if you built the binary:

```bash
./toy-blockchain
```

## Test

Run the test suite from the project root:

```bash
go test ./...
```

## CLI Commands

The CLI supports the following commands:

```bash
go run main.go help
```

### Add a transaction

```bash
go run main.go add -s FAUCET -r Alice -a 100
```

### Mine a new block

```bash
go run main.go mine -m MinerNode
```

### Print the blockchain

```bash
go run main.go print
```

### Validate the chain

```bash
go run main.go validate
```

### Check an account balance

```bash
go run main.go balance -a Alice
```

## Design Notes

- The project uses a deliberately simple blockchain design for learning and demonstration purposes.
- Proof of work is implemented with a configurable difficulty value.
- The ledger is rebuilt from block history when the app starts so the blockchain state can be recovered from disk.
- Transaction state is persisted to `blockchain.json` in the project root.

## Known Limitations

- This is not a production blockchain implementation.
- There is no peer-to-peer network, no wallet signing, and no real cryptographic identity model.
- Transactions are not modeled as a full UTXO system and there is no support for fees, smart contracts, or mempool prioritization.
- The CLI is intentionally minimal and uses simple command-line flags rather than a more complete interactive interface.
