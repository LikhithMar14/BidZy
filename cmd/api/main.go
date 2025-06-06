package main

import (
	"log"
	"os"
	"time"

	"github.com/LikhithMar14/Bidzy/internal/migrations"
	"github.com/LikhithMar14/Bidzy/internal/store/database"
	"github.com/joho/godotenv"
	"github.com/go-playground/validator/v10"
)

const Version = "1.0.0"
var validate *validator.Validate

func init() {
    validate = validator.New()
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	if os.Getenv("DATABASE_URL") == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := database.Open(os.Getenv("DATABASE_URL"), 10, 5, 30*time.Minute)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	log.Println("database opened")

	if err := database.MigrateFS(db, migrations.FS, "."); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}
}
