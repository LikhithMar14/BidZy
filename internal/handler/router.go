package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/LikhithMar14/BidZy/internal/service/auction"
	"github.com/LikhithMar14/BidZy/pkg/types"
	"github.com/LikhithMar14/BidZy/pkg/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
var oauthStateString, _ = utils.GenerateOAuthState(12)

func (app *Application) Routes() *chi.Mux {
	mux := chi.NewRouter()

	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.StripSlashes)
	mux.Use(middleware.Compress(5))
	mux.Use(middleware.RealIP)
	mux.Use(middleware.RequestID)
	mux.Use(middleware.Timeout(60 * time.Second))

	// Health check endpoint
	mux.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now(),
			"version":   app.Version,
		})
	})

	// WebSocket routes - no auth middleware for WebSocket connections
	// Token is passed as a query parameter and validated in the handler
	mux.Get("/join-auction", app.JoinAuction)

	// API routes
	mux.Route("/api/v1", func(r chi.Router) {
		r.Get("/stats", app.GetStats)
		r.Post("/register", app.RegisterUser)
		r.Post("/login", app.LoginUser)
		r.Get("/auth/google/login", app.GoogleLoginHandler)
		r.Get("/auth/google/callback", app.GoogleCallbackHandler)
		r.With(app.AuthMiddleware).Get("/about", app.AboutUser)
		r.With(app.AuthMiddleware).Get("/categories", app.GetCategories)

		r.Route("/auctions", func(r chi.Router) {
			r.Get("/", app.ListAuctions)
			r.Get("/{auctionId}/clients", app.GetAuctionClients)
			r.Get("/{auctionId}/bids", app.GetAuctionBids)
			r.With(app.AuthMiddleware).Post("/{auctionId}/create", app.CreateAuction)
			r.Delete("/{auctionId}", app.DeleteAuction)
			r.Get("/user/{userId}", app.GetAuctionByUserID)
			r.Get("/{auctionId}", app.GetAuctionByID)
		})

	})

	return mux
}

func (app *Application) RegisterUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := types.CreateUserRequest{}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request body",
			"data":    nil,
		})
		return
	}
	var userResponse *types.CreateUserResponse
	userResponse, err := app.Service.AuthService.Register(ctx, &user)
	if err != nil {
		if strings.Contains(err.Error(), "user already exists") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "User already exists",
				"data":    nil,
			})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Internal server error",
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    userResponse,
		"message": "User registered successfully",
	})
}

func (app *Application) LoginUser(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()

	user := types.LoginRequest{}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request body",
			"data":    nil,
		})
		return
	}

	var userResponse *types.LoginResponse
	userResponse, err := app.Service.AuthService.Login(ctx, &user)
	if err != nil {
		if strings.Contains(err.Error(), "invalid credentials") {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Invalid credentials",
				"data":    nil,
			})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Internal server error",
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    userResponse,
		"message": "Login successful",
	})
}

func (app *Application) GoogleLoginHandler(w http.ResponseWriter, r *http.Request) {
	url := app.Service.AuthService.GetGoogleLoginURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (app *Application) GoogleCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("state") != oauthStateString {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	userInfo, err := app.Service.AuthService.GetUserInfoFromGoogle(r.Context(), r.FormValue("code"))
	if err != nil {
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(userInfo)
}
func (app *Application) AboutUser(w http.ResponseWriter, r *http.Request) {
	log.Println("I AM IN ABOUT USER")

	log.Println("I AM IN ABOUT USER 1")
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	user := ctx.Value(types.UserContextKey)
	log.Println(user)
	log.Println("I AM IN ABOUT USER 2")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    user,
		"message": "User about",
	})

}
func (app *Application) JoinAuction(w http.ResponseWriter, r *http.Request) {
	fmt.Println("I AM IN JOIN AUCTION")

	auctionId := strings.TrimSpace(r.URL.Query().Get("auctionId"))
	tokenString := strings.TrimSpace(r.URL.Query().Get("token"))

	if auctionId == "" || tokenString == "" {
		app.Logger.Warnw("Missing required parameters", "auctionId", auctionId, "token", "[redacted]")
		return
	}

	token, err := jwt.ParseWithClaims(tokenString, &types.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(app.Config.JwtSecret), nil
	})
	if err != nil || !token.Valid {
		app.Logger.Warnw("Invalid token", "error", err)
		return
	}

	claims, ok := token.Claims.(*types.UserClaims)
	if !ok || claims == nil {
		app.Logger.Warnw("Invalid claims")
		return
	}

	senderId := claims.UserID

	if len(auctionId) > 50 || len(senderId) > 50 {
		app.Logger.Warnw("Parameter length too long", "auctionId", auctionId, "senderId", senderId)
		return
	}

	hub := app.HubManager.GetHub(auctionId)
	if hub == nil {
		app.Logger.Errorw("Auction not found", "auctionId", auctionId)
		http.Error(w, "Auction not found", http.StatusBadRequest)
		fmt.Println("Auction not found")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		app.Logger.Errorw("WebSocket upgrade failed", "error", err)
		return
	}

	client := &auction.Client{
		ID:   senderId,
		Hub:  hub,
		Conn: conn,
		Send: make(chan []byte, 256),
		Rdb:  app.Rdb,
	}

	app.Logger.Infow("New WebSocket connection", "auctionId", auctionId, "senderId", senderId)

	select {
	case hub.Register <- client:
		go client.WritePump()
		go client.ReadPump()
	case <-time.After(5 * time.Second):
		app.Logger.Errorw("Timeout registering client", "auctionId", auctionId)
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
	stats, err := app.Service.AuctionService.GetAllAuctions(r.Context())
	log.Println("error", err)

	if err != nil {
		http.Error(w, "Failed to get auctions", http.StatusInternalServerError)
		return
	}

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
	var req types.CreateAuctionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	claims, ok := r.Context().Value(types.UserContextKey).(*types.UserClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	req.UserID = claims.UserID
	log.Println("USER ID:", req.UserID)

	if req.Increment <= 0 {
		req.Increment = 100.0
	}
	if req.Duration <= 0 {
		req.Duration = 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	auction, err := app.Service.AuctionService.CreateAuction(ctx, &req, req.CategoryIDs, req.UserID)

	if err != nil {
		log.Printf("Failed to persist auction: %v", err)
		http.Error(w, "Failed to create auction", http.StatusInternalServerError)
		return
	}
	fmt.Println("AUCTION FROM HANDLER:", auction)
	fmt.Println("++Auction ID:++", auction.AuctionID)

	hub := app.HubManager.CreateHub(
		auction.AuctionID,
		req.Title,
		req.Description,
		req.StartingPrice,
		req.Increment,
		req.StartDateTime,
		req.EndDateTime,
		time.Duration(req.Duration)*time.Hour,
	)
	if hub == nil {
		http.Error(w, "Failed to create auction hub", http.StatusInternalServerError)
		return
	}

	log.Println("AUCTION FROM HANDLER:", auction)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"auctionId": auction.AuctionID,
		"message":   "Auction created and published to WS successfully",
		"data":      auction,
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

func (app *Application) GetCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := app.Service.CategoryService.GetAllCategories(r.Context())
	if err != nil {
		http.Error(w, "Failed to get categories", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true, "data": categories, "message": "Categories fetched successfully",
	})
}

func (app *Application) GetAuctionByID(w http.ResponseWriter, r *http.Request) {
	auctionId := chi.URLParam(r, "auctionId")
	if auctionId == "" {
		http.Error(w, "Auction ID is required", http.StatusBadRequest)
		return
	}

	auction, err := app.Service.AuctionService.GetAuctionByID(r.Context(), auctionId)
	fmt.Println("AUCTION FROM HANDLER:", auction)
	if err != nil {
		http.Error(w, "Failed to get auction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true, "data": auction, "message": "Auction fetched successfully",
	})
}

func (app *Application) GetAuctionByUserID(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(types.UserContextKey).(*types.UserClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userId := claims.UserID
	if userId == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	auctions, err := app.Service.AuctionService.GetAuctionsByUserID(r.Context(), userId)
	if err != nil {
		http.Error(w, "Failed to get auctions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true, "data": auctions, "message": "Auctions fetched successfully",
	})
}
