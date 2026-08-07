package transaction

// Transaction represents a transfer of funds
// from one account to another.
type Transaction struct {
	Sender   string  `json:"sender"`
	Receiver string  `json:"receiver"`
	Amount   float64 `json:"amount"`
	// PublicKey is the ed25519 public key of the sender (base64 when JSON encoded)
	PublicKey []byte `json:"public_key,omitempty"`
	// Signature is the ed25519 signature over the transaction fields
	Signature []byte `json:"signature,omitempty"`
}
