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

	for _, peer := range s.KnownPeers {
		go s.ConnectToPeer(peer)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}

		s.mu.Lock()
		s.connections = append(s.connections, conn)
		s.mu.Unlock()

		fmt.Printf("New peer connected: %s\n", conn.RemoteAddr().String())
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

func (s *Server) SendGetChain(conn net.Conn) {
	msg := Message{
		Type:    TypeGetChain,
		Payload: nil, // Для запроса полезная нагрузка не нужна
	}
	
	msgBytes, _ := json.Marshal(msg)
	conn.Write(append(msgBytes, '\n'))
}

// SendFullChain отправляет всю нашу цепочку в ответ на запрос
func (s *Server) SendFullChain(conn net.Conn) {
	// Получаем блоки из БД
	blocks := s.Blockchain.GetAllBlocks()
	
	// Сериализуем список блоков в JSON-байты
	payload, err := json.Marshal(blocks)
	if err != nil {
		log.Println("Error marshaling blocks for SendFullChain:", err)
		return
	}

	msg := Message{
		Type:    TypeChain,
		Payload: payload,
	}

	msgBytes, _ := json.Marshal(msg)
	conn.Write(append(msgBytes, '\n'))
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

	case TypeChain:
		var receivedBlocks []*blockchain.Block
		if err := json.Unmarshal(msg.Payload, &receivedBlocks); err != nil {
			log.Println("Error unmarshaling chain:", err)
			return
		}
		fmt.Printf("Received chain with %d blocks\n", len(receivedBlocks))
		s.handleChainResponse(receivedBlocks)

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

func (s *Server) handleChainResponse(blocks []*blockchain.Block) {
	myBlocks := s.Blockchain.GetAllBlocks()

	if len(blocks) > len(myBlocks) {
		fmt.Printf("Syncing: Local chain %d, Received chain %d\n", len(myBlocks), len(blocks))
		for _, b := range blocks {
			s.Blockchain.AddBlock(b) // AddBlock уже обновляет ключ "l" в базе
		}
		fmt.Println("Database updated with new blocks.")
	}
}

func (s *Server) Broadcast(msgType string, payload interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Сначала маршалим сам объект (блок или транзакцию)
	payloadBytes, _ := json.Marshal(payload)

	// Создаем сообщение с RawMessage
	message := Message{
		Type:    msgType,
		Payload: payloadBytes,
	}

	msgBytes, _ := json.Marshal(message)
	msgBytes = append(msgBytes, '\n')

	for _, conn := range s.connections {
		_, err := conn.Write(msgBytes)
		if err != nil {
			log.Printf("Broadcast error: %v", err)
		}
	}
}