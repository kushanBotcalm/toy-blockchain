# Toy Blockchain

A small Go-based blockchain simulator that demonstrates core concepts such as blocks, transactions, a simple proof-of-work mining process, a balance ledger, and a basic CLI.

## Features

- Creates a genesis block automatically
- Adds transactions to a pending pool
- Mines new blocks with a simple proof-of-work implementation
- Tracks balances in a ledger
- Persists each node's blockchain state to its port-specific JSON file
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

## Service mode (Gossip Protocol)

The blockchain implements a gossip protocol where:
- **Port 8080**: Main server that accepts transactions from CLI and gossips them to peers
- **Peers (8081, 8082, etc.)**: Receive gossip messages and propagate them

### Start Node 1 (Main Server - Port 8080)

```bash
go run main.go -mode serve -addr :8080
```

Each node uses its own files: `blockchain_8080.json` and `wallet_8080.json` for port 8080, with matching names for ports 8081 and 8082. The default `blockchain.json` and `wallet.json` files are not used.

### Start Node 2 (Peer - Port 8081)

```bash
go run main.go -mode serve -addr :8081 -peers http://localhost:8080
```

### Start Node 3 (Peer - Port 8082)

```bash
go run main.go -mode serve -addr :8082 -peers http://localhost:8080,http://localhost:8081
```

### Alternative: Using Environment Variables

```powershell
# Node 1
set NODE_PEERS=
go run main.go -mode serve -addr :8080

# Node 2
set NODE_PEERS=http://localhost:8080
go run main.go -mode serve -addr :8081

# Node 3
set NODE_PEERS=http://localhost:8080,http://localhost:8081
go run main.go -mode serve -addr :8082
```

### Configuration File

The `node_config.json` file now only contains the node address (peers are passed via CLI or environment variables):

```json
{
  "address": "e56d9e6a9ac61b45e422504ade87f663dde20e45bf57ba8110430a92586d2027"
}
```

### How Gossip Works

1. CLI sends a transaction to port 8080
2. Port 8080 validates and adds the transaction to its pool
3. Port 8080 automatically gossips the transaction to all configured peers
4. Peers receive the transaction and add it to their pools
5. Duplicate detection ensures transactions aren't processed twice

### Startup Chain Synchronization

When a node starts with peers configured, it compares its height with each peer:

1. Request the peer height with `GET /height`.
2. If the peer is ahead, request missing blocks with `GET /blocks?from=N`.
3. Validate each block's index, previous hash, hash, and proof of work.
4. Append valid blocks, rebuild the ledger, and save the node's blockchain and wallet files.

For example, start a late node with:

```bash
go run main.go -mode serve -addr :8082 -peers http://localhost:8080,http://localhost:8081
```

Without `-peers` or `NODE_PEERS`, a node has no source from which to synchronize.

## Test

Run the test suite from the project root:

```bash
go test ./...
go test ./transaction
go test ./transaction -v
```

## CLI Commands

The CLI supports the following commands:

```bash
go run main.go help
```

### Add a transaction

The CLI sends transactions to the main server (port 8080), which then gossips to other peers:

```bash
go run main.go add -s SENDER_ADDRESS -r RECEIVER_ADDRESS -a 100
```

Where:
- `-s` specifies the sender wallet address. The CLI loads the matching key from `wallet_8080.json`, `wallet_8081.json`, or `wallet_8082.json`.
- `-r` specifies the receiver address
- `-a` specifies the amount

The transaction is automatically sent to http://localhost:8080/tx and gossipped to all configured peers.

Mine to any of the three node wallets:

```bash
go run main.go mine -m <address from wallet_8080.json>
go run main.go mine -m <address from wallet_8081.json>
go run main.go mine -m <address from wallet_8082.json>
```

If `-m` is omitted, the CLI wallet address is used.
`WALLET_8080_ADDRESS`, `WALLET_8081_ADDRESS`, and `WALLET_8082_ADDRESS` are documentation labels only; they are not wallet addresses.

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
- Transaction state is persisted to the node's port-specific blockchain file in the project root.

## Known Limitations

- This is not a production blockchain implementation.
- There is no peer-to-peer network, no wallet signing, and no real cryptographic identity model.
- Transactions are not modeled as a full UTXO system and there is no support for fees, smart contracts, or mempool prioritization.
- The CLI is intentionally minimal and uses simple command-line flags rather than a more complete interactive interface.
