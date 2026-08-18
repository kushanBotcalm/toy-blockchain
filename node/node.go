package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"toy-blockchain/block"
	"toy-blockchain/blockchain"
	"toy-blockchain/mining"
	"toy-blockchain/transaction"
	"toy-blockchain/wallet"
)

type Config struct {
	Address string   `json:"address"`
	Peers   []string `json:"peers"`
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && filepath.Ext(path) == "" {
			data, err = os.ReadFile(path + ".json")
		}
		if err != nil {
			return cfg, err
		}
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func ParsePeers(peers string) []string {
	if peers == "" {
		return nil
	}
	list := strings.Split(peers, ",")
	for i := range list {
		list[i] = strings.TrimSpace(list[i])
	}
	return list
}

type Node struct {
	Blockchain *blockchain.Blockchain
	Wallet     *wallet.Wallet
	Config     Config
}

func NewNode(bc *blockchain.Blockchain, w *wallet.Wallet, cfg Config) *Node {
	return &Node{Blockchain: bc, Wallet: w, Config: cfg}
}

func (n *Node) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", n.handleHealth)
	mux.HandleFunc("/info", n.handleInfo)
	mux.HandleFunc("/chain", n.handleChain)
	mux.HandleFunc("/peers", n.handlePeers)
	mux.HandleFunc("/wallet", n.handleWallet)
	mux.HandleFunc("/pending", n.handlePending)
	mux.HandleFunc("/tx", n.handleTx)
	mux.HandleFunc("/block", n.handleBlock)
	mux.HandleFunc("/height", n.handleHeight)
	mux.HandleFunc("/blocks", n.handleBlocks)
	// Phase 4: Sync and fork resolution endpoints
	mux.HandleFunc("/sync", n.handleSync)
	mux.HandleFunc("/sync/request", n.handleSyncRequest)
	return mux
}

type nodeInfo struct {
	NodeAddress string   `json:"node_address"`
	Peers       []string `json:"peers"`
	ChainLength int      `json:"chain_length"`
}

type walletInfo struct {
	Address string  `json:"address"`
	Balance float64 `json:"balance"`
}

type pendingInfo struct {
	Count        int                       `json:"pending_count"`
	Transactions []transaction.Transaction `json:"pending_transactions"`
}

func (n *Node) handleHealth(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status":       "ok",
		"node_address": n.Config.Address,
	})
}

func (n *Node) handleInfo(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, nodeInfo{
		NodeAddress: n.Config.Address,
		Peers:       n.Config.Peers,
		ChainLength: len(n.Blockchain.Blocks),
	})
}

func (n *Node) handleChain(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{
		"blocks": n.Blockchain.GetAllBlocksCopy(),
	})
}

func (n *Node) handlePeers(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{
		"peers": n.Config.Peers,
	})
}

func (n *Node) handleWallet(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, walletInfo{
		Address: n.Wallet.Address,
		Balance: n.Wallet.Balance,
	})
}

func (n *Node) handlePending(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, pendingInfo{
		Count:        len(n.Blockchain.PendingTxPool),
		Transactions: n.Blockchain.PendingTxPool,
	})
}

// handleTx receives a transaction from a client or peer (POST /tx).
// It validates signatures, checks deduper/pending, adds to pool, and gossips to peers.
func (n *Node) handleTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "only POST allowed"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read body"})
		return
	}
	var tx transaction.Transaction
	if err := json.Unmarshal(body, &tx); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid transaction JSON"})
		return
	}

	// compute tx id and check deduper
	txID, err := transaction.ID(&tx)
	if err == nil && n.Blockchain.Deduper != nil {
		if n.Blockchain.Deduper.SeenTransaction(txID) {
			respondJSON(w, http.StatusOK, map[string]string{"status": "duplicate"})
			return
		}
	}

	if !transaction.VerifyTransaction(&tx) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid signature"})
		return
	}

	if err := n.Blockchain.AddTransaction(tx); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	//Broadcast to peers (best-effort, fire-and-forget)

	respondJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

// handleBlock receives a block from a peer (POST /block).
// It de-duplicates, validates PoW and linkage, and appends if it extends the chain.
// PHASE 4: Now detects competing chains and triggers sync
func (n *Node) handleBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "only POST allowed"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read body"})
		return
	}
	var blk block.Block
	if err := json.Unmarshal(body, &blk); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid block JSON"})
		return
	}

	// de-dup by block hash
	if blk.Hash != "" && n.Blockchain.Deduper != nil {
		if n.Blockchain.Deduper.SeenBlock(blk.Hash) {
			respondJSON(w, http.StatusOK, map[string]string{"status": "duplicate"})
			return
		}
	}

	// basic validation: pow
	if !mining.ValidatePoW(blk, n.Blockchain.Difficulty) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid proof of work"})
		return
	}

	last := n.Blockchain.GetLastBlock()

	// CASE 1: Block extends our chain
	if blk.PreviousHash == last.Hash {
		if err := n.applyAndAppendBlock(blk); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		go n.broadcastToPeers("/block", body)
		respondJSON(w, http.StatusAccepted, map[string]string{"status": "block accepted"})
		return
	}

	// CASE 2: Competing chain (block doesn't extend our tip)
	fmt.Printf("Received competing block at height %d. Initiating sync.\n", blk.Index)
	go func() {
		if len(n.Config.Peers) > 0 {
			_ = n.SyncFromPeer(n.Config.Peers[0])
		}
	}()

	respondJSON(w, http.StatusConflict, map[string]string{"status": "competing block received, syncing"})
}

// handleHeight returns current chain height and head hash.
func (n *Node) handleHeight(w http.ResponseWriter, _ *http.Request) {
	last := n.Blockchain.GetLastBlock()
	respondJSON(w, http.StatusOK, map[string]any{
		"height": n.Blockchain.GetHeight(),
		"head":   last.Hash,
	})
}

// handleBlocks returns blocks optionally starting from a given index: /blocks?from=N
func (n *Node) handleBlocks(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()
	fromStr := qs.Get("from")
	if fromStr == "" {
		fromStr = "0"
	}
	from, err := strconv.Atoi(fromStr)
	if err != nil || from < 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid from parameter"})
		return
	}
	blocks := n.Blockchain.GetBlocksCopy(from)
	respondJSON(w, http.StatusOK, blocks)
}

// broadcastToPeers sends the given JSON payload to each configured peer at the provided path.
func (n *Node) broadcastToPeers(path string, payload []byte) {
	for _, peer := range n.Config.Peers {
		peer := peer // capture
		go func() {
			url := peer + path
			_, _ = http.Post(url, "application/json", bytes.NewReader(payload))
		}()
	}
}

// handleSync handles both sync initiation and responses
// GET /sync - returns peer's chain for pulling
// POST /sync - receives full chain from peer for fork resolution
func (n *Node) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Peer is asking for our chain
		respondJSON(w, http.StatusOK, map[string]any{
			"height": n.Blockchain.GetHeight(),
			"blocks": n.Blockchain.GetAllBlocksCopy(),
		})
		return
	}

	if r.Method == http.MethodPost {
		// We're receiving a chain from a peer (fork resolution)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read body"})
			return
		}

		var syncPayload struct {
			Blocks []block.Block `json:"blocks"`
		}
		if err := json.Unmarshal(body, &syncPayload); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid sync JSON"})
			return
		}

		// Validate and reorganise if competing chain is better
		isValid, err := n.Blockchain.IsCompetingChainValid(syncPayload.Blocks)
		if !isValid {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid competing chain: %v", err)})
			return
		}

		// Reorganise to the competing chain
		if err := n.Blockchain.Reorganise(syncPayload.Blocks); err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("reorganisation failed: %v", err)})
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{"status": "reorganised to competing chain"})
		return
	}

	respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "only GET and POST allowed"})
}

// handleSyncRequest handles /sync/request?from=N to fetch blocks in batch
// Used by lagging nodes to catch up
func (n *Node) handleSyncRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "only GET allowed"})
		return
	}

	fromStr := r.URL.Query().Get("from")
	if fromStr == "" {
		fromStr = "0"
	}

	from, err := strconv.Atoi(fromStr)
	if err != nil || from < 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid from parameter"})
		return
	}

	blocks := n.Blockchain.GetBlocksCopy(from)
	respondJSON(w, http.StatusOK, map[string]any{
		"from":   from,
		"blocks": blocks,
	})
}

// SyncFromPeer attempts to sync the full chain from a peer
func (n *Node) SyncFromPeer(peerURL string) error {
	// 1. Get peer's height
	resp, err := http.Get(peerURL + "/height")
	if err != nil {
		return fmt.Errorf("failed to get peer height: %w", err)
	}
	defer resp.Body.Close()

	var peerHeight struct {
		Height int    `json:"height"`
		Head   string `json:"head"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&peerHeight); err != nil {
		return fmt.Errorf("failed to decode peer height: %w", err)
	}

	myHeight := n.Blockchain.GetHeight()
	if peerHeight.Height <= myHeight {
		fmt.Printf("Peer is not ahead. Peer height: %d, My height: %d\n", peerHeight.Height, myHeight)
		return nil
	}

	// 2. Download missing blocks in batches
	fmt.Printf("Syncing from peer %s (peer height: %d, my height: %d)\n", peerURL, peerHeight.Height, myHeight)

	from := myHeight + 1
	for from <= peerHeight.Height {
		resp, err := http.Get(fmt.Sprintf("%s/sync/request?from=%d", peerURL, from))
		if err != nil {
			return fmt.Errorf("failed to fetch blocks from %d: %w", from, err)
		}

		var syncResp struct {
			From   int           `json:"from"`
			Blocks []block.Block `json:"blocks"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
			resp.Body.Close()
			return fmt.Errorf("failed to decode sync response: %w", err)
		}
		resp.Body.Close()

		if len(syncResp.Blocks) == 0 {
			break
		}

		// 3. Validate and apply each block
		for _, blk := range syncResp.Blocks {
			if err := n.validateAndAppendBlock(blk); err != nil {
				return fmt.Errorf("failed to apply synced block at height %d: %w", blk.Index, err)
			}
		}

		from += len(syncResp.Blocks)
	}

	fmt.Printf("Sync complete. Chain height: %d\n", n.Blockchain.GetHeight())
	return nil
}

// validateAndAppendBlock safely validates and appends a block during sync
func (n *Node) validateAndAppendBlock(blk block.Block) error {
	// Validate PoW
	if !mining.ValidatePoW(blk, n.Blockchain.Difficulty) {
		return fmt.Errorf("invalid PoW")
	}

	// Validate linkage
	last := n.Blockchain.GetLastBlock()
	if blk.PreviousHash != last.Hash {
		return fmt.Errorf("broken chain linkage: expected %s, got %s", last.Hash, blk.PreviousHash)
	}

	// Validate all transactions
	for _, tx := range blk.Transactions {
		if !transaction.VerifyTransaction(&tx) {
			return fmt.Errorf("invalid tx signature in synced block")
		}
	}

	// Apply block (under lock via AppendBlock)
	if err := n.Blockchain.AppendBlock(blk); err != nil {
		return fmt.Errorf("failed to append block: %w", err)
	}

	// Remove included txs from pending pool
	n.Blockchain.RemoveIncludedTransactions(blk.Transactions)

	return nil
}

// applyAndAppendBlock safely applies a block and appends it to chain
func (n *Node) applyAndAppendBlock(blk block.Block) error {
	if err := n.Blockchain.AppendBlock(blk); err != nil {
		return err
	}
	n.Blockchain.RemoveIncludedTransactions(blk.Transactions)
	return nil
}

func (n *Node) Start(listenAddr string) error {
	fmt.Printf("Node HTTP service listening on %s\n", listenAddr)
	return http.ListenAndServe(listenAddr, n.routes())
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
