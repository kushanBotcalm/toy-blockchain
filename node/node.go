package node

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"toy-blockchain/blockchain"
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

func (n *Node) Start(listenAddr string) error {
	fmt.Printf("Node HTTP service listening on %s\n", listenAddr)
	return http.ListenAndServe(listenAddr, n.routes())
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
