package transaction

// Transaction represents a transfer of funds
// from one account to another.
type Transaction struct {
	Sender   string  `json:"sender"`
	Receiver string  `json:"receiver"`
	Amount   float64 `json:"amount"`
}
