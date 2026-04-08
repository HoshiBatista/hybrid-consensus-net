package main

import (
	"flag"
	"fmt"
	"log"

	"go-hybrid-blockchain/internal/api"
	"go-hybrid-blockchain/internal/blockchain"
	"go-hybrid-blockchain/internal/consensus"
)

func main() {
	dbFile       := flag.String("db",         "blockchain.db", "Path to the bbolt database file")
	listenAddr   := flag.String("addr",       ":8080",         "HTTP API listen address")
	consensusType := flag.String("consensus", "pow",           "Consensus mechanism: pow or pos")
	powDifficulty := flag.Int("difficulty",   4,               "PoW difficulty (ignored for PoS)")
	flag.Parse()

	bc := blockchain.CreateBlockchain(*dbFile)
	defer bc.DB.Close()

	state   := blockchain.NewState()
	mempool := blockchain.NewMempool()

	var cs consensus.Consensus
	switch *consensusType {
	case "pos":
		cs = consensus.NewProofOfStake(state)
	default:
		cs = consensus.NewProofOfWork(*powDifficulty)
	}

	fmt.Printf("Node started  chain-tip=%x  consensus=%s  api=%s\n",
		bc.Tip, cs.Type(), *listenAddr)

	srv := api.NewServer(bc, state, mempool, cs, *powDifficulty)
	if err := srv.Run(*listenAddr); err != nil {
		log.Fatalf("API server error: %v", err)
	}
}
