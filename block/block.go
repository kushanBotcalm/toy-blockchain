package block

import (
	"crypto/sha256"
	"fmt"
	"strconv"
)

type Block struct {
	Index        int
	Timestamp    int64
	Transactions []string
	PrevHash     string
	Nonce        int
	Hash         string
}

func (b *Block) CalculateHash() string {
	record := strconv.Itoa(b.Index) + strconv.FormatInt(b.Timestamp, 10) + fmt.Sprint(b.Transactions) + b.PrevHash + strconv.Itoa(b.Nonce)
	h := sha256.New()
	h.Write([]byte(record))
	return fmt.Sprintf("%x", h.Sum(nil))
}
