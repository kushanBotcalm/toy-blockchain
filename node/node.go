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
	"toy-blockchain/storage"
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
	Blockchain  *blockchain.Blockchain
	Wallet      *wallet.Wallet
	Config      Config
	StorageFile string
}

func NewNode(bc *blockchain.Blockchain, w *wallet.Wallet, cfg Config, storageFile string) *Node {
	return &Node{Blockchain: bc, Wallet: w, Config: cfg, StorageFile: storageFile}
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
	respondJSON(w, http.StatusOK, n.Blockchain)
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

	if !transaction.VerifyTransaction(&tx) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid signature"})
		return
	}

	if err := n.Blockchain.AddTransaction(tx); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := storage.SaveToFile(n.StorageFile, n.Blockchain); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to persist transaction"})
		return
	}

	// Broadcast to peers (best-effort, fire-and-forget)
	go n.broadcastToPeers("/tx", body)

	respondJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

// handleBlock receives a block from a peer (POST /block).
// It de-duplicates, validates PoW and linkage, and appends if it extends the chain.
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

	// basic validation: pow and linkage
	if !mining.ValidatePoW(blk, n.Blockchain.Difficulty) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid proof of work"})
		return
	}

	last := n.Blockchain.Blocks[len(n.Blockchain.Blocks)-1]
	if blk.PreviousHash != last.Hash {
		// competing or out-of-order block — ask peer for sync in real implementation
		respondJSON(w, http.StatusConflict, map[string]string{"status": "not extending"})
		return
	}

	// apply transactions to ledger
	for _, tx := range blk.Transactions {
		_ = n.Blockchain.Ledger.ApplyTransaction(tx)
		// mark tx seen in deduper
		if n.Blockchain.Deduper != nil {
			if txID, err := transaction.ID(&tx); err == nil {
				n.Blockchain.Deduper.SeenTransaction(txID)
			}
		}
	}

	// append block
	n.Blockchain.Blocks = append(n.Blockchain.Blocks, blk)
	if n.Blockchain.Deduper != nil && blk.Hash != "" {
		n.Blockchain.Deduper.SeenBlock(blk.Hash)
	}

	// remove any included transactions from pending pool
	n.Blockchain.PendingMu.Lock()
	if len(n.Blockchain.PendingTxPool) > 0 {
		keep := make([]transaction.Transaction, 0, len(n.Blockchain.PendingTxPool))
		for _, ptx := range n.Blockchain.PendingTxPool {
			id, err := transaction.ID(&ptx)
			if err != nil {
				keep = append(keep, ptx)
				continue
			}
			// if tx was included in block, don't keep
			included := false
			for _, btx := range blk.Transactions {
				bid, err := transaction.ID(&btx)
				if err == nil && bid == id {
					included = true
					break
				}
			}
			if !included {
				keep = append(keep, ptx)
			}
		}
		n.Blockchain.PendingTxPool = keep
		// rebuild pending index
		n.Blockchain.PendingIndex = make(map[string]struct{})
		for _, ptx := range n.Blockchain.PendingTxPool {
			if id, err := transaction.ID(&ptx); err == nil {
				n.Blockchain.PendingIndex[id] = struct{}{}
			}
		}
	}
	n.Blockchain.PendingMu.Unlock()

	if err := wallet.SyncAllWalletBalances(n.Blockchain.Ledger); err != nil {
		fmt.Printf("wallet balance update failed: %v\n", err)
	}
	n.Wallet.SyncBalance(n.Blockchain.Ledger)
	if err := storage.SaveToFile(n.StorageFile, n.Blockchain); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to persist block"})
		return
	}

	// gossip accepted block to peers
	go n.broadcastToPeers("/block", body)

	respondJSON(w, http.StatusAccepted, map[string]string{"status": "block accepted"})
}

// handleHeight returns current chain height and head hash.
func (n *Node) handleHeight(w http.ResponseWriter, _ *http.Request) {
	last := n.Blockchain.Blocks[len(n.Blockchain.Blocks)-1]
	respondJSON(w, http.StatusOK, map[string]any{"height": len(n.Blockchain.Blocks) - 1, "head": last.Hash})
}

// handleBlocks returns blocks optionally starting from a given index: /blocks?from=N
func (n *Node) handleBlocks(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()
	fromStr := qs.Get("from")
	if fromStr == "" {
		respondJSON(w, http.StatusOK, n.Blockchain.Blocks)
		return
	}
	from, err := strconv.Atoi(fromStr)
	if err != nil || from < 0 || from >= len(n.Blockchain.Blocks) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid from parameter"})
		return
	}
	respondJSON(w, http.StatusOK, n.Blockchain.Blocks[from:])
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

func (n *Node) Start(listenAddr string) error {
	fmt.Printf("Node HTTP service listening on %s\n", listenAddr)
	return http.ListenAndServe(listenAddr, n.routes())
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
