// Package api provides a REST interface to the blockchain node.
// It exposes endpoints for submitting transactions, querying the chain,
// checking balances, triggering mining, and switching consensus at runtime.
package api

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"go-hybrid-blockchain/internal/blockchain"
	"go-hybrid-blockchain/internal/consensus"
)

// Server wires the HTTP router to the node's core components.
type Server struct {
	router   *gin.Engine
	bc       *blockchain.Blockchain
	state    *blockchain.State
	mempool  *blockchain.Mempool
	mu       sync.RWMutex   // protects cs and powDiff
	cs       consensus.Consensus
	powDiff  int
}

// NewServer constructs and configures the Gin router. Call Run to start listening.
func NewServer(
	bc *blockchain.Blockchain,
	state *blockchain.State,
	mempool *blockchain.Mempool,
	cs consensus.Consensus,
	powDiff int,
) *Server {
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})
	s := &Server{
		router:  r,
		bc:      bc,
		state:   state,
		mempool: mempool,
		cs:      cs,
		powDiff: powDiff,
	}
	s.registerRoutes()
	return s
}

// Run starts the HTTP server on addr (e.g. ":8080").
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

func (s *Server) registerRoutes() {
	s.router.GET("/status", s.getStatus)
	s.router.POST("/transactions", s.submitTransaction)
	s.router.GET("/blocks", s.getBlocks)
	s.router.GET("/balance/:address", s.getBalance)
	s.router.POST("/mine", s.mineBlock)
	s.router.GET("/mempool", s.getMempool)
	s.router.POST("/consensus", s.switchConsensus)
	s.router.POST("/faucet", s.faucet)
	s.router.Static("/static", "./frontend")
	s.router.GET("/", func(c *gin.Context) {
		c.File("./frontend/index.html")
	})
}

// GET /status
// Returns node metadata: consensus type, chain height, pending txs.
func (s *Server) getStatus(c *gin.Context) {
	s.mu.RLock()
	csType := s.cs.Type()
	diff := s.powDiff
	s.mu.RUnlock()

	blocks := s.bc.GetAllBlocks()
	height := 0
	if len(blocks) > 0 {
		height = blocks[len(blocks)-1].Height
	}

	c.JSON(http.StatusOK, gin.H{
		"consensus":   csType,
		"difficulty":  diff,
		"height":      height,
		"blocks":      len(blocks),
		"pending_txs": s.mempool.Size(),
	})
}

// switchConsensusRequest is the JSON body for POST /consensus.
type switchConsensusRequest struct {
	Type       string `json:"type"       binding:"required"` // "pow" or "pos"
	Difficulty int    `json:"difficulty"`                    // only for pow; 0 → keep current
}

// POST /consensus
// Switches the active consensus mechanism at runtime.
// For PoS, at least one account with a positive balance must exist in State.
func (s *Server) switchConsensus(c *gin.Context) {
	var req switchConsensusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	switch req.Type {
	case "pow":
		if req.Difficulty > 0 {
			s.powDiff = req.Difficulty
		}
		s.cs = consensus.NewProofOfWork(s.powDiff)
	case "pos":
		if s.state.TotalStake() == 0 {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": "PoS requires at least one account with balance — use POST /faucet first",
			})
			return
		}
		s.cs = consensus.NewProofOfStake(s.state)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown consensus type: use \"pow\" or \"pos\""})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"consensus":  s.cs.Type(),
		"difficulty": s.powDiff,
	})
}

// faucetRequest is the JSON body for POST /faucet.
type faucetRequest struct {
	Address string `json:"address" binding:"required"`
	Amount  uint64 `json:"amount"  binding:"required"`
}

// POST /faucet
// Credits tokens to an address — intended for testing only (PoS validator setup).
func (s *Server) faucet(c *gin.Context) {
	var req faucetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Amount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be > 0"})
		return
	}

	s.state.Credit(req.Address, req.Amount)
	c.JSON(http.StatusOK, gin.H{
		"address": req.Address,
		"balance": s.state.Balance(req.Address),
	})
}

// submitTransactionRequest is the JSON body expected by POST /transactions.
type submitTransactionRequest struct {
	Sender    string `json:"sender"    binding:"required"`
	Recipient string `json:"recipient" binding:"required"`
	Amount    int    `json:"amount"    binding:"required,min=1"`
	Signature string `json:"signature" binding:"required"`
}

// POST /transactions
func (s *Server) submitTransaction(c *gin.Context) {
	var req submitTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sender, err := hex.DecodeString(req.Sender)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid sender hex: %v", err)})
		return
	}
	recipient, err := hex.DecodeString(req.Recipient)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid recipient hex: %v", err)})
		return
	}
	sig, err := hex.DecodeString(req.Signature)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid signature hex: %v", err)})
		return
	}

	tx := blockchain.NewTransaction(sender, recipient, req.Amount)
	tx.Signature = sig

	if err := s.mempool.Add(tx, s.state); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"tx_id":   fmt.Sprintf("%x", tx.TxID),
		"pending": s.mempool.Size(),
	})
}

// GET /blocks
func (s *Server) getBlocks(c *gin.Context) {
	c.JSON(http.StatusOK, s.bc.GetAllBlocks())
}

// GET /balance/:address
func (s *Server) getBalance(c *gin.Context) {
	address := c.Param("address")
	c.JSON(http.StatusOK, gin.H{
		"address": address,
		"balance": s.state.Balance(address),
	})
}

// GET /mempool
func (s *Server) getMempool(c *gin.Context) {
	txs := s.mempool.Pending()
	c.JSON(http.StatusOK, gin.H{
		"count":        len(txs),
		"transactions": txs,
	})
}

// POST /mine
func (s *Server) mineBlock(c *gin.Context) {
	txs := s.mempool.Flush(0)

	height := len(s.bc.GetAllBlocks())
	block := blockchain.NewBlock(txs, s.bc.Tip, height)

	s.mu.RLock()
	cs := s.cs
	s.mu.RUnlock()

	if err := cs.MineBlock(block); err != nil {
		for _, tx := range txs {
			_ = s.mempool.Add(tx, s.state)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.bc.AddBlock(block)

	for _, tx := range txs {
		sender := fmt.Sprintf("%x", tx.Sender)
		recipient := fmt.Sprintf("%x", tx.Recipient)
		if err := s.state.Debit(sender, uint64(tx.Amount)); err != nil {
			_ = err
		}
		s.state.Credit(recipient, uint64(tx.Amount))
	}

	c.JSON(http.StatusOK, block)
}
