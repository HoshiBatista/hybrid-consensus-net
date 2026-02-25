package main

import (
	"flag"
	"fmt"
	"go-hybrid-blockchain/internal/blockchain"
	"go-hybrid-blockchain/internal/network"
	"strings"
)

func main() {
	port := flag.String("port", "9000", "Port for P2P server")
	peersStr := flag.String("peers", "", "Comma separated list of peers")
	dbFile := flag.String("db", "blockchain.db", "Database file name")
	flag.Parse()

	var peers []string
	if *peersStr != "" {
		peers = strings.Split(*peersStr, ",")
	}

	fmt.Printf("Starting Node on port %s, db: %s\n", *port, *dbFile)

	bc := blockchain.CreateBlockchain(*dbFile)

	p2pServer := network.NewServer(*port, peers, bc)
	go p2pServer.Start()

	select {}
}