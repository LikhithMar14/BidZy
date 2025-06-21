package handler

import (
	"encoding/json"
	"net/http"
	"time"

	ratelimitmw "github.com/LikhithMar14/BidZy/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (app *Application) Routes() *chi.Mux {
	mux := chi.NewRouter()

	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-User-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Max age for preflight cache
	}))

	// Global middlewares
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.StripSlashes)
	mux.Use(middleware.Compress(5))
	mux.Use(middleware.RealIP)
	mux.Use(middleware.RequestID)
	mux.Use(middleware.Timeout(60 * time.Second))

	// Request size limit (1MB)
	mux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
			next.ServeHTTP(w, r)
		})
	})
	rateLimiter := ratelimitmw.NewHybridRateLimiter(10.0, 50, app.Rdb)
	rateLimiter.SetDebug(true)

	mux.Use(rateLimiter.Middleware)

	// Health check
	mux.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now(),
			"version":   app.Version,
		})
	})

	// WebSocket join (public)
	mux.Get("/join-auction", app.JoinAuction)

	// S3 Upload route (public)
	mux.Post("/api/v1/upload", app.Uploader.HandleGetPresignedURL)

	// API v1 routes
	mux.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Post("/register", app.RegisterUser)
		r.Post("/login", app.LoginUser)
		r.Get("/auth/google/login", app.GoogleLoginHandler)
		r.Get("/auth/google/callback", app.GoogleCallbackHandler)

		// 🔐 Authenticated routes
		r.With(ratelimitmw.AuthMiddleware(app.Config.JwtSecret)).Group(func(protected chi.Router) {
			// User info
			protected.Get("/about", app.AboutUser)
			protected.Get("/categories", app.GetCategories)
			protected.Get("/stats", app.GetStats)

			// Auction Routes
			protected.Route("/auctions", func(ra chi.Router) {
				ra.Get("/", app.ListAuctions)
				ra.Post("/", app.CreateAuction)
				ra.Get("/{auctionId}/clients", app.GetAuctionClients)
				ra.Get("/{auctionId}/bids", app.GetAuctionBids)
				ra.Delete("/{auctionId}", app.DeleteAuction)
				ra.Get("/user/{userId}", app.GetAuctionByUserID)
				ra.Get("/{auctionId}", app.GetAuctionByID)
			})

			// Image Upload Routes
			protected.Route("/upload", func(ru chi.Router) {
				ru.Post("/auction-image", app.GenerateAuctionImageUploadURL)
			})

			// User Routes
			protected.Route("/users", func(ru chi.Router) {
				ru.Get("/", app.GetUserByID)
				ru.Get("/auctions", app.GetAuctionsByUserID)
				ru.Get("/bids", app.GetBidsByUserID)
				ru.Get("/stats", app.GetOwnStats)
				ru.Get("/stats/{userId}", app.GetUserStatsByID)
				ru.Get("/participated-auctions", app.GetParticipatedAuctions)
			})

			// Bid Routes
			protected.Route("/bids", func(rb chi.Router) {
				rb.Get("/{auctionId}/timeline", app.GetBidTimelineByAuctionID)
			})
		})
	})

	return mux
}
