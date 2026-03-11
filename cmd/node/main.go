package main

import (
	"flag"
	"fmt"
	"go-hybrid-blockchain/internal/api"
	"go-hybrid-blockchain/internal/blockchain"
	"go-hybrid-blockchain/internal/network"
	"strings"
)

func main() {
	port := flag.String("port", "9000", "Port for P2P server")
	apiPort := flag.String("api", "8080", "Port for HTTP API") 
	peersStr := flag.String("peers", "", "Comma separated list of peers")
	dbFile := flag.String("db", "blockchain.db", "Database file name")
	flag.Parse()

	var peers []string
	if *peersStr != "" {
		peers = strings.Split(*peersStr, ",")
	}

	bc := blockchain.CreateBlockchain(*dbFile)
	p2pServer := network.NewServer(*port, peers, bc)
	
	go p2pServer.Start()

	fmt.Printf("HTTP API started on port %s\n", *apiPort)
	api.StartServer(*apiPort, bc, p2pServer)
}