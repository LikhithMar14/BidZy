package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LikhithMar14/BidZy/internal/service/auction"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (app *Application) Routes() *chi.Mux {
	mux := chi.NewRouter()

	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.StripSlashes)
	mux.Use(middleware.Compress(5))
	mux.Use(middleware.RealIP)
	mux.Use(middleware.RequestID)
	mux.Use(middleware.Timeout(60 * time.Second))

	mux.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now(),
			"version":   app.Version,
		})
	})

	mux.Get("/join-auction", app.JoinAuction)

	mux.Route("/api/v1", func(r chi.Router) {
		r.Get("/stats", app.GetStats)


		r.Route("/auctions", func(r chi.Router) {
			r.Get("/", app.ListAuctions)
			r.Get("/{auctionId}", app.GetAuctionData)
			r.Get("/{auctionId}/clients", app.GetAuctionClients)
			r.Get("/{auctionId}/bids", app.GetAuctionBids)
			r.Post("/{auctionId}/create", app.CreateAuction)
			r.Delete("/{auctionId}", app.DeleteAuction)
		})
	})

	return mux
}

func (app *Application) JoinAuction(w http.ResponseWriter, r *http.Request) {
	auctionId := strings.TrimSpace(r.URL.Query().Get("auctionId"))
	senderId := strings.TrimSpace(r.URL.Query().Get("senderId"))
	


	if auctionId == "" || senderId == "" {
		app.Logger.Warnw("Missing required parameters",
			"auctionId", auctionId,
			"senderId", senderId,
			"remoteAddr", r.RemoteAddr)
		http.Error(w, "auctionId and senderId are required", http.StatusBadRequest)
		return
	}


	if len(auctionId) > 50 || len(senderId) > 50 {
		app.Logger.Warnw("Parameter length validation failed",
			"auctionId", auctionId,
			"senderId", senderId,
			"remoteAddr", r.RemoteAddr)
		http.Error(w, "Parameter length exceeds maximum allowed", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		app.Logger.Errorw("WebSocket upgrade failed",
			"error", err,
			"auctionId", auctionId,
			"senderId", senderId,
			"remoteAddr", r.RemoteAddr)
		http.Error(w, "Failed to upgrade to websocket", http.StatusInternalServerError)
		return
	}


	increment := 100.0
	if incrementStr := r.URL.Query().Get("increment"); incrementStr != "" {
		if parsed, err := strconv.ParseFloat(incrementStr, 64); err == nil && parsed > 0 {
			increment = parsed
		}
	}

	hub := app.HubManager.GetOrCreateHub(auctionId, increment)
	if hub == nil {
		app.Logger.Errorw("Failed to create or get hub",
			"auctionId", auctionId,
			"senderId", senderId)
		conn.Close()
		return
	}

	client := &auction.Client{
		ID:   senderId,
		Hub:  hub,
		Conn: conn,
		Send: make(chan []byte, 256),
		Rdb:  app.Rdb,
	}

	app.Logger.Infow("New WebSocket connection",
		"auctionId", auctionId,
		"senderId", senderId,
		"remoteAddr", r.RemoteAddr)


	select {
	case hub.Register <- client:
		go client.WritePump()
		go client.ReadPump()
	case <-time.After(5 * time.Second):
		app.Logger.Errorw("Timeout registering client",
			"auctionId", auctionId,
			"senderId", senderId)
		conn.Close()
	}
}

func (app *Application) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := app.HubManager.GetStats()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(stats))
}

func (app *Application) ListAuctions(w http.ResponseWriter, r *http.Request) {
	stats := app.HubManager.GetAllAuctionData()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		app.Logger.Errorw("Failed to encode auction list", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (app *Application) GetAuctionData(w http.ResponseWriter, r *http.Request) {
	auctionId := chi.URLParam(r, "auctionId")
	if auctionId == "" {
		http.Error(w, "Auction ID is required", http.StatusBadRequest)
		return
	}

	hub := app.HubManager.GetHub(auctionId)
	if hub == nil {
		http.Error(w, "Auction not found", http.StatusNotFound)
		return
	}

	auctionData := hub.GetAuctionData()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(auctionData); err != nil {
		app.Logger.Errorw("Failed to encode auction data", "error", err, "auctionId", auctionId)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (app *Application) GetAuctionClients(w http.ResponseWriter, r *http.Request) {
	auctionId := chi.URLParam(r, "auctionId")
	if auctionId == "" {
		http.Error(w, "Auction ID is required", http.StatusBadRequest)
		return
	}

	hub := app.HubManager.GetHub(auctionId)
	if hub == nil {
		http.Error(w, "Auction not found", http.StatusNotFound)
		return
	}

	clientInfo := hub.GetClientInfo()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"auctionId":   auctionId,
		"clientCount": len(clientInfo),
		"clients":     clientInfo,
		"timestamp":   time.Now(),
	}); err != nil {
		app.Logger.Errorw("Failed to encode client info", "error", err, "auctionId", auctionId)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (app *Application) GetAuctionBids(w http.ResponseWriter, r *http.Request) {
	auctionId := chi.URLParam(r, "auctionId")
	if auctionId == "" {
		http.Error(w, "Auction ID is required", http.StatusBadRequest)
		return
	}

	hub := app.HubManager.GetHub(auctionId)
	if hub == nil {
		http.Error(w, "Auction not found", http.StatusNotFound)
		return
	}

	bidHistory := hub.GetBidHistory()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"auctionId":  auctionId,
		"bidCount":   len(bidHistory),
		"bids":       bidHistory,
		"highestBid": hub.HighestBid,
		"timestamp":  time.Now(),
	}); err != nil {
		app.Logger.Errorw("Failed to encode bid history", "error", err, "auctionId", auctionId)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (app *Application) CreateAuction(w http.ResponseWriter, r *http.Request) {
	auctionId := chi.URLParam(r, "auctionId")
	if auctionId == "" {
		http.Error(w, "Auction ID is required", http.StatusBadRequest)
		return
	}


	var config struct {
		Title         string  `json:"title"`
		Description   string  `json:"description"`
		StartingPrice float64 `json:"startingPrice"`
		Increment     float64 `json:"increment"`
		Duration      int     `json:"durationHours"` 
	}

	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}


	if config.Increment <= 0 {
		config.Increment = 100.0
	}
	if config.Duration <= 0 {
		config.Duration = 1 
	}


	if app.HubManager.GetHub(auctionId) != nil {
		http.Error(w, "Auction already exists", http.StatusConflict)
		return
	}

	hub := app.HubManager.CreateHub(auctionId, config.Title, config.Description, config.StartingPrice, config.Increment, time.Duration(config.Duration)*time.Hour)
	if hub == nil {
		http.Error(w, "Failed to create auction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"auctionId": auctionId,
		"message":   "Auction created successfully",
		"data":      hub.GetAuctionData(),
	})
}

func (app *Application) DeleteAuction(w http.ResponseWriter, r *http.Request) {
	auctionId := chi.URLParam(r, "auctionId")
	if auctionId == "" {
		http.Error(w, "Auction ID is required", http.StatusBadRequest)
		return
	}

	if app.HubManager.GetHub(auctionId) == nil {
		http.Error(w, "Auction not found", http.StatusNotFound)
		return
	}

	app.HubManager.DeleteHub(auctionId)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"auctionId": auctionId,
		"message":   "Auction deleted successfully",
	})
}
