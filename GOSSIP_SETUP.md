# Gossip Protocol Setup Guide

## Overview
The toy-blockchain now implements a gossip protocol where:
- **Port 8080**: Main server that accepts transactions from CLI and gossips them to other peers
- **Port 8081, 8082**: Peer nodes that receive gossip messages
- Peers are **NOT** hardcoded in config files but passed via CLI or environment variables

## Architecture

```
CLI (local machine)
    ↓
    ├─→ PORT 8080 (Main Server)
             ↓
             ├─→ Gossip to PORT 8081
             └─→ Gossip to PORT 8082
```

## How to Run

### Start Node 1 (Port 8080) - Main Server
```bash
go run main.go -mode serve -addr :8080
```

### Start Node 2 (Port 8081) - Peer
```bash
go run main.go -mode serve -addr :8081 -peers http://localhost:8080
```

### Start Node 3 (Port 8082) - Peer
```bash
go run main.go -mode serve -addr :8082 -peers http://localhost:8080,http://localhost:8081
```

### Or use Environment Variables
```bash
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

## Add Transaction via CLI

Once the servers are running, use the CLI to add a transaction:

```bash
# Terminal 4 - Add transaction (automatically sends to port 8080)
go run main.go add -r Alice -a 100
```

## Flow

1. **CLI sends to 8080**: Transaction is sent to the main server (port 8080)
2. **Server validates**: Port 8080 validates the transaction signature
3. **Server adds to pool**: Transaction is added to the pending transaction pool
4. **Gossip to peers**: Transaction is automatically gossiped to all configured peers (8081, 8082)
5. **Peers receive**: Peers receive the transaction and add it to their own pools
6. **Duplicate detection**: If a peer has already seen the transaction, it ignores it

## Configuration Changes

### Before
- `node_config.json` had hardcoded peers
- Peers had to be pre-configured

### After
- `node_config.json` has NO peers
- Peers are passed via:
  - CLI flag: `-peers http://localhost:8081,http://localhost:8082`
  - Environment variable: `NODE_PEERS=http://localhost:8081,http://localhost:8082`

## Key Code Changes

### CLI (cli/cli.go)
- Only sends transactions to `http://localhost:8080/tx`
- Relies on the main server to gossip to other peers

### Node (node/node.go)
- `handleTx()` now calls `broadcastToPeers()` after accepting a transaction
- Gossips both transactions and blocks to all configured peers

### Main (main.go)
- Added `-peers` flag for specifying peer addresses
- Falls back to `NODE_PEERS` environment variable
- Prints peer configuration on startup
