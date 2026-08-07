package transaction

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
)

// GenerateKeyPair returns a newly generated ed25519 key pair.
func GenerateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	return pub, priv, err
}

// signableBytes returns a deterministic byte slice for signing a transaction.
// It deliberately excludes the PublicKey and Signature fields.
func signableBytes(tx *Transaction) ([]byte, error) {
	if tx == nil {
		return nil, errors.New("nil transaction")
	}
	payload := struct {
		Sender   string  `json:"sender"`
		Receiver string  `json:"receiver"`
		Amount   float64 `json:"amount"`
	}{
		Sender:   tx.Sender,
		Receiver: tx.Receiver,
		Amount:   tx.Amount,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(b)
	return sum[:], nil
}

// SignTransaction signs the given transaction using the provided private key.
// It sets the PublicKey and Signature fields on success.
func SignTransaction(tx *Transaction, priv ed25519.PrivateKey) error {
	if tx == nil {
		return errors.New("nil transaction")
	}
	msg, err := signableBytes(tx)
	if err != nil {
		return err
	}
	sig := ed25519.Sign(priv, msg)
	// extract public key from private key
	pub := priv.Public().(ed25519.PublicKey)
	tx.PublicKey = []byte(pub)
	tx.Signature = sig
	return nil
}

// VerifyTransaction verifies the transaction's signature using its PublicKey.
// It returns true if the signature is present and valid for the transaction payload.
func VerifyTransaction(tx *Transaction) bool {
	if tx == nil || len(tx.PublicKey) == 0 || len(tx.Signature) == 0 {
		return false
	}
	msg, err := signableBytes(tx)
	if err != nil {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(tx.PublicKey), msg, tx.Signature)
}
