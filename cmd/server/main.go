package main

import (
	"fmt"
	"os"

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
		logger.Fatalw("failed to load environment variables", "error", err)
	}
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
	fmt.Println(cfg.Db.Addr)

	database, err := db.Open(cfg.Db.Addr, cfg.Db.MaxOpenConns, cfg.Db.MaxIdleConns, cfg.Db.MaxLifetime.String())
	if err != nil {
		logger.Fatalw("failed to open database", "error", err)
	}
	defer database.Close()

	store := store.NewStorage(database)

	err = db.MigrateFS(database, migrations.FS, ".")
	if err != nil {
		logger.Fatalw("failed to migrate database", "error", err)
	}

	// Initialize mail service with all required repositories
	mailer := mail.NewMailService(smtpCfg, store.Auction, store.Auth, store.Bid)

	// Initialize and start the auction scheduler
	auctionScheduler := scheduler.NewAuctionScheduler(store.Auction, mailer)
	auctionScheduler.Start()

	logger.Infow("database connection pool established and migrated successfully",
		"addr", cfg.Db.Addr,
		"maxOpenConns", cfg.Db.MaxOpenConns,
		"maxIdleConns", cfg.Db.MaxIdleConns,
		"maxLifetime", cfg.Db.MaxLifetime.String())

	// Create the application and get the server
	app := handler.NewApplication(cfg, Version, logger, hubManager, rdb, store, googleOauthClient, smtpCfg)
	mux := app.Routes()

	if err := app.Server(mux); err != nil {
		logger.Errorw("server exited with error", "error", err)
	}
}
