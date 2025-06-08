package main

import (
	"github.com/LikhithMar14/BidZy/internal/handler"
	"github.com/LikhithMar14/BidZy/internal/service/auction"
	"github.com/LikhithMar14/BidZy/internal/migrations"
	db "github.com/LikhithMar14/BidZy/internal/store/database"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

const Version = "1.0.0"

func main() {

	logger := zap.Must(zap.NewProduction()).Sugar()

	err := godotenv.Load()
	if err != nil {
		logger.Fatalw("failed to load environment variables", "error", err)
	}
	hubManager := auction.NewHubManager()


	cfg := handler.Load()



	defer logger.Sync()

	database, err := db.Open(cfg.Db.Addr, cfg.Db.MaxOpenConns, cfg.Db.MaxIdleConns, cfg.Db.MaxLifetime.String())

	if err != nil {
		logger.Fatalw("failed to open database", "error", err)
	}

	defer database.Close()

	err = db.MigrateFS(database, migrations.FS, ".")

	if err != nil {
		logger.Fatalw("failed to migrate database", "error", err)
	}

	logger.Infow("database connection pool established and migrated successfully", "addr", cfg.Db.Addr, "maxOpenConns", cfg.Db.MaxOpenConns, "maxIdleConns", cfg.Db.MaxIdleConns, "maxLifetime", cfg.Db.MaxLifetime.String())

	app := handler.NewApplication(cfg, Version, logger,hubManager)

	mux := app.Routes()

	logger.Fatal(app.Server(mux))

}
