package handler

import (
	"encoding/json"
	"net/http"
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
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	})

	mux.Get("/join-auction", app.JoinAuction)

	return mux
}

func (app *Application) JoinAuction(w http.ResponseWriter, r *http.Request) {
	auctionId := r.URL.Query().Get("auctionId")
	senderId := r.URL.Query().Get("senderId")

	if auctionId == "" || senderId == "" {
		http.Error(w, "auctionId and senderId are required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		app.Logger.Errorw("WebSocket upgrade failed", "error", err)
		http.Error(w, "Failed to upgrade to websocket", http.StatusInternalServerError)
		return
	}


	hub := app.HubManager.GetOrCreateHub(auctionId)

	client := &auction.Client{
		ID:   senderId,
		Hub:  hub,
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	hub.Register <- client

	go client.WritePump()
	go client.ReadPump()

}
