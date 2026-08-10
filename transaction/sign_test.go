package transaction_test

import (
	"testing"

	"toy-blockchain/blockchain"
	"toy-blockchain/transaction"
)

func TestFR2InvalidSignatureRejected(t *testing.T) {

	// Generate a valid key pair.
	_, privateKey, err := transaction.GenerateKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	// Create a transaction.
	tx := &transaction.Transaction{
		Sender:   "Alice",
		Receiver: "Bob",
		Amount:   100,
	}

	// Sign the transaction with the private key.
	err = transaction.SignTransaction(tx, privateKey)
	if err != nil {
		t.Fatalf("failed to sign transaction: %v", err)
	}

	// Replace the valid signature with an invalid signature.
	// The public key is still the original public key.
	tx.Signature = []byte("invalid-signature")

	// Verify the transaction.
	isValid := transaction.VerifyTransaction(tx)

	// The transaction must be rejected.
	if isValid {
		t.Fatal("FR-2 failed: transaction with invalid signature was accepted")
	}

	// Create a blockchain/node.
	bc := blockchain.NewBlockchain(2)

	// Record the pending pool size before adding the invalid transaction.
	initialPendingCount := len(bc.PendingTxPool)

	// Try to add the invalid transaction.
	err = bc.AddTransaction(*tx)

	// The node MUST reject the transaction.
	if err == nil {
		t.Fatal("FR-2 failed: invalid transaction was accepted by the node")
	}

	// The invalid transaction MUST NOT be added to the pending pool.
	if len(bc.PendingTxPool) != initialPendingCount {
		t.Fatal("FR-2 failed: invalid transaction was added to the pending pool")
	}

	t.Log("FR-2 passed: invalid transaction rejected and not added to pending pool")
}
