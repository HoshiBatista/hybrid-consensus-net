package network

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"go-hybrid-blockchain/internal/blockchain"
)

type Server struct {
	Port        string
	KnownPeers  []string
	Blockchain  *blockchain.Blockchain
	mu          sync.Mutex
	connections []net.Conn
}

func NewServer(port string, peers []string, bc *blockchain.Blockchain) *Server {
	return &Server{
		Port:       port,
		KnownPeers: peers,
		Blockchain: bc,
	}
}

// Start запускает прослушивание входящих соединений
func (s *Server) Start() {
	ln, err := net.Listen("tcp", ":"+s.Port)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	fmt.Printf("P2P Server started on port %s\n", s.Port)

	// Подключаемся к уже известным пирам из конфигурации
	for _, peer := range s.KnownPeers {
		go s.ConnectToPeer(peer)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}
		go s.HandleConnection(conn)
	}
}

// ConnectToPeer подключается к другому узлу
func (s *Server) ConnectToPeer(address string) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("Could not connect to peer %s: %v\n", address, err)
		return
	}
	s.mu.Lock()
	s.connections = append(s.connections, conn)
	s.mu.Unlock()
	
	fmt.Printf("Connected to peer: %s\n", address)
	go s.HandleConnection(conn)
	
	// Сразу запрашиваем цепочку у нового знакомого для синхронизации
	s.SendGetChain(conn)
}

// HandleConnection читает сообщения из сокета
func (s *Server) HandleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		data, err := reader.ReadBytes('\n') // Каждое сообщение заканчивается новой строкой
		if err != nil {
			if err != io.EOF {
				log.Println("Read error:", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Println("JSON unmarshal error:", err)
			continue
		}

		s.ProcessMessage(msg, conn)
	}
}

// ProcessMessage обрабатывает логику в зависимости от типа сообщения
func (s *Server) ProcessMessage(msg Message, conn net.Conn) {
	switch msg.Type {
	case TypeGetChain:
		fmt.Println("Received get_chain request")
		s.SendFullChain(conn)

	case TypeBlock:
		var block blockchain.Block
		if err := json.Unmarshal(msg.Payload, &block); err != nil {
			log.Println("Error unmarshaling block:", err)
			return
		}
		fmt.Printf("Received new block: %x\n", block.Hash)
		
		s.Blockchain.AddBlock(&block)
		fmt.Println("Block added to database.")
	}
}

// Broadcast отправляет сообщение всем подключенным пирам
func (s *Server) Broadcast(msgType string, payload interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, _ := json.Marshal(payload)
	message := Message{Type: msgType, Payload: data}
	msgBytes, _ := json.Marshal(message)
	msgBytes = append(msgBytes, '\n')

	for _, conn := range s.connections {
		conn.Write(msgBytes)
	}
}

func (s *Server) SendGetChain(conn net.Conn) {
	msg := Message{Type: TypeGetChain}
	data, _ := json.Marshal(msg)
	conn.Write(append(data, '\n'))
}

func (s *Server) SendFullChain(conn net.Conn) {
	// В реальности тут нужно вычитать все блоки из BoltDB
	// blocks := s.Blockchain.GetAllBlocks() 
	// ... логика отправки ...
}