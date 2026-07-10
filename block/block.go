package block

import "toy-blockchain/transaction"

// Block represents a single block in the blockchain.
type Block struct {
	Index        int                       `json:"index"`
	Timestamp    int64                     `json:"timestamp"`
	Transactions []transaction.Transaction `json:"transactions"`
	PreviousHash string                    `json:"previous_hash"`
	Nonce        int                       `json:"nonce"`
	Hash         string                    `json:"hash"`
}
