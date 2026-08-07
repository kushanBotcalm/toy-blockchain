package transaction

import (
	"testing"
)

func TestSignAndVerify(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair error: %v", err)
	}
	tx := &Transaction{
		Sender:   "alice",
		Receiver: "bob",
		Amount:   1.23,
	}
	if err := SignTransaction(tx, priv); err != nil {
		t.Fatalf("SignTransaction error: %v", err)
	}
	if len(tx.PublicKey) == 0 || len(tx.Signature) == 0 {
		t.Fatalf("signature or public key not set")
	}
	// verify should succeed
	if !VerifyTransaction(tx) {
		t.Fatalf("VerifyTransaction failed (expected success)")
	}
	// tamper the tx
	tx.Amount = 9.99
	if VerifyTransaction(tx) {
		t.Fatalf("VerifyTransaction succeeded after tampering (expected failure)")
	}
	// ensure public key matches generated pub
	if string(pub) != string(tx.PublicKey) {
		t.Fatalf("public key mismatch")
	}
}
