package transaction

import "testing"

// TestFR2InvalidSignatureRejected verifies:
//
// Given: a transaction exists
// When:  the transaction signature does not match its public key
// Then:  the transaction is rejected
func TestFR2InvalidSignatureRejected(t *testing.T) {

	// Generate a valid key pair.
	_, privateKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	// Create a transaction.
	tx := &Transaction{
		Sender:   "Alice",
		Receiver: "Bob",
		Amount:   100,
	}

	// Sign the transaction with the private key.
	err = SignTransaction(tx, privateKey)
	if err != nil {
		t.Fatalf("failed to sign transaction: %v", err)
	}

	// Replace the valid signature with an invalid signature.
	// The public key is still the original public key.
	tx.Signature = []byte("invalid-signature")

	// Verify the transaction.
	isValid := VerifyTransaction(tx)

	// The transaction must be rejected.
	if isValid {
		t.Fatal("FR-2 failed: transaction with invalid signature was accepted")
	}

	t.Log("FR-2 passed: transaction with invalid signature was rejected")
}
