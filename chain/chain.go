package chain

import (
	"strings"
	"time"
	"toy-blockchain/block"
)

type Blockchain struct {
	Blocks []block.Block
}

func NewBlockchain() *Blockchain {
	genesisBlock := block.Block{
		Index: 0, Timestamp: time.Now().Unix(), Transactions: []string{"Genesis Block"}, PrevHash: "0",
	}
	genesisBlock.Hash = genesisBlock.CalculateHash()
	return &Blockchain{Blocks: []block.Block{genesisBlock}}
}

func (bc *Blockchain) MineBlock(transactions []string, difficulty int) {
	newBlock := block.Block{
		Index: len(bc.Blocks), Timestamp: time.Now().Unix(), Transactions: transactions, PrevHash: bc.Blocks[len(bc.Blocks)-1].Hash,
	}

	target := strings.Repeat("0", difficulty)
	for {
		newBlock.Hash = newBlock.CalculateHash()
		if strings.HasPrefix(newBlock.Hash, target) {
			break
		}
		newBlock.Nonce++
	}
	bc.Blocks = append(bc.Blocks, newBlock)
}
