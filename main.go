package main

import (
	"fmt"

	"toy-blockchain/blockchain"
)

func main() {

	chain := blockchain.NewBlockchain()

	fmt.Println("Blockchain created")

	fmt.Println("Number of blocks:", len(chain.Blocks))

	genesis := chain.Blocks[0]

	fmt.Println("Genesis Index:", genesis.Index)
	fmt.Println("Genesis Previous Hash:", genesis.PreviousHash)

	fmt.Printf("Full Block Details: %+v\n", genesis)

}
