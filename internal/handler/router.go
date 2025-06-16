package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

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

	// WebSocket endpoint - delegated to ws-handler
	mux.Get("/join-auction", app.JoinAuction)

	// API routes - delegated to http-handler
	mux.Route("/api/v1", func(r chi.Router) {
		// Stats endpoint
		r.Get("/stats", app.GetStats)

		// Authentication routes
		r.Post("/register", app.RegisterUser)
		r.Post("/login", app.LoginUser)
		r.Get("/auth/google/login", app.GoogleLoginHandler)
		r.Get("/auth/google/callback", app.GoogleCallbackHandler)

		// Protected routes
		r.With(app.AuthMiddleware).Get("/about", app.AboutUser)
		r.With(app.AuthMiddleware).Get("/categories", app.GetCategories)

		// Auction routes
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
