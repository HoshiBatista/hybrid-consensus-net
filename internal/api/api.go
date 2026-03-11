package api

import (
	"encoding/json"
	"fmt"
	"go-hybrid-blockchain/internal/blockchain"
	"go-hybrid-blockchain/internal/network"
	"net/http"
)

type API struct {
	BC  *blockchain.Blockchain
	P2P *network.Server
}

func StartServer(port string, bc *blockchain.Blockchain, p2p *network.Server) {
	nodeAPI := &API{bc, p2p}

	http.HandleFunc("/chain", nodeAPI.getChain)
	http.HandleFunc("/mine", nodeAPI.mine)

	http.HandleFunc("/mine/pos", nodeAPI.minePoS)

	http.ListenAndServe(":"+port, nil)
}

func (api *API) getChain(w http.ResponseWriter, r *http.Request) {
	blocks := api.BC.GetAllBlocks()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blocks)
}

func (api *API) mine(w http.ResponseWriter, r *http.Request) {
	// Создаем блок с пустыми транзакциями (для примера)
	// В MineBlock должна быть логика из пакета consensus (PoW)
	newBlock := api.BC.MineBlock([]*blockchain.Transaction{})
	
	// Рассылаем новый блок всем пирам
	api.P2P.Broadcast(network.TypeBlock, newBlock)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newBlock)
}

// И сам метод:
func (api *API) minePoS(w http.ResponseWriter, r *http.Request) {
    fmt.Println("API: PoS Mining requested...")
    
    // В реальном PoS тут проверяется стейк, но для демо мы просто передаем имя узла
    validatorName := "Node_Validator_" + api.P2P.Port
    
    newBlock := api.BC.MineBlockPoS([]*blockchain.Transaction{}, validatorName)
    
    // Рассылаем блок по сети
    api.P2P.Broadcast(network.TypeBlock, newBlock)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(newBlock)
}