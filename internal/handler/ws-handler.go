package handler

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"net/http"

	"github.com/LikhithMar14/BidZy/internal/service/auction"
	auction_ws "github.com/LikhithMar14/BidZy/internal/service/auction/auction-ws"
	"github.com/LikhithMar14/BidZy/pkg/types"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// JoinAuction handles WebSocket connections for auction participation
func (app *Application) JoinAuction(w http.ResponseWriter, r *http.Request) {
	fmt.Println("I AM IN JOIN AUCTION")

	auctionId := strings.TrimSpace(r.URL.Query().Get("auctionId"))
	tokenString := strings.TrimSpace(r.URL.Query().Get("token"))

	if auctionId == "" || tokenString == "" {
		app.Logger.Warnw("Missing required parameters", "auctionId", auctionId, "token", "[redacted]")
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Validate JWT token
	token, err := jwt.ParseWithClaims(tokenString, &types.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(app.Config.JwtSecret), nil
	})
	if err != nil || !token.Valid {
		app.Logger.Warnw("Invalid token", "error", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	claims, ok := token.Claims.(*types.UserClaims)
	if !ok || claims == nil {
		app.Logger.Warnw("Invalid claims")
		http.Error(w, "Invalid claims", http.StatusUnauthorized)
		return
	}

	senderId := claims.UserID

	// Basic input validation
	if len(auctionId) > 50 || len(senderId) > 50 {
		app.Logger.Warnw("Parameter length too long", "auctionId", auctionId, "senderId", senderId)
		http.Error(w, "Parameter length too long", http.StatusBadRequest)
		return
	}

	// Try to get hub, or lazily initialize if it doesn't exist
	hub := app.HubManager.GetHub(auctionId)
	if hub == nil {
		fmt.Println("==== Auction not found ====")

		fmt.Println("==== Recovering from Server Crash ====")
		auctionData, err := app.Service.AuctionService.GetAuctionByID(r.Context(), auctionId)
		if err != nil || auctionData.Status != "ACTIVE" || time.Now().After(auctionData.EndTime) {
			app.Logger.Errorw("Auction not found or not active", "auctionId", auctionId, "error", err)
			http.Error(w, "Auction not found or not active", http.StatusBadRequest)
			return
		}
		fmt.Println("==== Auction data ====")
		app.Logger.Infow("Auction found", "auctionId", auctionId, "auctionData", auctionData)

		hub = app.HubManager.GetOrCreateHub(
			auctionId,
			auctionData.Increment,
			auctionData.Title,
			auctionData.Description,
			int(auctionData.CurrentPrice),
			auctionData.StartTime,
			auctionData.EndTime,
			time.Duration(auctionData.EndTime.Sub(auctionData.StartTime).Hours())*time.Hour,
			app.Service.AuctionService.(*auction.AuctionHTTP),
		)

		// Schedule auction end handling
		go func() {
			time.Sleep(time.Until(auctionData.EndTime))
			app.HubManager.HandleAuctionEnd(auctionId)
		}()
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		app.Logger.Errorw("WebSocket upgrade failed", "error", err)
		return
	}

	// Create WebSocket client
	client := &auction_ws.Client{
		ID:   senderId,
		Hub:  hub,
		Conn: conn,
		Send: make(chan []byte, 256),
		Rdb:  app.Rdb,
		UserName: claims.UserName,
	}

	app.Logger.Infow("New WebSocket connection", "auctionId", auctionId, "senderId", senderId)

	// Register client to hub with timeout
	select {
	case hub.Register <- client:
		// Start client goroutines for handling WebSocket communication
		go client.WritePump()
		go client.ReadPump()
	case <-time.After(5 * time.Second):
		app.Logger.Errorw("Timeout registering client", "auctionId", auctionId)
		conn.Close()
	}
}

// GetAuctionData returns current auction state (could be moved to http-handler if needed)
func (app *Application) GetAuctionData(w http.ResponseWriter, r *http.Request) {
	auctionId := r.URL.Query().Get("auctionId")
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
