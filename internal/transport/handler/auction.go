package handler

import (
	"encoding/json"
	"net/http"

	"github.com/LikhithMar14/Bidzy/internal/models"
	"github.com/LikhithMar14/Bidzy/internal/service"
)


type AuctionHandler struct {
	auctionService service.AuctionServiceRepository
}
func NewAuctionHandler(auctionService service.AuctionServiceRepository) *AuctionHandler {
	return &AuctionHandler{
		auctionService: auctionService,
	}
}

func (h *AuctionHandler) CreateAuction(w http.ResponseWriter, r *http.Request) {
	var auction models.AuctionRequest
	if err := json.NewDecoder(r.Body).Decode(&auction); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := validate.Struct(auction); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := h.auctionService.CreateAuction(r.Context(), &auction); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(auction)
	
}