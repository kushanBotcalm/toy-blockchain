package mining

import (
	"strings"
	"time"
	"toy-blockchain/block"
)

// MineBlock performs proof-of-work to meet the difficulty target[cite: 1].
func MineBlock(b *block.Block, difficulty int) {
	target := strings.Repeat("0", difficulty)
	b.Timestamp = time.Now().Unix()

	for {
		b.Hash = b.CalculateHash()
		if strings.HasPrefix(b.Hash, target) {
			break
		}
		b.Nonce++
	}
}

// ValidatePoW checks if a block's hash satisfies the difficulty target[cite: 1].
func ValidatePoW(b block.Block, difficulty int) bool {
	target := strings.Repeat("0", difficulty)
	return strings.HasPrefix(b.Hash, target) && b.Hash == b.CalculateHash()
}
