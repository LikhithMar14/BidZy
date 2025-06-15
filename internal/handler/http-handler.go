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
)

var oauthStateString, _ = utils.GenerateOAuthState(12)


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

	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	user := ctx.Value(types.UserContextKey)
	log.Println(user)
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    user,
		"message": "User about",
	})
}

func (app *Application) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := app.HubManager.GetStats()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(stats))
}

func (app *Application) ListAuctions(w http.ResponseWriter, r *http.Request) {
	auctions, err := app.Service.AuctionService.GetAllAuctions(r.Context())
	log.Println("error", err)

	if err != nil {
		http.Error(w, "Failed to get auctions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(auctions); err != nil {
		app.Logger.Errorw("Failed to encode auction list", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
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
		"success": true, 
		"data":    auction, 
		"message": "Auction fetched successfully",
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
		"success": true, 
		"data":    auctions, 
		"message": "Auctions fetched successfully",
	})
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

	createdAuction, err := app.Service.AuctionService.CreateAuction(ctx, &req, req.CategoryIDs, req.UserID)
	if err != nil {
		log.Printf("Failed to persist auction: %v", err)
		http.Error(w, "Failed to create auction", http.StatusInternalServerError)
		return
	}

	fmt.Println("AUCTION FROM HANDLER:", createdAuction)
	fmt.Println("++Auction ID:++", createdAuction.AuctionID)

	hub := app.HubManager.CreateHub(
		createdAuction.AuctionID,
		req.Title,
		req.Description,
		req.StartingPrice,
		req.Increment,
		req.StartDateTime,
		req.EndDateTime,
		time.Duration(req.Duration)*time.Hour,
		app.Service.AuctionService.(*auction.AuctionHTTP),
	)
	if hub == nil {
		http.Error(w, "Failed to create auction hub", http.StatusInternalServerError)
		return
	}

	log.Println("AUCTION FROM HANDLER:", createdAuction)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"auctionId": createdAuction.AuctionID,
		"message":   "Auction created and published to WS successfully",
		"data":      createdAuction,
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

func (app *Application) GetCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := app.Service.CategoryService.GetAllCategories(r.Context())
	if err != nil {
		http.Error(w, "Failed to get categories", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true, 
		"data":    categories, 
		"message": "Categories fetched successfully",
	})
}