package store

import (
	"context"
	"database/sql"

	"github.com/LikhithMar14/Bidzy/internal/models"
)


type AuctionStorage struct {
	db *sql.DB
}

func NewAuctionStorage(db *sql.DB) *AuctionStorage {
	return &AuctionStorage {
		db:db,
	}
}


func (s *AuctionStorage) CreateAuction(ctx context.Context,auction *models.AuctionRequest) error {
	var newAuction models.Auction

	query := `INSERT INTO auctions (title, description, starting_price, start_date, end_date, categories, image, user_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`

	row := s.db.QueryRowContext(ctx, query, auction.Title, auction.Description, auction.StartingPrice, auction.StartDate, auction.EndDate, auction.Categories, auction.Image, auction.UserID)

	if err := row.Scan(&newAuction.ID); err != nil {
		return err
	}

	return nil
}

