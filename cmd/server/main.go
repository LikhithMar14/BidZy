package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/LikhithMar14/BidZy/internal/handler"
	"github.com/LikhithMar14/BidZy/internal/migrations"
	"github.com/LikhithMar14/BidZy/internal/service/auction"
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

	cfg := handler.Load()

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	hubManager := auction.NewHubManager()


	database, err := db.Open(cfg.Db.Addr, cfg.Db.MaxOpenConns, cfg.Db.MaxIdleConns, cfg.Db.MaxLifetime.String())
	if err != nil {
		logger.Fatalw("failed to open database", "error", err)
	}
	defer database.Close()

	err = db.MigrateFS(database, migrations.FS, ".")
	if err != nil {
		logger.Fatalw("failed to migrate database", "error", err)
	}

	logger.Infow("database connection pool established and migrated successfully",
		"addr", cfg.Db.Addr,
		"maxOpenConns", cfg.Db.MaxOpenConns,
		"maxIdleConns", cfg.Db.MaxIdleConns,
		"maxLifetime", cfg.Db.MaxLifetime.String())

	app := handler.NewApplication(cfg, Version, logger, hubManager, rdb)

	mux := app.Routes()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Infow("received shutdown signal", "signal", sig)

		hubManager.Stop()

		if err := rdb.Close(); err != nil {
			logger.Errorw("failed to close Redis connection", "error", err)
		}

		logger.Info("graceful shutdown completed")
		os.Exit(0)
	}()

	logger.Fatal(app.Server(mux))
}
