package handlers

import (
	"encoding/json"
	"net/http"

	"chainpulse/shared/database"

	"github.com/gorilla/mux"
)

// ContractHandler handles contract-related API requests
type ContractHandler struct {
	DB *database.DB
}

// NewContractHandler creates a new contract handler
func NewContractHandler(db *database.DB) *ContractHandler {
	return &ContractHandler{
		DB: db,
	}
}

// GetContracts returns a list of contracts
func (h *ContractHandler) GetContracts(w http.ResponseWriter, r *http.Request) {
	contracts, err := h.DB.GetContracts()
	if err != nil {
		http.Error(w, "Failed to get contracts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"contracts": contracts,
		"total":     len(contracts),
	})
}

// GetContractByAddress returns a contract by its address
func (h *ContractHandler) GetContractByAddress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	contract, err := h.DB.GetContractByAddress(address)
	if err != nil {
		http.Error(w, "Failed to get contract", http.StatusInternalServerError)
		return
	}

	if contract == nil {
		http.Error(w, "Contract not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contract)
}