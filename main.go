package main

import (
	"fmt"

	"toy-blockchain/block"
	"toy-blockchain/transaction"
)

func main() {
	tx1 := transaction.Transaction{
		Sender:   "Alice",
		Receiver: "Bob",
		Amount:   100,
	}

	tx2 := transaction.Transaction{
		Sender:   "Bob",
		Receiver: "Charlie",
		Amount:   50,
	}

	b := block.Block{
		Index:        0,
		Timestamp:    0,
		Transactions: []transaction.Transaction{tx1, tx2},
		PreviousHash: "",
		Nonce:        0,
		Hash:         "",
	}

	fmt.Printf("%+v\n", b)
}
