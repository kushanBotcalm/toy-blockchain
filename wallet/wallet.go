package wallet

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"toy-blockchain/ledger"
)

type Wallet struct {
	PrivateKey ed25519.PrivateKey `json:"private_key"`
	PublicKey  ed25519.PublicKey  `json:"public_key"`
	Address    string             `json:"address"`
	Balance    float64            `json:"balance"`
}

func NewWallet() (*Wallet, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Wallet{
		PrivateKey: priv,
		PublicKey:  pub,
		Address:    AddressFromPublicKey(pub),
	}, nil
}

func AddressFromPublicKey(pub ed25519.PublicKey) string {
	sum := sha256.Sum256(pub)
	return hex.EncodeToString(sum[:])
}

func LoadWallet(path string) (*Wallet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var w Wallet
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, err
	}
	if len(w.PrivateKey) == 0 || len(w.PublicKey) == 0 || w.Address == "" {
		return nil, errors.New("invalid wallet file")
	}
	return &w, nil
}

func SaveWallet(path string, wallet *Wallet) error {
	data, err := json.MarshalIndent(wallet, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (w *Wallet) SyncBalance(l *ledger.Ledger) {
	if l == nil {
		w.Balance = 0
		return
	}
	w.Balance = l.GetBalance(w.Address)
}

// SyncAllWalletBalances recalculates and persists the three node wallet files.
func SyncAllWalletBalances(l *ledger.Ledger) error {
	for _, path := range []string{
		"wallet_8080.json",
		"wallet_8081.json",
		"wallet_8082.json",
	} {
		wallet, err := LoadWallet(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		wallet.SyncBalance(l)
		if err := SaveWallet(path, wallet); err != nil {
			return err
		}
	}
	return nil
}

func LoadOrCreateWallet(path string) (*Wallet, error) {
	if _, err := os.Stat(path); err == nil {
		return LoadWallet(path)
	}
	wallet, err := NewWallet()
	if err != nil {
		return nil, err
	}
	if err := SaveWallet(path, wallet); err != nil {
		return nil, err
	}
	return wallet, nil
}

func WalletFileForListenAddr(listenAddr string) string {
	addr := strings.TrimSpace(listenAddr)
	addr = strings.TrimPrefix(addr, ":")
	addr = strings.TrimPrefix(addr, "http://")
	addr = strings.TrimPrefix(addr, "https://")
	addr = strings.TrimPrefix(addr, "localhost")
	addr = strings.Trim(addr, ":/")
	addr = strings.ReplaceAll(addr, ":", "_")
	addr = strings.ReplaceAll(addr, "/", "-")
	if addr == "" {
		addr = "default"
	}
	return "wallet_" + addr + ".json"
}
