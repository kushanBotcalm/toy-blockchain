package block

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"toy-blockchain/transaction"
)

type Block struct {
	Index        int                       `json:"index"`
	Timestamp    int64                     `json:"timestamp"`
	Transactions []transaction.Transaction `json:"transactions"`
	PreviousHash string                    `json:"previous_hash"`
	Nonce        int                       `json:"nonce"`
	Hash         string                    `json:"hash"`
}

func (b Block) CalculateHash() string {

	data := struct {
		Index        int
		Timestamp    int64
		Transactions []transaction.Transaction
		PreviousHash string
		Nonce        int
	}{
		Index:        b.Index,
		Timestamp:    b.Timestamp,
		Transactions: b.Transactions,
		PreviousHash: b.PreviousHash,
		Nonce:        b.Nonce,
	}

	bytes, _ := json.Marshal(data)

	hash := sha256.Sum256(bytes)

	return hex.EncodeToString(hash[:])
}
