package store

import (
	"context"

	"github.com/LikhithMar14/Bidzy/internal/models"
)


type Store struct {
	Auction AuctionRepository
}

type AuctionRepository interface {
	CreateAuction(ctx context.Context, auction *models.AuctionRequest) error
}