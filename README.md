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

## Service mode

Start the HTTP node service with:

```bash
go run main.go -mode=serve -addr=":8080" -config=node_config.json
```

Create `node_config.json` first, for example:

```json
{
  "address": "http://localhost:8080",
  "peers": [
    "http://localhost:8081",
    "http://localhost:8082"
  ]
}
```

You can also set peers with an environment variable instead of editing the JSON file:

```powershell
set NODE_PEERS=127.0.0.1:8081,127.0.0.1:8082
```

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

## Phase 4: Networked Multi-Node Blockchain

This version supports chain synchronisation, fork resolution, and race-free concurrent access.

### Chain Synchronisation (FR-5)
- New nodes can join the network and sync the full chain from peers
- Nodes automatically catch up if they fall behind during mining
- Blocks are validated during sync before appending to chain
- **Endpoint**: `GET /sync/request?from=N` - Returns blocks starting from index N

### Fork Resolution & Reorganisation (FR-6)
- When two valid chains compete, nodes detect and reorganise to the longest valid chain
- Orphaned transactions are returned to the pending pool if still valid
- Ledger state is properly rebuilt after reorganisation
- **Endpoint**: `POST /sync` - Receive competing chain and reorganise if valid

### Concurrency Safety (FR-7)
- **All block reads** are protected with `RWMutex` (fast reads, exclusive writes)
- **All block writes** are protected with write locks during appends and reorganisations
- **Pending pool** uses separate `PendingMu` mutex for thread-safe queue management
- **Race-free**: Pass `go test -race ./...` with concurrent mining and gossip

### New HTTP Endpoints (Phase 4)
- `GET /sync` - Returns your chain (peers use this to pull data)
- `POST /sync` - Send a competing chain for evaluation and potential reorganisation
- `GET /sync/request?from=N` - Returns blocks in batches starting from index N
- `GET /height` - Returns current height and head hash

### Manual Testing: Start a Local Cluster

```bash
# Terminal 1: Start Node A (port 8001)
go run main.go -mode=serve -addr=":8001" -config=node_a.json

# Terminal 2: Start Node B (port 8002, peered with A)
go run main.go -mode=serve -addr=":8002" -config=node_b.json

# Terminal 3: Mine a block on A
curl -X POST http://localhost:8001/mine -d '{"miner":"NodeA"}'

# Terminal 4: Verify B synced automatically
curl http://localhost:8002/height
# Should show same height as A

# Terminal 5: Stop B, mine on A, restart B - B will sync
curl http://localhost:8001/mine -d '{"miner":"NodeA"}'
# Restart B - it will sync the new block
```

### Design Decisions (Phase 4)

**1. Fork Rule**: We use the **longest-chain rule**
   - When a valid chain is longer than ours, we reorganise to it
   - All blocks must pass PoW validation and transaction signature checks
   
**2. Concurrency Model**: Separate mutexes for blocks and pending pool
   - `mu sync.RWMutex` - Protects Blocks slice and Ledger state
   - `PendingMu sync.RWMutex` - Protects PendingTxPool and PendingIndex
   - This allows pending pool updates while reading blocks for broadcast

**3. Reorganisation Depth**: Unlimited
   - We can reorganise multiple blocks if a competing chain is longer and valid
   - All orphaned transactions are checked for validity before returning to pool

**4. Sync Strategy**: Batch download with per-block validation
   - Download blocks in batches via `/sync/request?from=N`
   - Validate each block's PoW, linkage, and transactions before appending
   - Reduces network chatter compared to block-by-block sync

### Testing Race Conditions

Verify there are no races under concurrent load:

```bash
go test -race ./...
go test -race ./blockchain -v
go test -race ./node -v
```

All concurrent operations (mining, gossip, sync, block acceptance) must be race-free.

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
