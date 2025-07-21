package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/LikhithMar14/BidZy/internal/store"
	db "github.com/LikhithMar14/BidZy/internal/store/database"
	"github.com/joho/godotenv"
)

// Category enum mapping
const (
	ART          = 1
	COLLECTIBLES = 2
	ELECTRONICS  = 3
	FASHION      = 4
	HOME         = 5
	OTHER        = 6
)

// Category names mapping
var categoryNames = map[int]string{
	ART:          "Art",
	COLLECTIBLES: "Collectibles",
	ELECTRONICS:  "Electronics",
	FASHION:      "Fashion",
	HOME:         "Home",
	OTHER:        "Other",
}

// SeedCategories seeds the database with predefined categories
func SeedCategories() error {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: failed to load .env file, using system environment variables: %v", err)
	}

	// Get database connection string from environment
	dbAddr := getEnvOrDefault("DB_ADDR", "postgres://postgres:password@localhost:5432/auction_db?sslmode=disable")

	// Open database connection
	database, err := db.Open(dbAddr, 25, 25, "5m")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Create store instance
	storeInstance := store.NewStorage(database)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("Starting category seeding...")

	// Seed each category
	for _, name := range categoryNames {
		if err := seedCategory(ctx, storeInstance, name); err != nil {
			return fmt.Errorf("failed to seed category %s: %w", name, err)
		}
	}

	log.Println("Category seeding completed successfully!")
	return nil
}

// seedCategory seeds a single category if it doesn't exist
func seedCategory(ctx context.Context, storeInstance *store.Store, name string) error {
	// Check if category already exists
	existingCategory, err := storeInstance.Category.GetCategoryByName(ctx, name)
	if err != nil {
		return fmt.Errorf("error checking if category exists: %w", err)
	}

	if existingCategory != nil {
		log.Printf("Category '%s' already exists (ID: %s), skipping...", name, *existingCategory.ID)
		return nil
	}

	// Create the category
	createdCategory, err := storeInstance.Category.CreateCategory(ctx, name)
	if err != nil {
		return fmt.Errorf("error creating category: %w", err)
	}

	log.Printf("✅ Created category '%s' with ID: %s", name, *createdCategory.ID)
	return nil
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := getEnv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnv is a helper function to get environment variables
func getEnv(key string) string {
	return os.Getenv(key)
}

// RunSeed is the main function to run the seeding process
func RunSeed() {
	log.Println("🚀 Starting BidZy Database Seeding...")

	if err := SeedCategories(); err != nil {
		log.Fatalf("❌ Seeding failed: %v", err)
	}

	log.Println("🎉 Seeding completed successfully!")
}
