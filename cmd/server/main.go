package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LikhithMar14/BidZy/internal/handler"
	"github.com/LikhithMar14/BidZy/internal/migrations"
	"github.com/LikhithMar14/BidZy/internal/scheduler"
	auction_ws "github.com/LikhithMar14/BidZy/internal/service/auction/auction-ws"
	"github.com/LikhithMar14/BidZy/internal/service/mail"
	"github.com/LikhithMar14/BidZy/internal/store"
	db "github.com/LikhithMar14/BidZy/internal/store/database"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const Version = "1.0.0"

func main() {
	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()

	err := godotenv.Load()
	if err != nil {
		logger.Warnw("failed to load .env file, using system environment variables", "error", err)
	}

	// Debug: Log key environment variables
	logger.Infow("Environment check",
		"DB_ADDR", os.Getenv("DB_ADDR"),
		"JWT_SECRET", len(os.Getenv("JWT_SECRET")),
		"PORT", os.Getenv("PORT"),
	)

	smtpCfg, err := mail.Load()
	if err != nil {
		logger.Fatalw("failed to load smtp config", "error", err)
	}

	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	googleRedirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
	googleOauthClient := handler.InitGoogleOauthConfig(googleClientID, googleClientSecret, googleRedirectURL)

	cfg := handler.Load()

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	hubManager := auction_ws.NewHubManager()

	database, err := db.Open(cfg.Db.Addr, cfg.Db.MaxOpenConns, cfg.Db.MaxIdleConns, cfg.Db.MaxLifetime.String())
	if err != nil {
		logger.Fatalw("failed to open database", "error", err)
	}
	defer database.Close()

	store := store.NewStorage(database)

	if err := db.MigrateFS(database, migrations.FS, "."); err != nil {
		logger.Fatalw("failed to migrate database", "error", err)
	}

	mailer := mail.NewMailService(smtpCfg, store.Auction, store.Auth, store.Bid)
	auctionScheduler := scheduler.NewAuctionScheduler(store.Auction, mailer)
	auctionScheduler.Start()

	app := handler.NewApplication(cfg, Version, logger, hubManager, rdb, store, googleOauthClient, smtpCfg)
	mux := app.Routes()

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: mux,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)

		<-sigint
		logger.Infow("shutting down gracefully...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Errorw("server shutdown failed", "error", err)
		}

		auctionScheduler.Stop()
		hubManager.Stop()

		close(idleConnsClosed)
	}()

	logger.Infow("starting server", "addr", cfg.Addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		logger.Fatalw("server error", "error", err)
	}

	<-idleConnsClosed
	logger.Infow("server stopped")
}
